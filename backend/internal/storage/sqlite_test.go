package storage

import (
	"path/filepath"
	"testing"
	"time"

	"drawguess/internal/game"
)

func TestResultsPersistAndHistoryIsScoped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Record(game.Result{ID: "match1", Room: "ROOM", FinishedAt: 123, Players: []game.ResultPlayer{{ID: "alice", Name: "阿狸", Score: 500}}}) {
		t.Fatal("enqueue failed")
	}
	if err = s.SaveSession("token", Session{ID: "alice", Name: "阿狸", Expires: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if identity, ok := s.Session("token"); !ok || identity.ID != "alice" {
		t.Fatal("identity lost after restart")
	}
	results, err := s.History("alice")
	if err != nil || len(results) != 1 || results[0].Players[0].Score != 500 {
		t.Fatalf("persist failed: %v %v", results, err)
	}
	other, err := s.History("bob")
	if err != nil || len(other) != 0 {
		t.Fatal("unrelated player can access results")
	}
}
