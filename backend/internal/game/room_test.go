package game

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func testClient(t *testing.T, id string) *client {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &client{id: id, name: id, ctx: ctx, cancel: cancel, out: make(chan []byte, 128)}
}
func setup(t *testing.T) (*room, *client, *client) {
	t.Helper()
	a, b := testClient(t, "alice"), testClient(t, "bob")
	r := newRoom("TESTROOM", a.id, Settings{1, 60}, nil)
	r.join(a)
	r.join(b)
	r.command(a, Command{Type: "start"})
	return r, a, b
}
func drain(c *client) {
	for {
		select {
		case <-c.out:
		default:
			return
		}
	}
}
func testStroke(id string) Stroke {
	return Stroke{ID: id, Color: "#292d2b", Width: 6, Tool: "pen", Points: []Point{{100, 200}, {300, 400}}}
}

func TestSecretIsolationAndReconnect(t *testing.T) {
	r, a, b := setup(t)
	if len(r.personalState(a.id).Choices) != 3 || len(r.personalState(b.id).Choices) != 0 {
		t.Fatal("choices leaked")
	}
	r.command(a, Command{Type: "choose", Turn: r.turn, Index: 1})
	r.command(a, Command{Type: "stroke", Turn: r.turn, Stroke: testStroke("first")})
	if r.personalState(b.id).Word != "" || r.personalState(a.id).Word == "" {
		t.Fatal("answer isolation failed")
	}
	r.disconnect(b)
	replacement := testClient(t, b.id)
	r.join(replacement)
	var f struct {
		Type    string
		State   State
		Strokes []Stroke
	}
	if err := json.Unmarshal(<-replacement.out, &f); err != nil {
		t.Fatal(err)
	}
	if f.Type != "snapshot" || f.State.Word != "" || len(f.State.Choices) != 0 || len(f.Strokes) != 1 || f.State.Seq != 1 {
		t.Fatalf("invalid guest snapshot: %+v", f)
	}
	r.disconnect(b)
	if !r.player(b.id).Connected {
		t.Fatal("stale disconnect evicted replacement")
	}
}
func TestAuthorityScoringAndTransitions(t *testing.T) {
	r, a, b := setup(t)
	r.command(b, Command{Type: "choose", Turn: r.turn, Index: 0})
	if r.phase != "choosing" {
		t.Fatal("guest chose word")
	}
	r.command(a, Command{Type: "choose", Turn: r.turn, Index: 0})
	r.command(b, Command{Type: "stroke", Turn: r.turn, Stroke: testStroke("illegal")})
	r.command(a, Command{Type: "stroke", Turn: r.turn - 1, Stroke: testStroke("stale")})
	if len(r.strokes) != 0 {
		t.Fatal("unauthorized stroke")
	}
	drain(b)
	answer := r.word
	r.command(b, Command{Type: "guess", Text: answer})
	if r.phase != "reveal" || r.player(b.id).Score < 490 || r.player(a.id).Score != 100 {
		t.Fatalf("bad scoring: %+v", r.publicState())
	}
	var first frame
	if err := json.Unmarshal(<-b.out, &first); err != nil {
		t.Fatal(err)
	}
	// The answer is disclosed only in the reveal phase.
	if first.State.Phase != "reveal" {
		t.Fatal("unexpected scoring state")
	}
	before := r.player(b.id).Score
	r.command(b, Command{Type: "guess", Text: answer})
	if r.player(b.id).Score != before {
		t.Fatal("duplicate scoring")
	}
	r.tick(r.deadline.Add(time.Second))
	if r.drawer != b.id || r.turn != 2 {
		t.Fatal("rotation failed")
	}
	r.command(b, Command{Type: "choose", Turn: 2, Index: 0})
	r.endTurn("done")
	r.tick(r.deadline.Add(time.Second))
	if r.phase != "finished" {
		t.Fatal("game did not finish")
	}
	r.command(a, Command{Type: "start"})
	if r.phase != "choosing" || r.turn != 1 || r.player(a.id).Score != 0 {
		t.Fatal("replay failed")
	}
}
func TestCorrectGuessNeverBroadcastAsChat(t *testing.T) {
	r, a, b := setup(t)
	c := testClient(t, "charlie")
	r.phase = "lobby"
	r.join(c)
	r.command(a, Command{Type: "start"})
	r.command(a, Command{Type: "choose", Turn: r.turn, Index: 0})
	drain(c)
	r.command(b, Command{Type: "guess", Text: r.word})
	var f frame
	_ = json.Unmarshal(<-c.out, &f)
	if f.State == nil || f.State.Phase != "drawing" || f.State.Word != "" || f.State.RevealedWord != "" {
		t.Fatal("answer leaked while still drawing")
	}
	for _, m := range f.State.Messages {
		if m.Kind == "chat" && m.Text == r.word {
			t.Fatal("correct answer sent as chat")
		}
	}
}
func TestStrokeLimitsUndoAndClear(t *testing.T) {
	r, a, _ := setup(t)
	r.command(a, Command{Type: "choose", Turn: r.turn, Index: 0})
	r.command(a, Command{Type: "stroke", Turn: r.turn, Stroke: testStroke("one")})
	r.command(a, Command{Type: "stroke", Turn: r.turn, Stroke: testStroke("one")})
	if len(r.strokes) != 1 || r.points != 4 {
		t.Fatal("batch append failed")
	}
	bad := testStroke("bad")
	bad.Points[0].X = -1
	r.command(a, Command{Type: "stroke", Turn: r.turn, Stroke: bad})
	if r.points != 4 {
		t.Fatal("accepted invalid coordinate")
	}
	r.command(a, Command{Type: "undo", Turn: r.turn})
	if len(r.strokes) != 0 || r.points != 0 || r.seq != 3 {
		t.Fatal("undo failed")
	}
	r.command(a, Command{Type: "stroke", Turn: r.turn, Stroke: testStroke("two")})
	r.command(a, Command{Type: "clear", Turn: r.turn})
	if len(r.strokes) != 0 || r.points != 0 || r.seq != 5 {
		t.Fatal("clear failed")
	}
	r.points = 24000
	r.command(a, Command{Type: "stroke", Turn: r.turn, Stroke: testStroke("full")})
	if len(r.strokes) != 0 {
		t.Fatal("point cap failed")
	}
}
func TestDisconnectHostAndTimeout(t *testing.T) {
	r, a, b := setup(t)
	r.disconnect(a)
	if r.host != b.id {
		t.Fatal("host not transferred")
	}
	r.player(a.id).disconnected = time.Now().Add(-11 * time.Second)
	r.tick(time.Now())
	if r.phase != "reveal" {
		t.Fatal("drawer not skipped")
	}
	r.player(a.id).disconnected = time.Now().Add(-46 * time.Second)
	r.tick(time.Now())
	if r.player(a.id) != nil {
		t.Fatal("seat not reclaimed")
	}
	r.tick(r.deadline.Add(time.Second))
	if r.phase != "lobby" {
		t.Fatal("insufficient players should return to lobby")
	}
}
func TestSlowConsumerIsDisconnected(t *testing.T) {
	c := testClient(t, "slow")
	for i := 0; i < 129; i++ {
		c.send([]byte("test"))
	}
	select {
	case <-c.ctx.Done():
	default:
		t.Fatal("slow consumer not disconnected")
	}
}
func TestExpiredRoundCannotAcceptStroke(t *testing.T) {
	r, a, _ := setup(t)
	r.command(a, Command{Type: "choose", Turn: r.turn})
	r.deadline = time.Now().Add(-time.Second)
	r.command(a, Command{Type: "stroke", Turn: r.turn, Stroke: testStroke("late")})
	if len(r.strokes) != 0 || r.phase != "reveal" {
		t.Fatal("accepted a late stroke")
	}
}
