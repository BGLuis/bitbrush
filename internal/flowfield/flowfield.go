// Package flowfield paints thousands of ink strokes that follow an unseen
// current — a two-octave simplex noise field (or its curl) — each stroke
// thinning as its bristles run dry. A deterministic port of the "Sumi
// Field" reference sketch.
package flowfield

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
	Turbulence float64 `json:"turbulence"` // 0.6..5, scales the field angle
	Density    float64 `json:"density"`    // 0.25..3, stroke-count multiplier
	Curl       bool    `json:"curl"`       // divergence-free curl field instead of raw angle
	Palette    string  `json:"palette"`    // ink | indigo | vermilion
	Grain      float64 `json:"grain"`      // 0..1 paper grain amount
	Background string  `json:"background"` // optional override of the palette paper
	Time       float64 `json:"time"`       // temporal offset for continuous animation
}

func defaults() params {
	return params{Turbulence: 2.4, Density: 1, Palette: "ink", Grain: 0.4}
}

type layer struct {
	count    int
	steps    int
	step     float64
	wLo, wHi float64
	aLo, aHi float64
	dry      float64
	useWash  bool
}

func render(raw json.RawMessage, w, h int) (*image.RGBA, error) {
	p := defaults()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("flowfield: bad params: %w", err)
		}
	}
	p.Turbulence = clampf(p.Turbulence, 0.2, 8)
	p.Density = clampf(p.Density, 0.15, 4)
	p.Grain = clampf(p.Grain, 0, 1)

	pal := paletteFor(p.Palette)
	paper := pal.paper
	if p.Background != "" {
		paper = generators.ParseHex(p.Background, paper)
	}

	rng := rand.New(rand.NewSource(p.Seed))
	noise := newSimplex(rand.New(rand.NewSource(p.Seed*7 + 1)))

	img := generators.NewCanvas(w, h, paper)
	if p.Grain > 0 {
		applyGrain(img, p.Seed, p.Grain)
	}

	// Field heading at a point.
	const f = 0.0022
	t := p.Time * 0.15
	angleAt := func(x, y float64) float64 {
		if p.Curl {
			const e = 1.0
			phi := func(px, py float64) float64 {
				return noise.noise2(px*f+t*0.5, py*f+t*0.3)*0.7 + noise.noise2(px*f*2.3+31.7-t*0.4, py*f*2.3-12.1+t*0.2)*0.3
			}
			dx := phi(x+e, y) - phi(x-e, y)
			dy := phi(x, y+e) - phi(x, y-e)
			return math.Atan2(-dx, dy) // curl: rotate the gradient 90 degrees
		}
		n := noise.noise2(x*f+t*0.5, y*f+t*0.3)*0.7 + noise.noise2(x*f*2.3+31.7-t*0.4, y*f*2.3-12.1+t*0.2)*0.3
		return n * math.Pi * p.Turbulence
	}

	area := float64(w*h) / (1280 * 800) * p.Density
	layers := []layer{
		{count: iround(260 * area), steps: 36, step: 4.5, wLo: 7, wHi: 16, aLo: 0.025, aHi: 0.06, dry: 0.05, useWash: true},
		{count: iround(1400 * area), steps: 24, step: 3.2, wLo: 1.4, wHi: 4, aLo: 0.12, aHi: 0.3, dry: 0.12},
		{count: iround(2600 * area), steps: 14, step: 2.6, wLo: 0.35, wHi: 1.1, aLo: 0.3, aHi: 0.7, dry: 0.22},
	}

	fw, fh := float64(w), float64(h)
	for _, L := range layers {
		for s := 0; s < L.count; s++ {
			x := rng.Float64() * fw
			y := rng.Float64() * fh
			width := L.wLo + rng.Float64()*(L.wHi-L.wLo)
			alpha := L.aLo + rng.Float64()*(L.aHi-L.aLo)
			col := pal.tone(rng, L.useWash)
			for k := 0; k < L.steps; k++ {
				t := float64(k) / float64(L.steps)
				ang := angleAt(x, y)
				nx := x + math.Cos(ang)*L.step
				ny := y + math.Sin(ang)*L.step
				if nx < -20 || ny < -20 || nx > fw+20 || ny > fh+20 {
					break
				}
				if rng.Float64() >= L.dry*(0.3+t) { // not a dry-bristle gap
					wd := width * (1 - t*0.85) * (0.85 + rng.Float64()*0.3)
					thickStroke(img, x, y, nx, ny, wd, alpha*(1-t*0.55), col)
				} else {
					rng.Float64() // keep the RNG draw count identical to the non-gap branch
				}
				x, y = nx, ny
			}
		}
	}
	return img, nil
}

func thickStroke(img *image.RGBA, x0, y0, x1, y1, width, alpha float64, c color.RGBA) {
	dx, dy := x1-x0, y1-y0
	l := math.Hypot(dx, dy)
	if l == 0 {
		generators.SplatAA(img, x0, y0, alpha, c)
		return
	}
	nx, ny := -dy/l, dx/l
	hw := math.Max(0.5, width*0.5)
	for off := -hw; off <= hw; off += 0.85 {
		generators.Line(img, x0+nx*off, y0+ny*off, x1+nx*off, y1+ny*off, alpha, c)
	}
}

func applyGrain(img *image.RGBA, seed int64, amt float64) {
	rng := rand.New(rand.NewSource(seed ^ 0x9e3779b9))
	k := amt * 16
	for i := 0; i < len(img.Pix); i += 4 {
		g := (rng.Float64() - 0.5) * k
		img.Pix[i] = u8b(float64(img.Pix[i]) + g)
		img.Pix[i+1] = u8b(float64(img.Pix[i+1]) + g*0.95)
		img.Pix[i+2] = u8b(float64(img.Pix[i+2]) + g*0.85)
	}
}

type palette struct {
	paper color.RGBA
	tones []color.RGBA
	wash  color.RGBA
}

func (p palette) tone(rng *rand.Rand, wash bool) color.RGBA {
	if wash {
		return p.wash
	}
	return p.tones[rng.Intn(len(p.tones))]
}

func paletteFor(name string) palette {
	switch name {
	case "indigo":
		return palette{
			paper: color.RGBA{238, 240, 236, 255},
			tones: []color.RGBA{{21, 32, 74, 255}, {27, 42, 94, 255}, {47, 69, 144, 255}},
			wash:  color.RGBA{123, 140, 204, 255},
		}
	case "vermilion":
		return palette{
			paper: color.RGBA{247, 241, 230, 255},
			tones: []color.RGBA{{122, 30, 16, 255}, {168, 48, 26, 255}, {200, 69, 42, 255}},
			wash:  color.RGBA{224, 144, 111, 255},
		}
	default: // ink
		return palette{
			paper: color.RGBA{243, 238, 227, 255},
			tones: []color.RGBA{{22, 22, 26, 255}, {42, 42, 48, 255}, {61, 61, 69, 255}},
			wash:  color.RGBA{106, 106, 114, 255},
		}
	}
}

func iround(v float64) int { return int(v + 0.5) }

func u8b(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 255:
		return 255
	}
	return uint8(v + 0.5)
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

func init() { generators.Register("flowfield", render) }
