package game

import (
	"encoding/json"
	"errors"

	"drawguess/internal/wirepb"
	"google.golang.org/protobuf/proto"
)

const ProtobufProtocol = "drawguess.pb.v1"

func strokeToProto(s Stroke) *wirepb.StrokeChunk {
	xy := make([]uint32, 0, len(s.Points)*2)
	for _, p := range s.Points {
		xy = append(xy, uint32(p.X), uint32(p.Y))
	}
	return &wirepb.StrokeChunk{Id: s.ID, Color: s.Color, Width: uint32(s.Width), Eraser: s.Tool == "eraser", Xy: xy}
}
func strokeFromProto(s *wirepb.StrokeChunk, maxPoints int) (Stroke, error) {
	if s == nil || len(s.Xy)%2 != 0 || len(s.Xy)/2 > maxPoints {
		return Stroke{}, errors.New("invalid stroke points")
	}
	out := Stroke{ID: s.Id, Color: s.Color, Width: int(s.Width), Tool: "pen", Points: make([]Point, len(s.Xy)/2)}
	if s.Eraser {
		out.Tool = "eraser"
	}
	for i := range out.Points {
		x, y := s.Xy[i*2], s.Xy[i*2+1]
		if x > 10000 || y > 10000 {
			return Stroke{}, errors.New("coordinate out of range")
		}
		out.Points[i] = Point{int(x), int(y)}
	}
	return out, nil
}
func EncodeStrokeCommand(cmd Command) ([]byte, error) {
	if cmd.Type != "stroke" || cmd.Turn < 0 {
		return nil, errors.New("binary command must be a stroke")
	}
	return proto.Marshal(&wirepb.DrawingPacket{Type: wirepb.PacketType_STROKE, Turn: uint32(cmd.Turn), Stroke: strokeToProto(cmd.Stroke)})
}
func decodeProtobufCommand(data []byte) (Command, error) {
	var p wirepb.DrawingPacket
	if err := (proto.UnmarshalOptions{DiscardUnknown: true, RecursionLimit: 16}).Unmarshal(data, &p); err != nil {
		return Command{}, err
	}
	if p.Type != wirepb.PacketType_STROKE || len(p.Strokes) != 0 || len(p.StateJson) != 0 || p.Seq != 0 {
		return Command{}, errors.New("unsupported binary command")
	}
	stroke, err := strokeFromProto(p.Stroke, 96)
	return Command{Type: "stroke", Turn: int(p.Turn), Stroke: stroke}, err
}
func encodeProtobufFrame(f frame) []byte {
	p := &wirepb.DrawingPacket{Turn: uint32(f.Turn), Seq: uint32(f.Seq)}
	switch f.Type {
	case "stroke":
		p.Type = wirepb.PacketType_STROKE
		p.Stroke = strokeToProto(*f.Stroke)
	case "snapshot":
		p.Type = wirepb.PacketType_SNAPSHOT
		p.StateJson = encode(f.State)
		p.Turn = uint32(f.State.Turn)
		p.Seq = uint32(f.State.Seq)
		p.Strokes = make([]*wirepb.StrokeChunk, len(f.Strokes))
		for i, s := range f.Strokes {
			p.Strokes[i] = strokeToProto(s)
		}
	default:
		panic("unsupported protobuf frame")
	}
	data, err := proto.Marshal(p)
	if err != nil {
		panic(err)
	}
	return data
}

// DecodeDrawingFrame is shared with the load generator to measure full decoding.
func DecodeDrawingFrame(data []byte) (Frame, error) {
	var p wirepb.DrawingPacket
	if err := (proto.UnmarshalOptions{DiscardUnknown: true, RecursionLimit: 16}).Unmarshal(data, &p); err != nil {
		return Frame{}, err
	}
	f := Frame{Turn: int(p.Turn), Seq: int(p.Seq)}
	switch p.Type {
	case wirepb.PacketType_STROKE:
		f.Type = "stroke"
		s, err := strokeFromProto(p.Stroke, 96)
		if err != nil {
			return Frame{}, err
		}
		f.Stroke = &s
	case wirepb.PacketType_SNAPSHOT:
		f.Type = "snapshot"
		if len(p.Strokes) > 600 {
			return Frame{}, errors.New("too many strokes")
		}
		if err := json.Unmarshal(p.StateJson, &f.State); err != nil {
			return Frame{}, err
		}
		f.Strokes = make([]Stroke, len(p.Strokes))
		count := 0
		for i, s := range p.Strokes {
			out, err := strokeFromProto(s, 24000)
			if err != nil {
				return Frame{}, err
			}
			count += len(out.Points)
			if count > 24000 {
				return Frame{}, errors.New("too many points")
			}
			f.Strokes[i] = out
		}
	default:
		return Frame{}, errors.New("unsupported binary frame")
	}
	return f, nil
}
