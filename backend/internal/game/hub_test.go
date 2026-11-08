package game

import "testing"

func TestRoomLimit(t *testing.T) {
	h := NewHubWithLimit(nil, 1)
	defer h.Close()
	if _, err := h.Create("alice", Settings{1, 60}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Create("bob", Settings{1, 60}); err == nil {
		t.Fatal("room limit was not enforced")
	}
}
