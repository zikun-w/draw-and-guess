package game

import (
	"errors"
	"math"
)

// Coordinates use the web board's logical 1000 by 650 space.
type Shape struct {
	Kind   string  `json:"kind"`
	Points [][]int `json:"points"`
	Color  string  `json:"color"`
	Width  int     `json:"width"`
}

func makeStrokes(shapes []Shape) ([]Stroke, error) {
	if len(shapes) < 1 || len(shapes) > 16 {
		return nil, errors.New("一次提交 1–16 个图形")
	}
	strokes := make([]Stroke, 0, len(shapes))
	for _, shape := range shapes {
		if len(shape.Points) < 1 || len(shape.Points) > 96 {
			return nil, errors.New("每个图形需要 1–96 个点")
		}
		if shape.Width == 0 {
			shape.Width = 6
		}
		if shape.Color == "" {
			shape.Color = "#292d2b"
		}
		if shape.Width < 1 || shape.Width > 40 || !colorPattern.MatchString(shape.Color) {
			return nil, errors.New("颜色应为 #RRGGBB，线宽应为 1–40")
		}
		for _, p := range shape.Points {
			if len(p) != 2 || p[0] < 0 || p[0] > 1000 || p[1] < 0 || p[1] > 650 {
				return nil, errors.New("坐标格式为 [x,y]，范围为 0–1000、0–650")
			}
		}
		s := Stroke{ID: ID(), Color: shape.Color, Width: shape.Width, Tool: "pen"}
		add := func(x, y float64) {
			s.Points = append(s.Points, Point{int(math.Round(x * 10)), int(math.Round(y * 10000 / 650))})
		}
		switch shape.Kind {
		case "line":
			for _, p := range shape.Points {
				add(float64(p[0]), float64(p[1]))
			}
		case "rect", "ellipse":
			if len(shape.Points) != 2 {
				return nil, errors.New("矩形和椭圆需要外框的两个对角点")
			}
			a, b := shape.Points[0], shape.Points[1]
			x, y, dx, dy := float64(a[0]), float64(a[1]), float64(b[0]-a[0]), float64(b[1]-a[1])
			if dx == 0 || dy == 0 {
				return nil, errors.New("外框宽高必须大于零")
			}
			if shape.Kind == "rect" {
				add(x, y)
				add(x+dx, y)
				add(x+dx, y+dy)
				add(x, y+dy)
				add(x, y)
			} else {
				for i := 0; i <= 48; i++ {
					t := float64(i) * 2 * math.Pi / 48
					add(x+dx*(1+math.Cos(t))/2, y+dy*(1+math.Sin(t))/2)
				}
			}
		default:
			return nil, errors.New("kind 应为 line、rect 或 ellipse")
		}
		strokes = append(strokes, s)
	}
	return strokes, nil
}
