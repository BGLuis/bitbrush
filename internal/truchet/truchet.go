// Package truchet renders multi-scale Truchet tilings: a grid of square
// tiles, each carrying one motif in a randomly chosen orientation, so the
// motifs join across edges into a single sprawling pattern. Optional
// recursive subdivision gives the "multi-scale" look.
//
// Deterministic: a given seed + params always produces identical pixels.
package truchet

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"

	"bitbrush/internal/generators"
)

type params struct {
	Seed       int64   `json:"seed"`
	Tiles      int     `json:"tiles"`      // cells across the width
	Style      string  `json:"style"`      // arcs | lines | maze | triangles
	LineWidth  float64 `json:"lineWidth"`  // stroke half-width as a fraction of the cell
	MultiScale bool    `json:"multiScale"` // recursively subdivide some cells
	Colorful   bool    `json:"colorful"`   // tint each tile a different hue
	Background string  `json:"background"`
	Ink        string  `json:"ink"`
}

func defaults() params {
	return params{
		Tiles: 12, Style: "arcs", LineWidth: 0.18,
		Background: "#141414", Ink: "#f2efe6",
	}
}

func render(raw json.RawMessage, w, h int) (*image.RGBA, error) {
	p := defaults()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("truchet: bad params: %w", err)
		}
	}
	p.Tiles = clampi(p.Tiles, 2, 64)
	p.LineWidth = clampf(p.LineWidth, 0.03, 0.5)

	bg := generators.ParseHex(p.Background, color.RGBA{20, 20, 20, 255})
	ink := generators.ParseHex(p.Ink, color.RGBA{242, 239, 230, 255})
	img := generators.NewCanvas(w, h, bg)

	cell := float64(w) / float64(p.Tiles)
	rows := int(math.Ceil(float64(h)/cell)) + 1
	rng := rand.New(rand.NewSource(p.Seed))

	var draw func(x, y, size float64, depth int)
	draw = func(x, y, size float64, depth int) {
		if p.MultiScale && depth < 2 && rng.Float64() < 0.32 {
			half := size / 2
			draw(x, y, half, depth+1)
			draw(x+half, y, half, depth+1)
			draw(x, y+half, half, depth+1)
			draw(x+half, y+half, half, depth+1)
			return
		}
		orient := rng.Intn(2)
		col := ink
		if p.Colorful {
			col = tint(rng.Float64())
		}
		drawTile(img, p.Style, x, y, size, orient, p.LineWidth, col, rng)
	}

	for j := 0; j < rows; j++ {
		for i := 0; i < p.Tiles; i++ {
			draw(float64(i)*cell, float64(j)*cell, cell, 0)
		}
	}
	return img, nil
}

func drawTile(img *image.RGBA, style string, x, y, s float64, orient int, lw float64, c color.RGBA, rng *rand.Rand) {
	hw := math.Max(0.6, lw*s*0.5)
	switch style {
	case "lines":
		if orient == 0 {
			stroke(img, x, y, x+s, y+s, hw, c)
		} else {
			stroke(img, x+s, y, x, y+s, hw, c)
		}
	case "maze":
		if orient == 0 {
			stroke(img, x+s/2, y, x+s/2, y+s, hw, c)
		} else {
			stroke(img, x, y+s/2, x+s, y+s/2, hw, c)
		}
	case "triangles":
		fillTri(img, x, y, s, rng.Intn(4), c)
	default: // arcs
		r := s / 2
		if orient == 0 {
			arc(img, x, y, r, 0, math.Pi/2, hw, c)
			arc(img, x+s, y+s, r, math.Pi, 1.5*math.Pi, hw, c)
		} else {
			arc(img, x+s, y, r, math.Pi/2, math.Pi, hw, c)
			arc(img, x, y+s, r, 1.5*math.Pi, 2*math.Pi, hw, c)
		}
	}
}

func arc(img *image.RGBA, ccx, ccy, r, a0, a1, hw float64, c color.RGBA) {
	steps := int(math.Abs(a1-a0)*r) + 2
	for i := 0; i <= steps; i++ {
		t := a0 + (a1-a0)*float64(i)/float64(steps)
		ct, st := math.Cos(t), math.Sin(t)
		for rr := r - hw; rr <= r+hw; rr += 0.75 {
			generators.SplatAA(img, ccx+rr*ct, ccy+rr*st, 1, c)
		}
	}
}

func stroke(img *image.RGBA, x0, y0, x1, y1, hw float64, c color.RGBA) {
	dx, dy := x1-x0, y1-y0
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	nx, ny := -dy/l, dx/l
	for off := -hw; off <= hw; off += 0.75 {
		generators.Line(img, x0+nx*off, y0+ny*off, x1+nx*off, y1+ny*off, 1, c)
	}
}

func fillTri(img *image.RGBA, x, y, s float64, corner int, c color.RGBA) {
	n := int(s)
	for py := 0; py < n; py++ {
		for px := 0; px < n; px++ {
			u := (float64(px) + 0.5) / s
			v := (float64(py) + 0.5) / s
			var in bool
			switch corner {
			case 0:
				in = u+v <= 1
			case 1:
				in = u+v >= 1
			case 2:
				in = u >= v
			default:
				in = u <= v
			}
			if in {
				generators.SplatAA(img, x+float64(px)+0.5, y+float64(py)+0.5, 1, c)
			}
		}
	}
}

func tint(hue float64) color.RGBA {
	r, g, b := hsv(hue, 0.5, 0.95)
	return color.RGBA{u8(r), u8(g), u8(b), 255}
}

func hsv(h, s, v float64) (float64, float64, float64) {
	h = math.Mod(math.Mod(h, 1)+1, 1) * 6
	i := math.Floor(h)
	f := h - i
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))
	switch int(i) % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}

func u8(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 255
	}
	return uint8(v*255 + 0.5)
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func init() { generators.Register("truchet", render) }
