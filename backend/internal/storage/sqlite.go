package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"drawguess/internal/game"
	_ "modernc.org/sqlite"
)

type Store struct {
	db    *sql.DB
	queue chan game.Result
	wg    sync.WaitGroup
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;
 CREATE TABLE IF NOT EXISTS matches (id TEXT PRIMARY KEY, finished_at INTEGER NOT NULL, data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS participants (match_id TEXT NOT NULL, player_id TEXT NOT NULL, PRIMARY KEY(match_id,player_id));
 CREATE INDEX IF NOT EXISTS participants_player ON participants(player_id);
 CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, id TEXT NOT NULL, name TEXT NOT NULL, expires INTEGER NOT NULL);`); err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{db: db, queue: make(chan game.Result, 256)}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for r := range s.queue {
			if err := s.save(r); err != nil {
				slog.Error("save result failed", "match", r.ID, "error", err)
			}
		}
	}()
	return s, nil
}
func (s *Store) Record(r game.Result) bool {
	select {
	case s.queue <- r:
		return true
	default:
		slog.Error("result queue full", "match", r.ID)
		return false
	}
}
func (s *Store) save(r game.Result) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("INSERT OR REPLACE INTO matches(id,finished_at,data) VALUES(?,?,?)", r.ID, r.FinishedAt, string(b)); err != nil {
		return err
	}
	for _, p := range r.Players {
		if _, err = tx.Exec("INSERT OR IGNORE INTO participants(match_id,player_id) VALUES(?,?)", r.ID, p.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) History(player string) ([]game.Result, error) {
	rows, err := s.db.Query(`SELECT m.data FROM matches m JOIN participants p ON m.id=p.match_id WHERE p.player_id=? ORDER BY m.finished_at DESC LIMIT 10`, player)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []game.Result{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		var r game.Result
		if err = json.Unmarshal([]byte(raw), &r); err != nil {
			return nil, fmt.Errorf("invalid stored result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
func (s *Store) Close() error { close(s.queue); s.wg.Wait(); return s.db.Close() }

type Session struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Expires time.Time `json:"-"`
}

func (s *Store) SaveSession(token string, value Session) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO sessions(token,id,name,expires) VALUES(?,?,?,?)", token, value.ID, value.Name, value.Expires.Unix())
	return err
}
func (s *Store) Session(token string) (Session, bool) {
	var value Session
	var expires int64
	err := s.db.QueryRow("SELECT id,name,expires FROM sessions WHERE token=? AND expires>?", token, time.Now().Unix()).Scan(&value.ID, &value.Name, &expires)
	value.Expires = time.Unix(expires, 0)
	return value, err == nil
}
func (s *Store) CleanupSessions(now time.Time) {
	if _, err := s.db.Exec("DELETE FROM sessions WHERE expires<?", now.Unix()); err != nil {
		slog.Error("cleanup sessions", "error", err)
	}
}
