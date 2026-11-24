package game

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const agentLease = 45 * time.Second

// A polling seat is owned by the same room goroutine as WebSocket seats.
type pollSeat struct {
	seen    time.Time
	problem string
}

type AgentAction struct {
	Type     string  `json:"type"`
	RoundKey string  `json:"roundKey"`
	Seq      *int    `json:"seq"`
	Index    *int    `json:"index"`
	Text     string  `json:"text"`
	Shapes   []Shape `json:"shapes"`
}

type AgentView struct {
	Me       string `json:"me"`
	Next     string `json:"next"`
	RoundKey string `json:"roundKey"`
	State    State  `json:"state"`
}

type AgentResult struct {
	View    AgentView
	Strokes []Stroke
	Status  int
	Error   string
}

func denied(status int, message string) AgentResult {
	return AgentResult{Status: status, Error: message}
}

type agentRequest struct {
	ctx          context.Context
	id, name, op string
	action       AgentAction
	reply        chan AgentResult
}

func (h *Hub) Agent(ctx context.Context, roomID, id, name, op string, a AgentAction) AgentResult {
	h.mu.Lock()
	r := h.rooms[roomID]
	h.mu.Unlock()
	if r == nil {
		return denied(404, "房间不存在或已过期")
	}
	q := &agentRequest{ctx, id, name, op, a, make(chan AgentResult, 1)}
	select {
	case r.inbox <- event{kind: "agent", agent: q}:
	case <-ctx.Done():
		return denied(408, "请求已取消")
	case <-r.done:
		return denied(404, "房间已关闭")
	}
	select {
	case res := <-q.reply:
		return res
	case <-ctx.Done():
		return denied(408, "请求已取消")
	case <-r.done:
		return denied(404, "房间已关闭")
	}
}

func (r *room) handleAgent(q *agentRequest) {
	if q.ctx.Err() != nil {
		return
	}
	now := time.Now()
	if !r.deadline.IsZero() && !now.Before(r.deadline) {
		r.tick(now)
	}
	q.reply <- r.agentResult(q, now)
}

func (r *room) agentResult(q *agentRequest, now time.Time) AgentResult {
	p := r.player(q.id)
	if q.op == "join" && (p == nil || p.client == nil) {
		c := &client{id: q.id, name: q.name, poll: &pollSeat{seen: now}}
		r.join(c)
		if c.poll.problem != "" {
			return denied(409, c.poll.problem)
		}
		p = r.player(q.id)
	}
	if p == nil || p.client == nil || p.client.poll == nil {
		return denied(409, "此身份没有 HTTP 座位，请确认身份后调用 join")
	}
	c := p.client
	c.poll.seen = now
	c.poll.problem = ""
	if q.op == "action" {
		result := r.agentAction(p, q.action)
		if result.Error != "" {
			return result
		}
		if c.poll.problem != "" {
			return denied(409, c.poll.problem)
		}
		if q.action.Type == "leave" {
			return AgentResult{Status: http.StatusNoContent}
		}
	}
	result := AgentResult{Status: 200, View: AgentView{p.ID, r.nextAction(p), r.roundKey(), r.personalState(p.ID)}}
	if q.op == "board" {
		// A published point is immutable. Copy the slice headers before handing
		// them to the renderer; the room may extend the last stroke meanwhile.
		result.Strokes = append([]Stroke(nil), r.strokes...)
	}
	return result
}

func (r *room) roundKey() string { return r.matchID + "/" + strconv.Itoa(r.turn) }
func (r *room) nextAction(p *Player) string {
	switch r.phase {
	case "finished":
		return "finished"
	case "lobby":
		if r.host == p.ID && r.connected() >= 2 {
			return "start"
		}
	case "choosing":
		if r.drawer == p.ID {
			return "choose"
		}
	case "drawing":
		if r.drawer == p.ID {
			return "draw"
		}
		if !p.Guessed {
			return "guess"
		}
	}
	return "wait"
}

func (r *room) agentAction(p *Player, a AgentAction) AgentResult {
	cmd := Command{Type: a.Type, Turn: r.turn, Text: strings.TrimSpace(a.Text)}
	if a.Type != "leave" && a.RoundKey != r.roundKey() {
		return denied(409, "回合已变化，请重新读取 state")
	}
	next := r.nextAction(p)
	switch a.Type {
	case "leave":
	case "start":
		if p.ID != r.host || (r.phase != "lobby" && r.phase != "finished") || r.connected() < 2 {
			return denied(409, "需要房主、至少两人和可开局的阶段")
		}
	case "choose":
		if next != "choose" {
			return denied(409, "当前不能选词")
		}
		if a.Index == nil || *a.Index < 0 || *a.Index >= len(r.choices) {
			return denied(400, "index 必须是当前候选词的下标，从 0 开始")
		}
		cmd.Index = *a.Index
	case "guess":
		if next != "guess" {
			return denied(409, "当前不能猜词")
		}
		if cmd.Text == "" || utf8.RuneCountInString(cmd.Text) > 120 {
			return denied(400, "猜词需要 1–120 个字符")
		}
	case "draw", "clear", "undo":
		if next != "draw" {
			return denied(409, "当前不能画图")
		}
		if a.Seq == nil || *a.Seq != r.seq {
			return denied(409, "画板已变化，请读取最新 seq")
		}
		if a.Type == "draw" {
			strokes, err := makeStrokes(a.Shapes)
			if err != nil {
				return denied(400, err.Error())
			}
			points := r.points
			for _, s := range strokes {
				points += len(s.Points)
			}
			if len(r.strokes)+len(strokes) > 600 || points > 24000 {
				return denied(409, "本轮画板已满")
			}
			for _, s := range strokes {
				r.applyStroke(p.client, s)
			}
			return AgentResult{}
		}
	default:
		return denied(400, "未知的操作")
	}
	r.applyCommand(p.client, cmd)
	return AgentResult{}
}
