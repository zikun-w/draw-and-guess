package game

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"drawguess/internal/wirepb"
	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"
)

func codecStroke(n int) Stroke {
	s := Stroke{ID: "00000000-0000-4000-8000-000000000000", Color: "#292d2b", Width: 6, Tool: "pen", Points: make([]Point, n)}
	for i := range s.Points {
		s.Points[i] = Point{1000 + i%90*100, 2000 + i%100*70}
	}
	return s
}
func TestProtobufRoundTrip(t *testing.T) {
	s := codecStroke(24)
	s.Tool = "eraser"
	cmd := Command{Type: "stroke", Turn: 3, Stroke: s}
	data, err := EncodeStrokeCommand(cmd)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeProtobufCommand(data)
	if err != nil || !reflect.DeepEqual(decoded, cmd) {
		t.Fatalf("stroke round trip: %v", err)
	}
	state := State{Version: 1, ID: "TESTROOM", Phase: "drawing", Turn: 3, Seq: 10, Word: "宇航员"}
	snap := frame{Type: "snapshot", State: &state, Strokes: []Stroke{s}}
	out, err := DecodeDrawingFrame(encodeProtobufFrame(snap))
	if err != nil || !reflect.DeepEqual(out.Strokes, snap.Strokes) || !reflect.DeepEqual(out.State, &state) {
		t.Fatalf("snapshot round trip: %v", err)
	}
	if out.Seq != 10 || out.Turn != 3 {
		t.Fatal("snapshot sequence mismatch")
	}
}
func TestProtobufRejectsInvalidCommands(t *testing.T) {
	cases := []*wirepb.DrawingPacket{
		{Type: wirepb.PacketType_SNAPSHOT},
		{Type: wirepb.PacketType_STROKE, Stroke: &wirepb.StrokeChunk{Xy: []uint32{1}}},
		{Type: wirepb.PacketType_STROKE, Stroke: &wirepb.StrokeChunk{Xy: []uint32{10001, 0}}},
		{Type: wirepb.PacketType_STROKE, Stroke: &wirepb.StrokeChunk{Xy: make([]uint32, 194)}},
		{Type: wirepb.PacketType_STROKE, Stroke: strokeToProto(codecStroke(2)), StateJson: []byte(`{}`)},
	}
	for i, p := range cases {
		data, _ := proto.Marshal(p)
		if _, err := decodeProtobufCommand(data); err == nil {
			t.Errorf("accepted invalid case %d", i)
		}
	}
	if _, err := decodeProtobufCommand([]byte{0xff}); err == nil {
		t.Fatal("accepted malformed protobuf")
	}
}
func TestMixedJSONAndProtobufClients(t *testing.T) {
	hub := NewHub(nil)
	defer hub.Close()
	room, err := hub.Create("drawer", Settings{1, 30})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { id := r.URL.Query().Get("id"); hub.ServeWS(w, r, id, id) }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dial := func(id string, binary bool) *websocket.Conn {
		opts := &websocket.DialOptions{}
		if binary {
			opts.Subprotocols = []string{ProtobufProtocol}
		}
		c, _, e := websocket.Dial(ctx, server.URL+"/ws?room="+room+"&id="+id, opts)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { c.CloseNow() })
		return c
	}
	drawer, guest := dial("drawer", true), dial("guest", false)
	read := func(c *websocket.Conn, match func(Frame) bool) Frame {
		for {
			kind, data, e := c.Read(ctx)
			if e != nil {
				t.Fatal(e)
			}
			var f Frame
			if kind == websocket.MessageBinary {
				if c.Subprotocol() != ProtobufProtocol {
					t.Fatal("binary sent to legacy client")
				}
				f, e = DecodeDrawingFrame(data)
			} else {
				e = json.Unmarshal(data, &f)
			}
			if e != nil {
				t.Fatal(e)
			}
			if match(f) {
				return f
			}
		}
	}
	write := func(c *websocket.Conn, v any) {
		data, _ := json.Marshal(v)
		if err := c.Write(ctx, websocket.MessageText, data); err != nil {
			t.Fatal(err)
		}
	}
	read(drawer, func(f Frame) bool { return f.State != nil && len(f.State.Players) == 2 })
	read(guest, func(f Frame) bool { return f.Type == "snapshot" })
	write(drawer, Command{Type: "start"})
	choice := read(drawer, func(f Frame) bool { return f.State != nil && f.State.Phase == "choosing" })
	other := read(guest, func(f Frame) bool { return f.State != nil && f.State.Phase == "choosing" })
	if len(other.State.Choices) != 0 || len(choice.State.Choices) != 3 {
		t.Fatal("choice isolation failed")
	}
	write(drawer, Command{Type: "choose", Turn: 1})
	active := read(drawer, func(f Frame) bool { return f.State != nil && f.State.Phase == "drawing" })
	read(guest, func(f Frame) bool { return f.State != nil && f.State.Phase == "drawing" })
	stroke := codecStroke(3)
	data, _ := EncodeStrokeCommand(Command{Type: "stroke", Turn: 1, Stroke: stroke})
	if err = drawer.Write(ctx, websocket.MessageBinary, data); err != nil {
		t.Fatal(err)
	}
	for _, c := range []*websocket.Conn{drawer, guest} {
		f := read(c, func(f Frame) bool { return f.Type == "stroke" })
		if f.Seq != 1 || !reflect.DeepEqual(*f.Stroke, stroke) {
			t.Fatal("mixed broadcast mismatch")
		}
	}
	write(guest, Command{Type: "sync"})
	snap := read(guest, func(f Frame) bool { return f.Type == "snapshot" })
	if snap.State.Word != "" || len(snap.Strokes) != 1 {
		t.Fatal("legacy snapshot leaked answer or lost drawing")
	}
	write(drawer, Command{Type: "sync"})
	snap = read(drawer, func(f Frame) bool { return f.Type == "snapshot" })
	if snap.State.Word != active.State.Word || snap.State.Seq != 1 || !reflect.DeepEqual(snap.Strokes, []Stroke{stroke}) {
		t.Fatal("protobuf snapshot mismatch")
	}
}

func BenchmarkWire(b *testing.B) {
	for _, count := range []int{3, 24, 96} {
		cmd := Command{Type: "stroke", Turn: 1, Stroke: codecStroke(count)}
		text, _ := json.Marshal(struct {
			Type   string `json:"type"`
			Turn   int    `json:"turn"`
			Stroke Stroke `json:"stroke"`
		}{cmd.Type, cmd.Turn, cmd.Stroke})
		binary, _ := EncodeStrokeCommand(cmd)
		f := frame{Type: "stroke", Turn: 1, Seq: 1, Stroke: &cmd.Stroke}
		b.Run(fmt.Sprintf("stroke%d/json/encode", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = encode(f)
			}
		})
		b.Run(fmt.Sprintf("stroke%d/protobuf/encode", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = encodeProtobufFrame(f)
			}
		})
		b.Run(fmt.Sprintf("stroke%d/json/decode", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := decodeCommand(text); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("stroke%d/protobuf/decode", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := decodeProtobufCommand(binary); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
	for _, count := range []int{2400, 24000} {
		state := State{Version: 1, Turn: 1, Seq: 100, Phase: "drawing", Word: "宇航员"}
		f := frame{Type: "snapshot", State: &state, Strokes: []Stroke{codecStroke(count)}}
		text := encode(f)
		binary := encodeProtobufFrame(f)
		b.Run(fmt.Sprintf("snapshot%d/json/encode", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = encode(f)
			}
		})
		b.Run(fmt.Sprintf("snapshot%d/protobuf/encode", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = encodeProtobufFrame(f)
			}
		})
		b.Run(fmt.Sprintf("snapshot%d/json/decode", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var out frame
				if err := json.Unmarshal(text, &out); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("snapshot%d/protobuf/decode", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := DecodeDrawingFrame(binary); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
