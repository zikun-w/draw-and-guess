package game

import (
	"crypto/rand"
	_ "embed"
	"encoding/base32"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

//go:embed words.txt
var wordsText string
var words = strings.Fields(wordsText)
var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func ID() string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
}
func randomN(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err)
	}
	return int(v.Int64())
}

type room struct {
	id         string
	host       string
	phase      string
	settings   Settings
	players    []*Player
	schedule   []string
	turn       int
	drawer     string
	word       string
	choices    []string
	deadline   time.Time
	created    time.Time
	emptySince time.Time
	strokes    []Stroke
	points     int
	seq        int
	messages   []Message
	messageID  int
	inbox      chan event
	done       chan struct{}
	recorder   Recorder
	matchID    string
	retire     func()
}

func newRoom(id, host string, s Settings, rec Recorder) *room {
	return &room{id: id, host: host, phase: "lobby", settings: s, created: time.Now(), inbox: make(chan event, 256), done: make(chan struct{}), recorder: rec, players: []*Player{}, strokes: []Stroke{}, messages: []Message{}}
}
func (r *room) run() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	defer close(r.done)
	defer r.retire()
	for {
		select {
		case e := <-r.inbox:
			switch e.kind {
			case "join":
				r.join(e.client)
			case "disconnect":
				r.disconnect(e.client)
			case "command":
				r.command(e.client, e.cmd)
			case "stop":
				r.closeClients()
				return
			}
		case now := <-ticker.C:
			r.tick(now)
			if !r.emptySince.IsZero() && now.Sub(r.emptySince) > 2*time.Minute {
				r.closeClients()
				return
			}
			if len(r.players) == 0 && now.Sub(r.created) > 5*time.Minute {
				return
			}
		}
	}
}
func (r *room) submit(e event) bool {
	select {
	case r.inbox <- e:
		return true
	case <-r.done:
		return false
	}
}
func (r *room) player(id string) *Player {
	for _, p := range r.players {
		if p.ID == id {
			return p
		}
	}
	return nil
}
func (r *room) connected() int {
	n := 0
	for _, p := range r.players {
		if p.Connected {
			n++
		}
	}
	return n
}
func (r *room) closeClients() {
	for _, p := range r.players {
		if p.client != nil {
			p.client.stop()
		}
	}
}
func (r *room) addMessage(name, text, kind string) {
	r.messageID++
	r.messages = append(r.messages, Message{r.messageID, name, text, kind})
	if len(r.messages) > 40 {
		r.messages = r.messages[len(r.messages)-40:]
	}
}
func (r *room) join(c *client) {
	p := r.player(c.id)
	if p == nil {
		if len(r.players) >= MaxPlayers {
			c.fail("房间已满，请换一个房间")
			return
		}
		if r.phase != "lobby" && r.phase != "finished" {
			c.fail("游戏正在进行，结束后再加入吧")
			return
		}
		p = &Player{ID: c.id, Name: c.name}
		r.players = append(r.players, p)
		r.addMessage("", c.name+" 加入了房间", "system")
	}
	if p.client != nil {
		p.client.fail("该身份已在另一个页面进入房间")
	}
	p.client = c
	p.Connected = true
	p.disconnected = time.Time{}
	r.emptySince = time.Time{}
	if hp := r.player(r.host); hp == nil || !hp.Connected {
		r.host = c.id
	}
	r.snapshot(c)
	r.broadcastState()
}
func (r *room) disconnect(c *client) {
	p := r.player(c.id)
	if p == nil || p.client != c {
		return
	}
	p.client = nil
	p.Connected = false
	p.disconnected = time.Now()
	r.transferHost()
	if r.connected() == 0 {
		r.emptySince = time.Now()
	}
	r.broadcastState()
}
func (r *room) transferHost() {
	if p := r.player(r.host); p != nil && p.Connected {
		return
	}
	for _, p := range r.players {
		if p.Connected {
			r.host = p.ID
			return
		}
	}
}
func (r *room) publicState() State {
	ps := make([]Player, 0, len(r.players))
	for _, p := range r.players {
		ps = append(ps, *p)
	}
	s := State{Version: 1, ID: r.id, Phase: r.phase, HostID: r.host, DrawerID: r.drawer, Players: ps, Settings: r.settings, Turn: r.turn, TotalTurns: len(r.schedule), ServerTime: time.Now().UnixMilli(), WordLength: utf8.RuneCountInString(r.word), Seq: r.seq, Messages: r.messages}
	if !r.deadline.IsZero() {
		s.Deadline = r.deadline.UnixMilli()
	}
	if r.phase == "reveal" || r.phase == "finished" {
		s.RevealedWord = r.word
	}
	return s
}
func (r *room) personalState(id string) State {
	s := r.publicState()
	if id == r.drawer {
		if r.phase == "choosing" {
			s.Choices = r.choices
		}
		if r.phase == "drawing" {
			s.Word = r.word
		}
	}
	return s
}
func (r *room) snapshot(c *client) {
	s := r.personalState(c.id)
	if c.binary {
		c.sendBinary(encodeProtobufFrame(frame{Type: "snapshot", State: &s, Strokes: r.strokes}))
		return
	}
	c.send(encode(frame{Type: "snapshot", State: &s, Strokes: r.strokes}))
}
func (r *room) broadcastState() {
	s := r.publicState()
	data := encode(frame{Type: "state", State: &s})
	for _, p := range r.players {
		if p.client == nil {
			continue
		}
		if p.ID == r.drawer {
			private := r.personalState(p.ID)
			p.client.send(encode(frame{Type: "state", State: &private}))
		} else {
			p.client.send(data)
		}
	}
}
func (r *room) broadcast(f frame) {
	var data, binary []byte
	for _, p := range r.players {
		if p.client != nil {
			if p.client.binary && f.Type == "stroke" {
				if binary == nil {
					binary = encodeProtobufFrame(f)
				}
				p.client.sendBinary(binary)
			} else {
				if data == nil {
					data = encode(f)
				}
				p.client.send(data)
			}
		}
	}
}
func (r *room) command(c *client, cmd Command) {
	p := r.player(c.id)
	if p == nil || p.client != c {
		return
	}
	if !r.deadline.IsZero() && !time.Now().Before(r.deadline) {
		r.tick(time.Now())
	}
	switch cmd.Type {
	case "sync":
		r.snapshot(c)
	case "leave":
		p.client = nil
		p.Connected = false
		r.remove(p.ID)
		c.stop()
		r.transferHost()
		r.addMessage("", p.Name+" 离开了房间", "system")
		if r.connected() == 0 {
			r.emptySince = time.Now()
		}
		if r.drawer == p.ID && (r.phase == "drawing" || r.phase == "choosing") {
			r.endTurn("画手离开了，本轮结束")
		}
		r.broadcastState()
	case "settings":
		if p.ID != r.host || (r.phase != "lobby" && r.phase != "finished") {
			return
		}
		if validSettings(cmd.Settings) {
			r.settings = cmd.Settings
			r.broadcastState()
		}
	case "start":
		if p.ID != r.host || (r.phase != "lobby" && r.phase != "finished") {
			return
		}
		if r.connected() < 2 {
			c.error("至少需要两位玩家才能开始")
			return
		}
		r.schedule = nil
		for i := 0; i < r.settings.Rounds; i++ {
			for _, v := range r.players {
				if v.Connected {
					r.schedule = append(r.schedule, v.ID)
				}
			}
		}
		for _, v := range r.players {
			v.Score = 0
			v.Guessed = false
		}
		r.turn = 0
		r.matchID = ID()
		r.nextTurn()
	case "choose":
		if r.phase == "choosing" && p.ID == r.drawer && cmd.Turn == r.turn && cmd.Index >= 0 && cmd.Index < len(r.choices) {
			r.startDrawing(cmd.Index)
		}
	case "stroke":
		if r.phase != "drawing" || p.ID != r.drawer || cmd.Turn != r.turn {
			return
		}
		r.applyStroke(c, cmd.Stroke)
	case "undo", "clear":
		if r.phase != "drawing" || p.ID != r.drawer || cmd.Turn != r.turn {
			return
		}
		if cmd.Type == "clear" {
			r.strokes = []Stroke{}
			r.points = 0
		} else if len(r.strokes) > 0 {
			last := r.strokes[len(r.strokes)-1]
			r.points -= len(last.Points)
			r.strokes = r.strokes[:len(r.strokes)-1]
		}
		r.seq++
		r.broadcast(frame{Type: cmd.Type, Seq: r.seq, Turn: r.turn})
	case "guess":
		text := strings.TrimSpace(cmd.Text)
		if text == "" || utf8.RuneCountInString(text) > 120 {
			return
		}
		if time.Since(p.lastGuess) < 650*time.Millisecond {
			c.error("发言太快了，稍等一下")
			return
		}
		p.lastGuess = time.Now()
		if r.phase == "drawing" {
			if p.ID == r.drawer || p.Guessed {
				c.error("本轮结束后就可以继续聊天了")
				return
			}
			if normalize(text) == normalize(r.word) {
				p.Guessed = true
				remaining := max(0, time.Until(r.deadline).Seconds())
				score := 100 + int(remaining/float64(r.settings.DrawSeconds)*400)
				p.Score += score
				if d := r.player(r.drawer); d != nil {
					d.Score += 100
				}
				r.addMessage("", fmt.Sprintf("%s 猜中了！+%d 分", p.Name, score), "correct")
				if r.allGuessed() {
					r.endTurn("所有人都猜中了！")
				} else {
					r.broadcastState()
				}
				return
			}
		}
		r.addMessage(p.Name, text, "chat")
		r.broadcastState()
	}
}
func normalize(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), "")) }
func validSettings(s Settings) bool {
	return s.Rounds >= 1 && s.Rounds <= 5 && s.DrawSeconds >= 30 && s.DrawSeconds <= 180
}
func (r *room) applyStroke(c *client, s Stroke) {
	if len(s.ID) < 1 || len(s.ID) > 40 || !colorPattern.MatchString(s.Color) || s.Width < 1 || s.Width > 40 || (s.Tool != "pen" && s.Tool != "eraser") || len(s.Points) == 0 || len(s.Points) > 96 {
		return
	}
	for _, pt := range s.Points {
		if pt.X < 0 || pt.X > 10000 || pt.Y < 0 || pt.Y > 10000 {
			return
		}
	}
	if r.points+len(s.Points) > 24000 {
		c.error("本轮画板已满，可以清空后继续")
		return
	}
	last := len(r.strokes) - 1
	if last >= 0 && r.strokes[last].ID == s.ID {
		old := &r.strokes[last]
		if old.Color != s.Color || old.Width != s.Width || old.Tool != s.Tool {
			return
		}
		old.Points = append(old.Points, s.Points...)
	} else {
		if len(r.strokes) >= 600 {
			c.error("笔画太多了，可以清空后继续")
			return
		}
		for _, old := range r.strokes {
			if old.ID == s.ID {
				return
			}
		}
		r.strokes = append(r.strokes, s)
	}
	r.points += len(s.Points)
	r.seq++
	r.broadcast(frame{Type: "stroke", Stroke: &s, Seq: r.seq, Turn: r.turn})
}
func (r *room) allGuessed() bool {
	n := 0
	for _, p := range r.players {
		if p.ID != r.drawer && p.Connected {
			n++
			if !p.Guessed {
				return false
			}
		}
	}
	return n > 0
}
func (r *room) nextTurn() {
	if r.connected() < 2 {
		r.phase = "lobby"
		r.drawer = ""
		r.word = ""
		r.deadline = time.Time{}
		r.addMessage("", "人数不足，回到等待房间", "system")
		r.broadcastState()
		return
	}
	for {
		r.turn++
		if r.turn > len(r.schedule) {
			r.finish()
			return
		}
		p := r.player(r.schedule[r.turn-1])
		if p != nil && p.Connected {
			r.drawer = p.ID
			break
		}
	}
	r.phase = "choosing"
	r.word = ""
	r.choices = nil
	r.strokes = []Stroke{}
	r.points = 0
	r.seq = 0
	for _, p := range r.players {
		p.Guessed = false
	}
	used := map[string]bool{}
	for len(r.choices) < 3 {
		w := words[randomN(len(words))]
		if !used[w] {
			r.choices = append(r.choices, w)
			used[w] = true
		}
	}
	r.deadline = time.Now().Add(12 * time.Second)
	r.addMessage("", "新的画手正在选词…", "system")
	r.broadcastState()
}
func (r *room) startDrawing(i int) {
	r.word = r.choices[i]
	r.choices = nil
	r.phase = "drawing"
	r.deadline = time.Now().Add(time.Duration(r.settings.DrawSeconds) * time.Second)
	r.broadcastState()
}
func (r *room) endTurn(reason string) {
	if r.phase != "drawing" && r.phase != "choosing" {
		return
	}
	if r.word == "" && len(r.choices) > 0 {
		r.word = r.choices[0]
	}
	r.phase = "reveal"
	r.choices = nil
	r.deadline = time.Now().Add(6 * time.Second)
	r.addMessage("", reason+" 答案是「"+r.word+"」", "system")
	r.broadcastState()
}
func (r *room) finish() {
	r.phase = "finished"
	r.deadline = time.Time{}
	r.addMessage("", "本局结束，看看谁是灵魂画手！", "system")
	result := Result{ID: r.matchID, Room: r.id, FinishedAt: time.Now().UnixMilli(), Players: []ResultPlayer{}}
	for _, p := range r.players {
		result.Players = append(result.Players, ResultPlayer{p.ID, p.Name, p.Score})
	}
	if r.recorder != nil && !r.recorder.Record(result) {
		r.addMessage("", "战绩暂时无法保存，本局得分仍可查看", "system")
	}
	r.broadcastState()
}
func (r *room) remove(id string) {
	for i, p := range r.players {
		if p.ID == id {
			r.players = append(r.players[:i], r.players[i+1:]...)
			return
		}
	}
}
func (r *room) tick(now time.Time) {
	changed := false
	for i := len(r.players) - 1; i >= 0; i-- {
		p := r.players[i]
		if !p.Connected && !p.disconnected.IsZero() && now.Sub(p.disconnected) > 45*time.Second {
			r.remove(p.ID)
			changed = true
		}
	}
	if r.phase == "choosing" || r.phase == "drawing" {
		d := r.player(r.drawer)
		if d == nil || (!d.Connected && now.Sub(d.disconnected) > 10*time.Second) {
			r.endTurn("画手掉线，本轮结束")
			return
		}
	}
	if !r.deadline.IsZero() && !now.Before(r.deadline) {
		switch r.phase {
		case "choosing":
			r.startDrawing(randomN(len(r.choices)))
		case "drawing":
			r.endTurn("时间到！")
		case "reveal":
			r.nextTurn()
		}
		return
	}
	if changed {
		r.broadcastState()
	}
}
