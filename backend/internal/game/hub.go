package game

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type Hub struct {
	mu       sync.Mutex
	rooms    map[string]*room
	recorder Recorder
	closing  bool
	wg       sync.WaitGroup
	maxRooms int
}

func NewHub(rec Recorder) *Hub { return NewHubWithLimit(rec, 1000) }
func NewHubWithLimit(rec Recorder, limit int) *Hub {
	if limit < 1 || limit > 1000 {
		panic("room limit must be between 1 and 1000")
	}
	return &Hub{rooms: map[string]*room{}, recorder: rec, maxRooms: limit}
}
func (h *Hub) Exists(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return !h.closing && h.rooms[id] != nil
}
func (h *Hub) Create(host string, s Settings) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closing || len(h.rooms) >= h.maxRooms {
		return "", errors.New("房间数量已达上限，请稍后再试")
	}
	if !validSettings(s) {
		return "", errors.New("轮数应为 1–5，绘画时间应为 30–180 秒")
	}
	id := ID()[:8]
	for h.rooms[id] != nil {
		id = ID()[:8]
	}
	r := newRoom(id, host, s, h.recorder)
	r.retire = func() { h.mu.Lock(); delete(h.rooms, id); h.mu.Unlock(); h.wg.Done() }
	h.rooms[id] = r
	h.wg.Add(1)
	go r.run()
	return id, nil
}
func (h *Hub) Close() {
	h.mu.Lock()
	h.closing = true
	rs := make([]*room, 0, len(h.rooms))
	for _, r := range h.rooms {
		rs = append(rs, r)
	}
	h.mu.Unlock()
	for _, r := range rs {
		r.submit(event{kind: "stop"})
	}
	h.wg.Wait()
}
func (h *Hub) ServeWS(w http.ResponseWriter, req *http.Request, id, name string) {
	h.mu.Lock()
	r := h.rooms[req.URL.Query().Get("room")]
	h.mu.Unlock()
	if r == nil {
		http.Error(w, "房间不存在或已过期", http.StatusNotFound)
		return
	}
	conn, err := websocket.Accept(w, req, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(16 * 1024)
	ctx, cancel := context.WithCancel(context.Background())
	c := &client{id: id, name: name, conn: conn, ctx: ctx, cancel: cancel, out: make(chan []byte, 128)}
	defer c.stop()
	defer r.submit(event{kind: "disconnect", client: c})
	go c.writer()
	if !r.submit(event{kind: "join", client: c}) {
		return
	}
	tokens := 160.0
	last := time.Now()
	lastSync := time.Time{}
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		now := time.Now()
		tokens = min(160, tokens+now.Sub(last).Seconds()*80)
		last = now
		if tokens < 1 {
			c.error("操作太快，请稍后重试")
			return
		}
		tokens--
		cmd, err := decodeCommand(data)
		if err != nil {
			c.error("消息格式有误")
			continue
		}
		if cmd.Type == "sync" {
			if now.Sub(lastSync) < 2*time.Second {
				continue
			}
			lastSync = now
		}
		select {
		case r.inbox <- event{kind: "command", client: c, cmd: cmd}:
		case <-ctx.Done():
			return
		case <-r.done:
			return
		}
	}
}

type client struct {
	id, name string
	conn     *websocket.Conn
	ctx      context.Context
	cancel   context.CancelFunc
	out      chan []byte
	once     sync.Once
}

func (c *client) stop() {
	c.once.Do(func() {
		c.cancel()
		if c.conn != nil {
			_ = c.conn.CloseNow()
		}
	})
}
func (c *client) send(b []byte) {
	select {
	case <-c.ctx.Done():
		return
	default:
	}
	select {
	case c.out <- b:
	default:
		c.stop()
	}
}
func (c *client) error(s string) { c.send(encode(frame{Type: "error", Message: s})) }
func (c *client) fail(s string)  { c.send(encode(frame{Type: "fatal", Message: s})) }
func (c *client) writer() {
	defer c.stop()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case data := <-c.out:
			ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			err := c.conn.Write(ctx, websocket.MessageText, data)
			cancel()
			if err != nil {
				return
			}
			// Fatal frames are only sent on rejected room admission.
			if len(data) > 15 && string(data[:15]) == `{"type":"fatal"` {
				return
			}
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(c.ctx, 8*time.Second)
			err := c.conn.Ping(ctx)
			cancel()
			if err != nil {
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}
