package httpapi

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"drawguess/internal/game"
	"drawguess/internal/storage"
)

func TestAgentBoardChecksSeatEvenWhenCached(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	h := game.NewHub(store)
	defer h.Close()
	handler := New(h, store, false, false).Handler()
	call := func(method, path, body string, cookie *http.Cookie, etag string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		req.Header.Set("If-None-Match", etag)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}
	room, err := h.Create("host", game.Settings{Rounds: 1, DrawSeconds: 30})
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/agent/rooms/" + room
	login := func() *http.Cookie {
		res := call("POST", "/api/session", `{"name":"Player"}`, nil, "")
		if res.Code != 200 {
			t.Fatal(res.Code, res.Body)
		}
		return res.Result().Cookies()[0]
	}
	seat, outsider := login(), login()
	if res := call("POST", base+"/join", "{}", seat, ""); res.Code != 200 {
		t.Fatal(res.Code, res.Body)
	}
	first := call("GET", base+"/board.png", "", seat, "")
	tag := first.Header().Get("ETag")
	if first.Code != 200 || tag == "" || first.Header().Get("Content-Type") != "image/png" {
		t.Fatal("missing PNG or ETag")
	}
	if res := call("GET", base+"/board.png", "", seat, tag); res.Code != 304 || res.Body.Len() != 0 {
		t.Fatal("conditional GET failed")
	}
	if res := call("GET", base+"/board.png", "", outsider, tag); res.Code != 409 {
		t.Fatal("non-player got cached board status", res.Code)
	}
	if res := call("GET", base+"/state", "", nil, ""); res.Code != 401 {
		t.Fatal("anonymous state", res.Code)
	}
}
