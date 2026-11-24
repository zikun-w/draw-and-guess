package game

import (
	"testing"
	"time"
)

func TestSketchRevealsOnlyVisiblePointsAndCancelsOnUndo(t *testing.T) {
	r := newRoom("ROOM", "artist", Settings{1, 90}, nil)
	req := func(op string, a AgentAction) AgentResult {
		return r.agentResult(&agentRequest{id: "artist", name: "画手", op: op, action: a}, time.Now())
	}
	req("join", AgentAction{})
	r.join(testClient(t, "web"))
	r.applyCommand(r.player("artist").client, Command{Type: "start"})
	r.startDrawing(0)
	seq := 0
	action := AgentAction{Type: "sketch", RoundKey: r.roundKey(), Seq: &seq, Shapes: []Shape{
		{Kind: "line", Points: [][]int{{0, 0}, {800, 500}}}, {Kind: "ellipse", Points: [][]int{{100, 100}, {600, 500}}},
	}}
	if res := req("action", action); res.Status != 200 || r.seq != 1 || len(r.strokes) != 1 || len(r.strokes[0].Points) != 1 {
		t.Fatal("first frame", res.Status, r.seq)
	}
	first := req("board", AgentAction{}).Strokes
	id := first[0].ID
	r.advanceSketch(r.sketch.next)
	if r.strokes[0].ID != id || len(r.strokes[0].Points) <= 1 || len(first[0].Points) != 1 {
		t.Fatal("invalid partial stroke or snapshot")
	}
	seq = r.seq
	if req("action", action).Status != 409 {
		t.Fatal("overlapping drawings accepted")
	}
	if req("action", AgentAction{Type: "undo", RoundKey: r.roundKey(), Seq: &seq}).Status != 200 || r.sketch != nil || len(r.strokes) != 0 {
		t.Fatal("undo must cancel the remaining plan")
	}
	seq = r.seq
	req("action", action)
	for r.sketch != nil {
		r.advanceSketch(r.sketch.next)
	}
	if len(r.strokes) != 2 || r.strokes[0].Points[len(r.strokes[0].Points)-1] != (Point{8000, 7692}) {
		t.Fatal("incomplete drawing")
	}
	seq = r.seq
	req("action", action)
	r.endTurn("test")
	if r.sketch != nil {
		t.Fatal("old turn still drawing")
	}
}
