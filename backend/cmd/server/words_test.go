package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"drawguess/internal/game"
)

func TestWordRefreshPreservesValidCache(t *testing.T) {
	const body = "风筝\n沙漏\n雪橇\n"
	var bank game.WordBank
	dir := t.TempDir()
	cache := filepath.Join(dir, "word-bank.txt")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github.raw+json" {
			t.Error("missing GitHub raw content negotiation")
		}
		if r.URL.Path == "/bad" {
			w.Write([]byte("not a word bank"))
		} else if r.Header.Get("If-None-Match") == `"good"` {
			w.WriteHeader(http.StatusNotModified)
		} else {
			w.Header().Set("ETag", `"good"`)
			w.Write([]byte(body))
		}
	}))
	defer srv.Close()
	ctx := context.Background()
	tag, err := refreshWords(ctx, srv.Client(), srv.URL, cache, "", &bank)
	if err != nil || tag != `"good"` {
		t.Fatalf("initial refresh: %q, %v", tag, err)
	}
	if next, err := refreshWords(ctx, srv.Client(), srv.URL, cache, tag, &bank); err != nil || next != tag {
		t.Fatalf("conditional refresh: %q, %v", next, err)
	}
	if _, err := refreshWords(ctx, srv.Client(), srv.URL+"/bad", cache, tag, &bank); err == nil {
		t.Fatal("accepted invalid update")
	}
	if saved, err := os.ReadFile(cache); err != nil || string(saved) != body {
		t.Fatalf("bad cache after failed update: %q, %v", saved, err)
	}
}
