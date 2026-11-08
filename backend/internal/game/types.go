package game

import (
	"encoding/json"
	"time"
)

const MaxPlayers = 8

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}
type Stroke struct {
	ID     string  `json:"id"`
	Color  string  `json:"color"`
	Width  int     `json:"width"`
	Tool   string  `json:"tool"`
	Points []Point `json:"points"`
}
type Player struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Score        int    `json:"score"`
	Connected    bool   `json:"connected"`
	Guessed      bool   `json:"guessed"`
	client       *client
	disconnected time.Time
	lastGuess    time.Time
}
type Message struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Text string `json:"text"`
	Kind string `json:"kind"`
}
type Settings struct {
	Rounds      int `json:"rounds"`
	DrawSeconds int `json:"drawSeconds"`
}
type State struct {
	Version      int       `json:"version"`
	ID           string    `json:"id"`
	Phase        string    `json:"phase"`
	HostID       string    `json:"hostId"`
	DrawerID     string    `json:"drawerId"`
	Players      []Player  `json:"players"`
	Settings     Settings  `json:"settings"`
	Turn         int       `json:"turn"`
	TotalTurns   int       `json:"totalTurns"`
	Deadline     int64     `json:"deadline"`
	ServerTime   int64     `json:"serverTime"`
	WordLength   int       `json:"wordLength"`
	Word         string    `json:"word,omitempty"`
	Choices      []string  `json:"choices,omitempty"`
	RevealedWord string    `json:"revealedWord,omitempty"`
	Seq          int       `json:"seq"`
	Messages     []Message `json:"messages"`
}
type Command struct {
	Type     string   `json:"type"`
	Turn     int      `json:"turn"`
	Index    int      `json:"index"`
	Text     string   `json:"text"`
	Stroke   Stroke   `json:"stroke"`
	Settings Settings `json:"settings"`
}
type event struct {
	kind   string
	client *client
	cmd    Command
}
type frame struct {
	Type    string   `json:"type"`
	State   *State   `json:"state,omitempty"`
	Strokes []Stroke `json:"strokes,omitempty"`
	Stroke  *Stroke  `json:"stroke,omitempty"`
	Seq     int      `json:"seq,omitempty"`
	Turn    int      `json:"turn,omitempty"`
	Message string   `json:"message,omitempty"`
}

func encode(v any) []byte { b, _ := json.Marshal(v); return b }

type ResultPlayer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}
type Result struct {
	ID         string         `json:"id"`
	Room       string         `json:"room"`
	FinishedAt int64          `json:"finishedAt"`
	Players    []ResultPlayer `json:"players"`
}
type Recorder interface{ Record(Result) bool }
