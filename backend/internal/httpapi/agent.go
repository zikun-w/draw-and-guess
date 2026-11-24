package httpapi

import (
	"bytes"
	"fmt"
	"image/png"
	"net/http"

	"drawguess/internal/game"
)

type boardImage struct {
	tag string
	png []byte
}

func (a *API) registerAgent(mux *http.ServeMux) {
	for pattern, op := range map[string]string{
		"POST /api/agent/rooms/{id}/join":     "join",
		"GET /api/agent/rooms/{id}/state":     "state",
		"GET /api/agent/rooms/{id}/board.png": "board",
		"POST /api/agent/rooms/{id}/actions":  "action",
	} {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			s, ok := a.auth(r)
			if !ok {
				problem(w, 401, "请先用游戏昵称创建会话")
				return
			}
			var cmd game.AgentAction
			if op == "action" && !read(w, r, &cmd) {
				return
			}
			res := a.hub.Agent(r.Context(), r.PathValue("id"), s.ID, s.Name, op, cmd)
			if res.Error != "" {
				problem(w, res.Status, res.Error)
				return
			}
			if res.Status == 204 {
				w.WriteHeader(204)
				return
			}
			if op != "board" {
				respond(w, 200, res.View)
				return
			}
			tag := fmt.Sprintf(`"%s-%s-%d"`, res.View.State.ID, res.View.RoundKey, res.View.State.Seq)
			w.Header().Set("ETag", tag)
			if r.Header.Get("If-None-Match") == tag {
				w.WriteHeader(304)
				return
			}
			if cached := a.lastBoard.Load(); cached != nil && cached.tag == tag {
				w.Header().Set("Content-Type", "image/png")
				_, _ = w.Write(cached.png)
				return
			}
			select {
			case a.imageSlots <- struct{}{}:
				defer func() { <-a.imageSlots }()
			default:
				w.Header().Set("Retry-After", "1")
				problem(w, 503, "画板正在生成，请稍后重试")
				return
			}
			var buf bytes.Buffer
			encoder := png.Encoder{CompressionLevel: png.BestSpeed}
			if err := encoder.Encode(&buf, renderBoard(res.Strokes)); err != nil {
				problem(w, 500, "画板生成失败")
				return
			}
			w.Header().Set("Content-Type", "image/png")
			data := buf.Bytes()
			a.lastBoard.Store(&boardImage{tag: tag, png: data})
			_, _ = w.Write(data)
		})
	}
}
