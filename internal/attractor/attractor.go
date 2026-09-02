// Package attractor plots 2-D strange attractors (De Jong, Clifford,
// Svensson) as a log-density image: millions of iterations of a simple
// nonlinear map, each landing point brightening one pixel. Deterministic —
// the map is fixed by its four coefficients, which are either given or
// derived from the seed.
package attractor

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
	Type         string  `json:"type"` // dejong | clifford | svensson
	Seed         int64   `json:"seed"`
	A            float64 `json:"a"`
	B            float64 `json:"b"`
	C            float64 `json:"c"`
	D            float64 `json:"d"`
	Iterations   int     `json:"iterations"`
	Gamma        float64 `json:"gamma"`
	Zoom         float64 `json:"zoom"`
	ColorBySpeed bool    `json:"colorBySpeed"`
	Background   string  `json:"background"`
	Ink          string  `json:"ink"`
}

func defaults() params {
	return params{
		Type: "clifford", Iterations: 2_000_000, Gamma: 2.2, Zoom: 1,
		Background: "#07070b", Ink: "#efe7d8",
	}
}

func render(raw json.RawMessage, w, h int) (*image.RGBA, error) {
	p := defaults()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("attractor: bad params: %w", err)
		}
	}
	p.Iterations = clampi(p.Iterations, 50_000, 20_000_000)
	p.Gamma = clampf(p.Gamma, 0.5, 6)
	p.Zoom = clampf(p.Zoom, 0.2, 5)

	// Coefficients: use the given ones unless all four are zero, in which
	// case derive a set from the seed.
	if p.A == 0 && p.B == 0 && p.C == 0 && p.D == 0 {
		rng := rand.New(rand.NewSource(p.Seed))
		r := func() float64 { return (rng.Float64()*2 - 1) * 2.4 }
		p.A, p.B, p.C, p.D = r(), r(), r(), r()
	}

	next := mapFunc(p.Type, p.A, p.B, p.C, p.D)

	// Warm up past the transient, then measure the trajectory's extent.
	x, y := 0.1, 0.1
	for i := 0; i < 2000; i++ {
		x, y = next(x, y)
		if !finite(x) || !finite(y) {
			return nil, fmt.Errorf("attractor: diverged during warm-up (coeffs out of range)")
		}
	}
	minX, minY, maxX, maxY := x, y, x, y
	bx, by := x, y
	for i := 0; i < 40_000; i++ {
		bx, by = next(bx, by)
		if !finite(bx) || !finite(by) {
			return nil, fmt.Errorf("attractor: trajectory diverged")
		}
		minX, maxX = math.Min(minX, bx), math.Max(maxX, bx)
		minY, maxY = math.Min(minY, by), math.Max(maxY, by)
	}
	spanX := math.Max(maxX-minX, 1e-6)
	spanY := math.Max(maxY-minY, 1e-6)
	cxData := (minX + maxX) / 2
	cyData := (minY + maxY) / 2
	scale := p.Zoom * 0.92 * math.Min(float64(w)/spanX, float64(h)/spanY)
	cx, cy := float64(w)/2, float64(h)/2

	acc := generators.NewAccumulator(w, h)
	rx, ry := x, y // same post-warm-up start as the bounds pass
	px, py := x, y
	for i := 0; i < p.Iterations; i++ {
		rx, ry = next(rx, ry)
		sx := cx + (rx-cxData)*scale
		sy := cy + (ry-cyData)*scale
		ix, iy := int(sx), int(sy)
		if p.ColorBySpeed {
			sp := math.Hypot(rx-px, ry-py)
			r, g, b := hsv(0.66-math.Min(sp*0.7, 0.66), 0.72, 1)
			acc.Add(ix, iy, r, g, b)
		} else {
			acc.Add(ix, iy, 0, 0, 0)
		}
		px, py = rx, ry
	}

	bg := generators.ParseHex(p.Background, color.RGBA{7, 7, 11, 255})
	ink := generators.ParseHex(p.Ink, color.RGBA{239, 231, 216, 255})
	vib := 0.0
	if p.ColorBySpeed {
		vib = 1
	}
	return acc.ToImage(bg, ink, p.Gamma, vib), nil
}

func mapFunc(kind string, a, b, c, d float64) func(x, y float64) (float64, float64) {
	switch kind {
	case "dejong":
		return func(x, y float64) (float64, float64) {
			return math.Sin(a*y) - math.Cos(b*x), math.Sin(c*x) - math.Cos(d*y)
		}
	case "svensson":
		return func(x, y float64) (float64, float64) {
			return d*math.Sin(a*x) - math.Sin(b*y), c*math.Cos(a*x) + math.Cos(b*y)
		}
	default: // clifford
		return func(x, y float64) (float64, float64) {
			return math.Sin(a*y) + c*math.Cos(a*x), math.Sin(b*x) + d*math.Cos(b*y)
		}
	}
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && math.Abs(v) < 1e6
}

func hsv(h, s, v float64) (float64, float64, float64) {
	h = math.Mod(math.Mod(h, 1)+1, 1) * 6
	i := math.Floor(h)
	f := h - i
	pp := v * (1 - s)
	q := v * (1 - s*f)
	tt := v * (1 - s*(1-f))
	switch int(i) % 6 {
	case 0:
		return v, tt, pp
	case 1:
		return q, v, pp
	case 2:
		return pp, v, tt
	case 3:
		return pp, q, v
	case 4:
		return tt, pp, v
	default:
		return v, pp, q
	}
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

func init() { generators.Register("attractor", render) }
