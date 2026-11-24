package httpapi

import (
	"image/color"
	"testing"

	"drawguess/internal/game"
)

func TestBoardIncludesPenDotsAndEraser(t *testing.T) {
	img := renderBoard([]game.Stroke{
		{Tool: "pen", Color: "#ff0000", Width: 20, Points: []game.Point{{X: 1000, Y: 4000}, {X: 9000, Y: 4000}}},
		{Tool: "eraser", Color: "#000000", Width: 32, Points: []game.Point{{X: 5000, Y: 4000}}},
		{Tool: "pen", Color: "#0000ff", Width: 8, Points: []game.Point{{X: 9000, Y: 8000}}},
	})
	for xy, want := range map[[2]int]color.RGBA{
		{100, 130}: {255, 0, 0, 255}, {250, 130}: {255, 255, 255, 255}, {450, 260}: {0, 0, 255, 255}, {50, 200}: {255, 255, 255, 255},
	} {
		if got := img.RGBAAt(xy[0], xy[1]); got != want {
			t.Errorf("%v: got %v, want %v", xy, got, want)
		}
	}
}
