package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"drawguess/internal/game"
	"drawguess/internal/storage"
)

type session = storage.Session
type bucket struct {
	tokens float64
	last   time.Time
}
type API struct {
	hub                *game.Hub
	store              *storage.Store
	secure, trustProxy bool
	mu                 sync.Mutex
	sessions           map[string]session
	rates              map[string]bucket
}

func New(h *game.Hub, s *storage.Store, secure, trustProxy bool) *API {
	return &API{hub: h, store: s, secure: secure, trustProxy: trustProxy, sessions: map[string]session{}, rates: map[string]bucket{}}
}
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { respond(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/session", func(w http.ResponseWriter, r *http.Request) {
		s, ok := a.auth(r)
		if !ok {
			problem(w, 401, "请输入昵称后进入游戏")
			return
		}
		respond(w, 200, s)
	})
	mux.HandleFunc("POST /api/session", a.setSession)
	mux.HandleFunc("POST /api/rooms", a.createRoom)
	mux.HandleFunc("GET /api/rooms/{id}", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.auth(r); !ok {
			problem(w, 401, "请先输入昵称")
			return
		}
		if !a.hub.Exists(r.PathValue("id")) {
			problem(w, 404, "房间不存在或已过期")
			return
		}
		respond(w, 200, map[string]bool{"exists": true})
	})
	mux.HandleFunc("GET /api/history", func(w http.ResponseWriter, r *http.Request) {
		s, ok := a.auth(r)
		if !ok {
			problem(w, 401, "请先进入游戏")
			return
		}
		rows, err := a.store.History(s.ID)
		if err != nil {
			slog.Error("history query failed", "error", err)
			problem(w, 500, "暂时无法读取战绩")
			return
		}
		respond(w, 200, rows)
	})
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		s, ok := a.auth(r)
		if !ok {
			problem(w, 401, "会话已过期，请重新输入昵称")
			return
		}
		a.hub.ServeWS(w, r, s.ID, s.Name)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == "POST" {
			if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host && origin != "https://"+r.Host {
				problem(w, 403, "请求来源无效")
				return
			}
		}
		if r.Method == "POST" || r.URL.Path == "/ws" {
			if !a.allow(r) {
				problem(w, 429, "请求太频繁，请稍后再试")
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}
func (a *API) allow(r *http.Request) bool {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if a.trustProxy && net.ParseIP(ip).IsLoopback() {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			ip = strings.TrimSpace(parts[len(parts)-1])
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	b, exists := a.rates[ip]
	if !exists {
		if len(a.rates) >= 10000 {
			return false
		}
		b = bucket{12, now}
	}
	b.tokens = min(12, b.tokens+now.Sub(b.last).Seconds()*2)
	b.last = now
	ok := b.tokens >= 1
	if ok {
		b.tokens--
	}
	a.rates[ip] = b
	return ok
}
func (a *API) auth(r *http.Request) (session, bool) {
	cookie, err := r.Cookie("dg_session")
	if err != nil {
		return session{}, false
	}
	a.mu.Lock()
	s, ok := a.sessions[cookie.Value]
	a.mu.Unlock()
	if ok {
		return s, time.Now().Before(s.Expires)
	}
	if len(cookie.Value) != 32 {
		return session{}, false
	}
	s, ok = a.store.Session(cookie.Value)
	if ok {
		a.mu.Lock()
		if len(a.sessions) < 10000 {
			a.sessions[cookie.Value] = s
		}
		a.mu.Unlock()
	}
	return s, ok
}
func (a *API) setSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !read(w, r, &body) {
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if utf8.RuneCountInString(body.Name) < 1 || utf8.RuneCountInString(body.Name) > 16 || strings.ContainsFunc(body.Name, unicode.IsControl) {
		problem(w, 400, "昵称需要 1–16 个字符")
		return
	}
	existing, ok := a.auth(r)
	a.mu.Lock()
	defer a.mu.Unlock()
	if !ok && len(a.sessions) >= 10000 {
		problem(w, 503, "当前玩家太多，请稍后再试")
		return
	}
	token := ""
	if ok {
		cookie, _ := r.Cookie("dg_session")
		token = cookie.Value
	} else {
		token = game.ID() + game.ID()
		existing.ID = game.ID()
	}
	existing.Name = body.Name
	existing.Expires = time.Now().Add(24 * time.Hour)
	if err := a.store.SaveSession(token, existing); err != nil {
		slog.Error("save session", "error", err)
		problem(w, 500, "暂时无法创建会话，请稍后重试")
		return
	}
	a.sessions[token] = existing
	http.SetCookie(w, &http.Cookie{Name: "dg_session", Value: token, Path: "/", HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteStrictMode, MaxAge: 86400})
	respond(w, 200, existing)
}
func (a *API) createRoom(w http.ResponseWriter, r *http.Request) {
	s, ok := a.auth(r)
	if !ok {
		problem(w, 401, "请先输入昵称")
		return
	}
	var settings game.Settings
	if !read(w, r, &settings) {
		return
	}
	id, err := a.hub.Create(s.ID, settings)
	if err != nil {
		problem(w, 400, err.Error())
		return
	}
	respond(w, 201, map[string]string{"id": id})
}
func (a *API) Cleanup(now time.Time) {
	a.store.CleanupSessions(now)
	a.mu.Lock()
	defer a.mu.Unlock()
	for k, v := range a.sessions {
		if now.After(v.Expires) {
			delete(a.sessions, k)
		}
	}
	for k, v := range a.rates {
		if now.Sub(v.last) > 10*time.Minute {
			delete(a.rates, k)
		}
	}
}
func read(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		problem(w, 400, "请求格式有误")
		return false
	}
	if err := d.Decode(new(any)); err != io.EOF {
		problem(w, 400, "请求格式有误")
		return false
	}
	return true
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, s string) {
	respond(w, status, map[string]string{"error": s})
}
