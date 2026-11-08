// loadtest exercises the real room engine and WebSocket transport on loopback.
// It deliberately bypasses HTTP admission throttles; it does not expose a public server.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"drawguess/internal/game"
	"github.com/coder/websocket"
)

type metrics struct {
	mu        sync.Mutex
	latencies []float64
	received  atomic.Int64
	bytes     atomic.Int64
	failures  atomic.Int64
}
type senderTimes struct {
	mu    sync.Mutex
	times map[int]time.Time
}
type peer struct {
	conn   *websocket.Conn
	states chan game.State
}
type testRoom struct {
	peers []*peer
	times *senderTimes
	turn  int
}

func main() {
	rooms := flag.Int("rooms", 10, "number of active rooms")
	players := flag.Int("players", 8, "players per room (2-8)")
	batches := flag.Int("batches", 100, "batches per room (1-500)")
	interval := flag.Duration("interval", 40*time.Millisecond, "time between stroke batches")
	flag.Parse()
	if *rooms < 1 || *rooms > 1000 || *players < 2 || *players > 8 || *batches < 1 || *batches > 500 || *interval < 15*time.Millisecond {
		fmt.Fprintln(os.Stderr, "invalid workload")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	hub := game.NewHub(nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r, r.URL.Query().Get("player"), r.URL.Query().Get("player"))
	}))
	baseGoroutines := runtime.NumGoroutine()
	var m metrics
	var readers sync.WaitGroup
	var all []testRoom
	for i := 0; i < *rooms; i++ {
		host := fmt.Sprintf("p%d_0", i)
		id, err := hub.Create(host, game.Settings{Rounds: 1, DrawSeconds: 180})
		check(err)
		tr := testRoom{times: &senderTimes{times: map[int]time.Time{}}}
		for j := 0; j < *players; j++ {
			conn, _, err := websocket.Dial(ctx, fmt.Sprintf("%s/ws?room=%s&player=p%d_%d", server.URL, id, i, j), nil)
			check(err)
			conn.SetReadLimit(4 * 1024 * 1024)
			p := &peer{conn: conn, states: make(chan game.State, 64)}
			tr.peers = append(tr.peers, p)
			times := tr.times
			readers.Add(1)
			go func() {
				defer readers.Done()
				for {
					_, data, err := conn.Read(ctx)
					if err != nil {
						return
					}
					m.bytes.Add(int64(len(data)))
					var f struct {
						Type    string      `json:"type"`
						State   *game.State `json:"state"`
						Seq     int         `json:"seq"`
						Message string      `json:"message"`
					}
					if json.Unmarshal(data, &f) != nil {
						m.failures.Add(1)
						continue
					}
					if f.Type == "error" || f.Type == "fatal" {
						m.failures.Add(1)
					}
					if f.State != nil {
						select {
						case p.states <- *f.State:
						default:
						}
					}
					if f.Type == "stroke" {
						times.mu.Lock()
						sent := times.times[f.Seq]
						times.mu.Unlock()
						if !sent.IsZero() {
							latency := time.Since(sent).Seconds() * 1000
							m.mu.Lock()
							m.latencies = append(m.latencies, latency)
							m.mu.Unlock()
							m.received.Add(1)
						}
					}
				}
			}()
		}
		waitState(ctx, tr.peers[0], func(s game.State) bool { return len(s.Players) == *players })
		write(ctx, tr.peers[0], game.Command{Type: "start"})
		choice := waitState(ctx, tr.peers[0], func(s game.State) bool { return s.Phase == "choosing" })
		write(ctx, tr.peers[0], game.Command{Type: "choose", Turn: choice.Turn, Index: 0})
		waitState(ctx, tr.peers[0], func(s game.State) bool { return s.Phase == "drawing" })
		tr.turn = choice.Turn
		all = append(all, tr)
	}
	cpuStart := cpuSeconds()
	start := time.Now()
	var writers sync.WaitGroup
	for _, tr := range all {
		writers.Add(1)
		go func(tr testRoom) {
			defer writers.Done()
			ticker := time.NewTicker(*interval)
			defer ticker.Stop()
			for seq := 1; seq <= *batches; seq++ {
				select {
				case <-ticker.C:
				case <-ctx.Done():
					m.failures.Add(1)
					return
				}
				points := make([]game.Point, 24)
				for k := range points {
					points[k] = game.Point{X: (seq*10 + k*100) % 10000, Y: (seq*10 + k*90) % 10000}
				}
				tr.times.mu.Lock()
				tr.times.times[seq] = time.Now()
				tr.times.mu.Unlock()
				data, _ := json.Marshal(game.Command{Type: "stroke", Turn: tr.turn, Stroke: game.Stroke{ID: "load-stroke", Color: "#292d2b", Width: 6, Tool: "pen", Points: points}})
				if err := tr.peers[0].conn.Write(ctx, websocket.MessageText, data); err != nil {
					m.failures.Add(1)
					return
				}
			}
		}(tr)
	}
	writers.Wait()
	expected := int64(*rooms * *players * *batches)
	deadline := time.Now().Add(15 * time.Second)
	for m.received.Load() < expected && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	elapsed := time.Since(start)
	cpu := cpuSeconds() - cpuStart
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	m.mu.Lock()
	sort.Float64s(m.latencies)
	p50, p95, p99 := percentile(m.latencies, .50), percentile(m.latencies, .95), percentile(m.latencies, .99)
	m.mu.Unlock()
	for _, tr := range all {
		for _, p := range tr.peers {
			p.conn.CloseNow()
		}
	}
	hub.Close()
	server.Close()
	readers.Wait()
	time.Sleep(100 * time.Millisecond)
	output := map[string]any{"rooms": *rooms, "connections": *rooms * *players, "batches_per_room": *batches, "interval_ms": interval.Milliseconds(), "expected_deliveries": expected, "received_deliveries": m.received.Load(), "errors": m.failures.Load(), "elapsed_seconds": elapsed.Seconds(), "p50_ms": p50, "p95_ms": p95, "p99_ms": p99, "heap_mib": float64(mem.HeapAlloc) / 1024 / 1024, "outbound_mib_per_second": float64(m.bytes.Load()) / 1024 / 1024 / elapsed.Seconds(), "process_cpu_seconds": cpu, "goroutines_before": baseGoroutines, "goroutines_after_cleanup": runtime.NumGoroutine(), "scope": "loopback room engine + WebSocket; includes generator/clients; excludes HTTP admission, TLS, database and public network"}
	b, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(b))
	if m.received.Load() != expected || m.failures.Load() != 0 {
		os.Exit(1)
	}
}
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func write(ctx context.Context, p *peer, c game.Command) {
	b, _ := json.Marshal(c)
	check(p.conn.Write(ctx, websocket.MessageText, b))
}
func waitState(ctx context.Context, p *peer, predicate func(game.State) bool) game.State {
	for {
		select {
		case s := <-p.states:
			if predicate(s) {
				return s
			}
		case <-ctx.Done():
			check(ctx.Err())
		}
	}
}
func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return values[min(len(values)-1, int(float64(len(values))*p))]
}
