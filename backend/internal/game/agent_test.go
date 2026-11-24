package game

import (
	"testing"
	"time"
)

func TestPollingAndWebPlayersShareATurn(t *testing.T) {
	r := newRoom("ROOM", "agent", Settings{1, 90}, nil)
	request := func(op string, a AgentAction) AgentResult {
		return r.agentResult(&agentRequest{id: "agent", name: "澜", op: op, action: a}, time.Now())
	}
	join := request("join", AgentAction{})
	if join.Status != 200 || request("join", AgentAction{}).Status != 200 || len(r.players) != 1 {
		t.Fatal("join must be idempotent")
	}
	web := testClient(t, "web")
	r.join(web)
	start := request("action", AgentAction{Type: "start", RoundKey: join.View.RoundKey})
	if start.View.Next != "choose" || len(r.personalState(web.id).Choices) != 0 {
		t.Fatal("invalid chooser")
	}
	key := start.View.RoundKey
	zero := 0
	request("action", AgentAction{Type: "choose", RoundKey: key, Index: &zero})
	draw := AgentAction{Type: "draw", RoundKey: key, Seq: &zero, Shapes: []Shape{
		{Kind: "rect", Points: [][]int{{20, 20}, {30, 30}}},
		{Kind: "line", Points: [][]int{{4000, 5}}},
	}}
	if request("action", draw).Status != 400 || r.seq != 0 {
		t.Fatal("partial invalid batch")
	}
	draw.Shapes[1] = Shape{Kind: "ellipse", Points: [][]int{{40, 40}, {100, 100}}}
	drain(web)
	if request("action", draw).Status != 200 || len(web.out) != 2 || r.seq != 2 {
		t.Fatal("missing web strokes")
	}
	if request("action", draw).Status != 409 || r.seq != 2 {
		t.Fatal("replayed draw")
	}
	r.command(web, Command{Type: "guess", Text: r.word})
	if r.phase != "reveal" {
		t.Fatal("web guess failed")
	}
	r.tick(r.deadline.Add(time.Millisecond))
	r.command(web, Command{Type: "choose", Index: 0, Turn: r.turn})
	guest := request("state", AgentAction{}).View
	if guest.Next != "guess" || guest.State.Word != "" || len(guest.State.Choices) != 0 || guest.State.RevealedWord != "" {
		t.Fatal("guesser saw private state")
	}
	if request("action", AgentAction{Type: "guess", RoundKey: key, Text: r.word}).Status != 409 {
		t.Fatal("old turn guess")
	}
	if request("action", AgentAction{Type: "guess", RoundKey: guest.RoundKey, Text: r.word}).View.State.Phase != "reveal" {
		t.Fatal("agent guess failed")
	}
	r.tick(r.deadline.Add(time.Millisecond))
	if request("state", AgentAction{}).View.Next != "finished" {
		t.Fatal("game did not finish")
	}
	p := r.player("agent")
	score := p.Score
	r.tick(p.client.poll.seen.Add(agentLease + time.Second))
	if p.Connected || r.host != web.id {
		t.Fatal("expired seat remained online or kept host")
	}
	if request("join", AgentAction{}).Status != 200 || len(r.players) != 2 || p.Score != score {
		t.Fatal("seat recovery failed")
	}
}
