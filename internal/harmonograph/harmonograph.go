// Package harmonograph draws the figure traced by a harmonograph: the sum
// of several damped sinusoids on each axis, the pendulums slowly running
// down so the curve spirals inward. It is pure line art and a pure function
// of its seed + params.
package harmonograph

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
	Pendulums  int     `json:"pendulums"`  // damped terms per axis, 1..4
	Damping    float64 `json:"damping"`    // decay rate per time unit
	FreqSpread float64 `json:"freqSpread"` // detuning of each frequency from its integer
	Duration   float64 `json:"duration"`   // time span traced
	Steps      int     `json:"steps"`      // samples along the curve
	LineAlpha  float64 `json:"lineAlpha"`  // per-sample ink opacity
	Colorful   bool    `json:"colorful"`   // sweep hue along the curve
	Background string  `json:"background"`
	Ink        string  `json:"ink"`
}

func defaults() params {
	return params{
		Pendulums: 2, Damping: 0.006, FreqSpread: 0.012,
		Duration: 220, Steps: 60000, LineAlpha: 0.06,
		Background: "#0e0e12", Ink: "#e9e6dc",
	}
}

type term struct{ amp, freq, phase, decay float64 }

func render(raw json.RawMessage, w, h int) (*image.RGBA, error) {
	p := defaults()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("harmonograph: bad params: %w", err)
		}
	}
	p.Pendulums = clampi(p.Pendulums, 1, 4)
	p.Damping = clampf(p.Damping, 0, 0.05)
	p.FreqSpread = clampf(p.FreqSpread, 0, 0.2)
	p.Duration = clampf(p.Duration, 20, 2000)
	p.Steps = clampi(p.Steps, 2000, 400000)
	p.LineAlpha = clampf(p.LineAlpha, 0.005, 1)

	rng := rand.New(rand.NewSource(p.Seed))
	randn := func() float64 { // Box–Muller
		u1 := math.Max(1e-12, rng.Float64())
		u2 := rng.Float64()
		return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	}
	mkAxis := func() []term {
		ts := make([]term, p.Pendulums)
		for i := range ts {
			base := float64(1 + rng.Intn(4)) // small integer frequency
			ts[i] = term{
				amp:   0.5 + rng.Float64()*0.8,
				freq:  base + p.FreqSpread*randn(),
				phase: rng.Float64() * 2 * math.Pi,
				decay: p.Damping * (0.6 + rng.Float64()*0.8),
			}
		}
		return ts
	}
	xs, ys := mkAxis(), mkAxis()

	sum := func(ts []term, t float64) float64 {
		var v float64
		for _, k := range ts {
			v += k.amp * math.Sin(k.freq*t+k.phase) * math.Exp(-k.decay*t)
		}
		return v
	}

	ampX, ampY := 0.0, 0.0
	for _, k := range xs {
		ampX += k.amp
	}
	for _, k := range ys {
		ampY += k.amp
	}
	span := math.Max(ampX, ampY) * 2
	scale := 0.92 * math.Min(float64(w), float64(h)) / math.Max(span, 1e-6)
	cx, cy := float64(w)/2, float64(h)/2

	bg := generators.ParseHex(p.Background, color.RGBA{14, 14, 18, 255})
	ink := generators.ParseHex(p.Ink, color.RGBA{233, 230, 220, 255})
	img := generators.NewCanvas(w, h, bg)

	dt := p.Duration / float64(p.Steps)
	px, py := cx+sum(xs, 0)*scale, cy+sum(ys, 0)*scale
	for i := 1; i <= p.Steps; i++ {
		t := float64(i) * dt
		nx := cx + sum(xs, t)*scale
		ny := cy + sum(ys, t)*scale
		col := ink
		if p.Colorful {
			r, g, b := hsv(0.6*t/p.Duration+0.55, 0.55, 1)
			col = color.RGBA{u8(r), u8(g), u8(b), 255}
		}
		generators.Line(img, px, py, nx, ny, p.LineAlpha, col)
		px, py = nx, ny
	}
	return img, nil
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

func init() { generators.Register("harmonograph", render) }
