package httpapi

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"

	"drawguess/internal/game"
)

// Each stroke is the union of its round endpoints and the rectangles
// between them. Scan only intersecting rows, including for long diagonals.
func renderBoard(strokes []game.Stroke) *image.RGBA {
	const w, h = 500, 325
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
	type xy struct{ x, y float64 }
	for _, s := range strokes {
		v, _ := strconv.ParseUint(s.Color[1:], 16, 24)
		ink := image.NewUniform(color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255})
		if s.Tool == "eraser" {
			ink = image.White
		}
		radius := max(0.75, float64(s.Width)/4)
		span := func(y int, lo, hi float64) {
			start := int(math.Ceil(lo - .5))
			end := int(math.Floor(hi-.5)) + 1
			if end > start {
				draw.Draw(img, image.Rect(start, y, end, y+1), ink, image.Point{}, draw.Src)
			}
		}
		var prev xy
		for i, pt := range s.Points {
			p := xy{float64(pt.X) * w / 10000, float64(pt.Y) * h / 10000}
			for row := max(0, int(math.Floor(p.y-radius))); row < min(h, int(math.Ceil(p.y+radius))); row++ {
				dy := float64(row) + .5 - p.y
				if math.Abs(dy) > radius {
					continue
				}
				half := math.Sqrt(radius*radius - dy*dy)
				span(row, p.x-half, p.x+half)
			}
			if i > 0 && prev != p {
				dx, dy := p.x-prev.x, p.y-prev.y
				scale := radius / math.Hypot(dx, dy)
				nx, ny := -dy*scale, dx*scale
				corners := [4]xy{{prev.x + nx, prev.y + ny}, {p.x + nx, p.y + ny}, {p.x - nx, p.y - ny}, {prev.x - nx, prev.y - ny}}
				for row := max(0, int(math.Floor(min(prev.y, p.y)-math.Abs(ny)))); row < min(h, int(math.Ceil(max(prev.y, p.y)+math.Abs(ny)))); row++ {
					y := float64(row) + .5
					lo, hi := math.Inf(1), math.Inf(-1)
					for j, a := range corners {
						b := corners[(j+1)%4]
						if y < min(a.y, b.y) || y >= max(a.y, b.y) {
							continue
						}
						x := a.x + (b.x-a.x)*(y-a.y)/(b.y-a.y)
						lo, hi = min(lo, x), max(hi, x)
					}
					if lo <= hi {
						span(row, lo, hi)
					}
				}
			}
			prev = p
		}
	}
	return img
}
