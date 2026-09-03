// Package chladni synthesises 2D acoustic resonance patterns (Chladni figures)
// formed by standing vibrational waves on a flat plate. Particles drift toward
// nodal lines where vibration is zero, revealing cymatic harmonic geometry.
package chladni

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
	N          int     `json:"n"`          // harmonic mode along X (1..12)
	M          int     `json:"m"`          // harmonic mode along Y (1..12)
	A          float64 `json:"a"`          // mode coefficient a (0.1..4)
	B          float64 `json:"b"`          // mode coefficient b (0.1..4)
	Particles  int     `json:"particles"`  // sand grain count (10000..500000)
	Time       float64 `json:"time"`       // phase oscillation for live flow
	Glow       bool    `json:"glow"`       // render wave field interference
	Palette    string  `json:"palette"`    // sand | copper | monochrome | cyan
	Background string  `json:"background"` // optional hex override
	Ink        string  `json:"ink"`        // optional hex override
}

func defaults() params {
	return params{
		Seed:      42,
		N:         3,
		M:         5,
		A:         1.0,
		B:         1.0,
		Particles: 80000,
		Time:      0.0,
		Glow:      true,
		Palette:   "sand",
	}
}

type palette struct {
	bg   color.RGBA
	ink  color.RGBA
	wave color.RGBA
}

func paletteFor(name string) palette {
	switch name {
	case "copper":
		return palette{
			bg:   color.RGBA{24, 15, 12, 255},
			ink:  color.RGBA{245, 185, 130, 255},
			wave: color.RGBA{140, 60, 30, 255},
		}
	case "monochrome":
		return palette{
			bg:   color.RGBA{10, 10, 12, 255},
			ink:  color.RGBA{245, 245, 250, 255},
			wave: color.RGBA{70, 70, 80, 255},
		}
	case "cyan":
		return palette{
			bg:   color.RGBA{6, 14, 26, 255},
			ink:  color.RGBA{82, 224, 255, 255},
			wave: color.RGBA{20, 70, 110, 255},
		}
	default: // "sand"
		return palette{
			bg:   color.RGBA{22, 21, 20, 255},
			ink:  color.RGBA{225, 216, 196, 255},
			wave: color.RGBA{70, 65, 58, 255},
		}
	}
}

func init() {
	generators.Register("chladni", render)
}

func render(raw json.RawMessage, w, h int) (*image.RGBA, error) {
	p := defaults()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("chladni: bad params: %w", err)
		}
	}
	if p.N < 1 {
		p.N = 1
	} else if p.N > 12 {
		p.N = 12
	}
	if p.M < 1 {
		p.M = 1
	} else if p.M > 12 {
		p.M = 12
	}
	if p.A <= 0.01 {
		p.A = 1.0
	}
	if p.B <= 0.01 {
		p.B = 1.0
	}
	if p.Particles < 1000 {
		p.Particles = 1000
	} else if p.Particles > 500000 {
		p.Particles = 500000
	}

	pal := paletteFor(p.Palette)
	if p.Background != "" {
		pal.bg = generators.ParseHex(p.Background, pal.bg)
	}
	if p.Ink != "" {
		pal.ink = generators.ParseHex(p.Ink, pal.ink)
	}

	img := generators.NewCanvas(w, h, pal.bg)

	// Mode function: w(x, y) = a*cos(n*pi*x + phase)*cos(m*pi*y) - b*cos(m*pi*x)*cos(n*pi*y + phase)
	// on plate coordinates x in [-1, 1], y in [-1, 1]
	nPi := float64(p.N) * math.Pi * 0.5
	mPi := float64(p.M) * math.Pi * 0.5
	phase := p.Time * 0.5

	ampAt := func(x, y float64) float64 {
		return p.A*math.Cos(nPi*x+phase)*math.Cos(mPi*y) - p.B*math.Cos(mPi*x)*math.Cos(nPi*y+phase)
	}

	// 1. If Glow is enabled, paint faint standing wave interference field
	if p.Glow {
		wf := float64(w)
		hf := float64(h)
		pix := img.Pix
		stride := img.Stride
		br, bg, bb := float64(pal.bg.R), float64(pal.bg.G), float64(pal.bg.B)
		wr, wg, wb := float64(pal.wave.R), float64(pal.wave.G), float64(pal.wave.B)

		step := 2
		if w > 800 || h > 800 {
			step = 3
		}

		for py := 0; py < h; py += step {
			y := (float64(py)/hf)*2.0 - 1.0
			for px := 0; px < w; px += step {
				x := (float64(px)/wf)*2.0 - 1.0
				a := math.Abs(ampAt(x, y))
				// Wave interference intensity in [0, 1]
				t := math.Min(1.0, a*0.35)
				cr := uint8(br + (wr-br)*t)
				cg := uint8(bg + (wg-bg)*t)
				cb := uint8(bb + (wb-bb)*t)

				for dy := 0; dy < step && py+dy < h; dy++ {
					rowOff := (py + dy) * stride
					for dx := 0; dx < step && px+dx < w; dx++ {
						idx := rowOff + (px+dx)*4
						pix[idx] = cr
						pix[idx+1] = cg
						pix[idx+2] = cb
						pix[idx+3] = 255
					}
				}
			}
		}
	}

	// 2. Simulate sand grains collecting at nodal lines (where amp ≈ 0)
	rng := rand.New(rand.NewSource(p.Seed))
	pix := img.Pix
	stride := img.Stride
	ir, ig, ib := int(pal.ink.R), int(pal.ink.G), int(pal.ink.B)

	// To settle particles efficiently without expensive multi-step physics,
	// sample random positions and step down gradient towards the nodal surface (|amp| -> 0)
	wf := float64(w)
	hf := float64(h)
	eps := 0.005

	for i := 0; i < p.Particles; i++ {
		x := rng.Float64()*2.0 - 1.0
		y := rng.Float64()*2.0 - 1.0

		// Gradient descent toward nodal line
		for step := 0; step < 7; step++ {
			a := ampAt(x, y)
			absA := math.Abs(a)
			if absA < 0.02 {
				break
			}
			gradX := (math.Abs(ampAt(x+eps, y)) - math.Abs(ampAt(x-eps, y))) / (2 * eps)
			gradY := (math.Abs(ampAt(x, y+eps)) - math.Abs(ampAt(x, y-eps))) / (2 * eps)
			lenG := math.Hypot(gradX, gradY)
			if lenG > 1e-6 {
				stepSize := math.Min(absA*0.25, 0.08)
				x -= (gradX / lenG) * stepSize
				y -= (gradY / lenG) * stepSize
			}
		}

		// Add subtle grain jitter
		xj := x + (rng.Float64()-0.5)*0.008
		yj := y + (rng.Float64()-0.5)*0.008
		if xj < -0.98 || xj > 0.98 || yj < -0.98 || yj > 0.98 {
			continue
		}

		// Check if grain is settled on a node
		if math.Abs(ampAt(xj, yj)) > 0.12 {
			// Ejected grains scattered randomly with lower opacity
			if rng.Float64() > 0.08 {
				continue
			}
		}

		// Plot grain
		px := int((xj + 1.0) * 0.5 * wf)
		py := int((yj + 1.0) * 0.5 * hf)
		if px >= 0 && px < w && py >= 0 && py < h {
			idx := py*stride + px*4
			// Alpha blend grain onto canvas
			r0 := int(pix[idx])
			g0 := int(pix[idx+1])
			b0 := int(pix[idx+2])
			alpha := 0.65 + rng.Float64()*0.35
			pix[idx] = uint8(float64(r0)*(1-alpha) + float64(ir)*alpha)
			pix[idx+1] = uint8(float64(g0)*(1-alpha) + float64(ig)*alpha)
			pix[idx+2] = uint8(float64(b0)*(1-alpha) + float64(ib)*alpha)
		}
	}

	return img, nil
}
