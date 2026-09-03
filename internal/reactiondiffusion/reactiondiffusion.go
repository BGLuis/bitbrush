// Package reactiondiffusion simulates Gray-Scott reaction-diffusion partial
// differential equations to generate organic Turing patterns: leopard spots,
// brain coral, mitotic cells, and labyrinthine textures.
package reactiondiffusion

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
	Seed     int64   `json:"seed"`
	Preset   string  `json:"preset"`   // coral | mitosis | spots | labyrinth | rings | custom
	Feed     float64 `json:"feed"`     // feed rate F (0.01..0.09)
	Kill     float64 `json:"kill"`     // kill rate k (0.04..0.07)
	Steps    int     `json:"steps"`    // simulation steps (200..2500)
	GridSize int     `json:"gridSize"` // simulation grid size (80..240)
	Palette  string  `json:"palette"`  // bone | bioluminescent | oxblood | monochrome
	Invert   bool    `json:"invert"`   // invert foreground/background
}

func defaults() params {
	return params{
		Seed:     101,
		Preset:   "coral",
		Feed:     0.0545,
		Kill:     0.0620,
		Steps:    750,
		GridSize: 140,
		Palette:  "bioluminescent",
		Invert:   false,
	}
}

func presetValues(name string) (f, k float64) {
	switch name {
	case "mitosis":
		return 0.0367, 0.0649
	case "spots":
		return 0.0300, 0.0620
	case "labyrinth":
		return 0.0290, 0.0570
	case "rings":
		return 0.0180, 0.0510
	default: // "coral"
		return 0.0545, 0.0620
	}
}

type colorStop struct {
	pos float64
	col color.RGBA
}

func paletteStops(name string) []colorStop {
	switch name {
	case "bone":
		return []colorStop{
			{0.0, color.RGBA{14, 12, 10, 255}},
			{0.35, color.RGBA{75, 60, 45, 255}},
			{0.7, color.RGBA{185, 168, 140, 255}},
			{1.0, color.RGBA{245, 240, 230, 255}},
		}
	case "oxblood":
		return []colorStop{
			{0.0, color.RGBA{12, 5, 8, 255}},
			{0.35, color.RGBA{107, 15, 22, 255}},
			{0.7, color.RGBA{210, 60, 50, 255}},
			{1.0, color.RGBA{250, 225, 205, 255}},
		}
	case "monochrome":
		return []colorStop{
			{0.0, color.RGBA{10, 10, 12, 255}},
			{0.4, color.RGBA{45, 45, 52, 255}},
			{0.8, color.RGBA{210, 210, 220, 255}},
			{1.0, color.RGBA{255, 255, 255, 255}},
		}
	default: // "bioluminescent"
		return []colorStop{
			{0.0, color.RGBA{3, 10, 20, 255}},
			{0.35, color.RGBA{10, 85, 105, 255}},
			{0.7, color.RGBA{28, 195, 178, 255}},
			{1.0, color.RGBA{220, 252, 246, 255}},
		}
	}
}

func samplePalette(stops []colorStop, t float64) color.RGBA {
	if t <= stops[0].pos {
		return stops[0].col
	}
	last := stops[len(stops)-1]
	if t >= last.pos {
		return last.col
	}
	for i := 0; i < len(stops)-1; i++ {
		s0, s1 := stops[i], stops[i+1]
		if t >= s0.pos && t <= s1.pos {
			span := s1.pos - s0.pos
			u := (t - s0.pos) / span
			// Smoothstep interpolation
			u = u * u * (3.0 - 2.0*u)
			r := uint8(float64(s0.col.R) + float64(int(s1.col.R)-int(s0.col.R))*u)
			g := uint8(float64(s0.col.G) + float64(int(s1.col.G)-int(s0.col.G))*u)
			b := uint8(float64(s0.col.B) + float64(int(s1.col.B)-int(s0.col.B))*u)
			return color.RGBA{r, g, b, 255}
		}
	}
	return last.col
}

func init() {
	generators.Register("reactiondiffusion", render)
}

func render(raw json.RawMessage, w, h int) (*image.RGBA, error) {
	p := defaults()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("reactiondiffusion: bad params: %w", err)
		}
	}

	if p.Preset != "custom" {
		p.Feed, p.Kill = presetValues(p.Preset)
	} else {
		if p.Feed < 0.005 {
			p.Feed = 0.0545
		}
		if p.Kill < 0.01 {
			p.Kill = 0.0620
		}
	}

	N := p.GridSize
	if N < 60 {
		N = 60
	} else if N > 240 {
		N = 240
	}

	steps := p.Steps
	if steps < 100 {
		steps = 100
	} else if steps > 2500 {
		steps = 2500
	}

	size := N * N
	gridA := make([]float32, size)
	gridB := make([]float32, size)
	nextA := make([]float32, size)
	nextB := make([]float32, size)

	// Initialize A=1, B=0
	for i := 0; i < size; i++ {
		gridA[i] = 1.0
		gridB[i] = 0.0
	}

	// Seed perturbations
	rng := rand.New(rand.NewSource(p.Seed))
	numSpots := 7 + rng.Intn(8)
	for s := 0; s < numSpots; s++ {
		cx := 10 + rng.Intn(N-20)
		cy := 10 + rng.Intn(N-20)
		rad := 3 + rng.Intn(6)
		radSq := rad * rad

		x0 := max(0, cx-rad)
		x1 := min(N-1, cx+rad)
		y0 := max(0, cy-rad)
		y1 := min(N-1, cy+rad)

		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				dx := x - cx
				dy := y - cy
				if dx*dx+dy*dy <= radSq {
					idx := y*N + x
					gridA[idx] = 0.0
					gridB[idx] = 1.0
				}
			}
		}
	}

	// Centered seed
	mid := N / 2
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			if dx*dx+dy*dy <= 16 {
				idx := (mid+dy)*N + (mid + dx)
				gridA[idx] = 0.0
				gridB[idx] = 1.0
			}
		}
	}

	const dA = float32(1.0)
	const dB = float32(0.5)
	const dt = float32(1.0)
	F := float32(p.Feed)
	K := float32(p.Kill)

	// Simulation loop
	for step := 0; step < steps; step++ {
		for y := 0; y < N; y++ {
			yu := ((y - 1 + N) % N) * N
			yd := ((y + 1) % N) * N
			yc := y * N

			for x := 0; x < N; x++ {
				xl := (x - 1 + N) % N
				xr := (x + 1) % N
				idx := yc + x

				a := gridA[idx]
				b := gridB[idx]

				lapA := (gridA[yc+xl]+gridA[yc+xr]+gridA[yu+x]+gridA[yd+x])*0.2 +
					(gridA[yu+xl]+gridA[yu+xr]+gridA[yd+xl]+gridA[yd+xr])*0.05 - a
				lapB := (gridB[yc+xl]+gridB[yc+xr]+gridB[yu+x]+gridB[yd+x])*0.2 +
					(gridB[yu+xl]+gridB[yu+xr]+gridB[yd+xl]+gridB[yd+xr])*0.05 - b

				abb := a * b * b
				na := a + (dA*lapA-abb+F*(1.0-a))*dt
				nb := b + (dB*lapB+abb-(K+F)*b)*dt

				if na < 0 {
					na = 0
				} else if na > 1 {
					na = 1
				}
				if nb < 0 {
					nb = 0
				} else if nb > 1 {
					nb = 1
				}

				nextA[idx] = na
				nextB[idx] = nb
			}
		}

		// Swap buffers
		gridA, nextA = nextA, gridA
		gridB, nextB = nextB, gridB
	}

	// Render to output canvas with bilinear interpolation
	stops := paletteStops(p.Palette)
	img := generators.NewCanvas(w, h, color.RGBA{0, 0, 0, 255})
	pix := img.Pix
	stride := img.Stride

	scaleX := float64(N-1) / float64(w)
	scaleY := float64(N-1) / float64(h)

	for py := 0; py < h; py++ {
		gy := float64(py) * scaleY
		iy := int(gy)
		fy := gy - float64(iy)
		iy1 := min(N-1, iy+1)

		rowOff := py * stride

		for px := 0; px < w; px++ {
			gx := float64(px) * scaleX
			ix := int(gx)
			fx := gx - float64(ix)
			ix1 := min(N-1, ix+1)

			b00 := float64(gridB[iy*N+ix])
			b10 := float64(gridB[iy*N+ix1])
			b01 := float64(gridB[iy1*N+ix])
			b11 := float64(gridB[iy1*N+ix1])

			bVal := (b00*(1-fx)+b10*fx)*(1-fy) + (b01*(1-fx)+b11*fx)*fy
			// Normalize contrast
			bVal = math.Min(1.0, math.Max(0.0, (bVal-0.15)/0.45))
			if p.Invert {
				bVal = 1.0 - bVal
			}

			col := samplePalette(stops, bVal)
			off := rowOff + px*4
			pix[off] = col.R
			pix[off+1] = col.G
			pix[off+2] = col.B
			pix[off+3] = 255
		}
	}

	return img, nil
}
