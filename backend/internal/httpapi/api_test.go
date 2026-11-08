package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"drawguess/internal/game"
	"drawguess/internal/storage"
	"github.com/coder/websocket"
)

func TestHTTPAndWebSocketFlow(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	hub := game.NewHub(store)
	api := New(hub, store, false, false)
	server := httptest.NewServer(api.Handler())
	defer server.Close()
	defer store.Close()
	defer hub.Close()
	makePlayer := func(name string) *http.Client {
		jar, _ := cookiejar.New(nil)
		c := &http.Client{Jar: jar}
		resp, e := c.Post(server.URL+"/api/session", "application/json", strings.NewReader(`{"name":"`+name+`"}`))
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatal(resp.Status)
		}
		return c
	}
	alice := makePlayer("alice")
	bob := makePlayer("bob")
	response, err := alice.Post(server.URL+"/api/rooms", "application/json", strings.NewReader(`{"rounds":1,"drawSeconds":30}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var room struct{ ID string }
	if err = json.NewDecoder(response.Body).Decode(&room); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dial := func(c *http.Client) *websocket.Conn {
		conn, _, e := websocket.Dial(ctx, server.URL+"/ws?room="+room.ID, &websocket.DialOptions{HTTPClient: c})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { conn.CloseNow() })
		return conn
	}
	a, b := dial(alice), dial(bob)
	waitState := func(c *websocket.Conn, predicate func(game.State) bool) game.State {
		for {
			_, data, e := c.Read(ctx)
			if e != nil {
				t.Fatal(e)
			}
			var f struct{ State *game.State }
			if e = json.Unmarshal(data, &f); e != nil {
				t.Fatal(e)
			}
			if f.State != nil && predicate(*f.State) {
				return *f.State
			}
		}
	}
	waitState(a, func(s game.State) bool { return len(s.Players) == 2 })
	waitState(b, func(s game.State) bool { return len(s.Players) == 2 })
	if err = a.Write(ctx, websocket.MessageText, []byte(`{"type":"start"}`)); err != nil {
		t.Fatal(err)
	}
	choice := waitState(a, func(s game.State) bool { return s.Phase == "choosing" })
	guest := waitState(b, func(s game.State) bool { return s.Phase == "choosing" })
	if len(guest.Choices) != 0 || len(choice.Choices) != 3 {
		t.Fatal("choice isolation failed over transport")
	}
	msg, _ := json.Marshal(game.Command{Type: "choose", Turn: choice.Turn, Index: 0})
	_ = a.Write(ctx, websocket.MessageText, msg)
	drawing := waitState(a, func(s game.State) bool { return s.Phase == "drawing" })
	guest = waitState(b, func(s game.State) bool { return s.Phase == "drawing" })
	if drawing.Word == "" || guest.Word != "" {
		t.Fatal("word isolation failed over transport")
	}
	msg, _ = json.Marshal(game.Command{Type: "guess", Text: drawing.Word})
	_ = b.Write(ctx, websocket.MessageText, msg)
	reveal := waitState(b, func(s game.State) bool { return s.Phase == "reveal" })
	if reveal.RevealedWord != drawing.Word {
		t.Fatal("wrong reveal")
	}
	req, _ := http.NewRequest("POST", server.URL+"/api/session", strings.NewReader(`{"name":"evil"}`))
	req.Header.Set("Origin", "https://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatal("cross-origin mutation accepted")
	}
	_, resp, err = websocket.Dial(ctx, server.URL+"/ws?room="+room.ID, &websocket.DialOptions{HTTPClient: alice, HTTPHeader: http.Header{"Origin": []string{"https://evil.example"}}})
	if err == nil || resp.StatusCode != 403 {
		t.Fatal("cross-origin websocket accepted")
	}
}
