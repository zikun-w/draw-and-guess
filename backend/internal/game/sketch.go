package game

import (
	"math"
	"time"
)

// Each frame appends points to the same stroke just as a moving browser pen does.
type sketchFrame struct {
	stroke Stroke
	penUp  bool
}

type sketchPlayback struct {
	artist *client
	frames []sketchFrame
	delay  time.Duration
	next   time.Time
}

func sketchFrames(strokes []Stroke) ([]sketchFrame, int) {
	var frames []sketchFrame
	totalPoints := 0
	for _, s := range strokes {
		path := []Point{s.Points[0]}
		length := 0.0
		for i := 1; i < len(s.Points); i++ {
			a, b := s.Points[i-1], s.Points[i]
			distance := math.Hypot(float64(b.X-a.X)*0.1, float64(b.Y-a.Y)*0.065)
			length += distance
			steps := max(1, int(math.Ceil(distance/75)))
			for j := 1; j <= steps; j++ {
				path = append(path, Point{X: a.X + (b.X-a.X)*j/steps, Y: a.Y + (b.Y-a.Y)*j/steps})
			}
		}
		totalPoints += len(path)
		first := s
		first.Points = path[:1:1]
		frames = append(frames, sketchFrame{first, true})
		segments := max(1, min(14, int(math.Ceil(length/70))))
		size := max(1, min(96, (len(path)-1+segments-1)/segments))
		for i := 1; i < len(path); i += size {
			end := min(len(path), i+size)
			chunk := s
			chunk.Points = path[i:end:end]
			frames = append(frames, sketchFrame{chunk, false})
		}
	}
	return frames, totalPoints
}

func (r *room) startSketch(c *client, strokes []Stroke, now time.Time) AgentResult {
	frames, points := sketchFrames(strokes)
	if r.points+points > 24000 {
		return denied(409, "本轮画板已满")
	}
	// Include a short lift between strokes. Keep long plans inside the remaining turn.
	ticks := len(frames) + 2*(len(strokes)-1)
	budget := min(12*time.Second, max(time.Duration(0), r.deadline.Sub(now)-200*time.Millisecond))
	delay := min(100*time.Millisecond, budget/time.Duration(ticks))
	r.sketch = &sketchPlayback{artist: c, frames: frames, delay: delay, next: now}
	r.advanceSketch(now)
	return AgentResult{}
}

func (r *room) advanceSketch(now time.Time) {
	pen := r.sketch
	if pen == nil {
		return
	}
	drawer := r.player(r.drawer)
	if r.phase != "drawing" || !now.Before(r.deadline) || drawer == nil || drawer.client != pen.artist {
		r.sketch = nil
		return
	}
	for len(pen.frames) > 0 && !pen.next.After(now) {
		r.applyStroke(pen.artist, pen.frames[0].stroke)
		pen.frames = pen.frames[1:]
		if len(pen.frames) == 0 {
			r.sketch = nil
			return
		}
		pen.next = pen.next.Add(pen.delay)
		if pen.frames[0].penUp {
			pen.next = pen.next.Add(2 * pen.delay)
		}
	}
}
