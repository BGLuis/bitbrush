// Package contours renders an animated-topographic-map still: a domain-
// warped fBm heightfield, shaded relief, and iso-lines traced by marching
// squares with every Nth line drawn as a heavier index contour. A port of
// the "Living Landscape" reference sketch, deterministic on seed + time.
package contours

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"

	"bitbrush/internal/generators"
)

type params struct {
	Seed       int64   `json:"seed"`
	Levels     int     `json:"levels"`
	IndexEvery int     `json:"indexEvery"`
	Scale      float64 `json:"scale"`
	Warp       float64 `json:"warp"`
	Octaves    int     `json:"octaves"`
	Time       float64 `json:"time"`
	Hillshade  bool    `json:"hillshade"`
	Palette    string  `json:"palette"` // paper | blueprint | terrain
	Background string  `json:"background"`
}

func defaults() params {
	return params{
		Seed: 1337, Levels: 18, IndexEvery: 5, Scale: 1, Warp: 0.6,
		Octaves: 5, Hillshade: true, Palette: "paper",
	}
}

func render(raw json.RawMessage, w, h int) (*image.RGBA, error) {
	p := defaults()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("contours: bad params: %w", err)
		}
	}
	p.Levels = clampi(p.Levels, 4, 48)
	p.IndexEvery = clampi(p.IndexEvery, 0, 12)
	p.Scale = clampf(p.Scale, 0.2, 4)
	p.Warp = clampf(p.Warp, 0, 2)
	p.Octaves = clampi(p.Octaves, 1, 8)

	pal := paletteFor(p.Palette)
	if bg := generators.ParseHex(p.Background, color.RGBA{}); p.Background != "" {
		pal.bg = bg
	}

	cell := math.Max(3, math.Round(float64(max(w, h))/220))
	cols := int(math.Ceil(float64(w)/cell)) + 2
	rows := int(math.Ceil(float64(h)/cell)) + 2

	pl := newPerlin(p.Seed)
	sc := 0.0021 * (1400.0 / math.Max(float64(w), 900)) * p.Scale
	heights := make([]float64, cols*rows)
	for j := 0; j < rows; j++ {
		for i := 0; i < cols; i++ {
			x := float64(i) * cell * sc
			y := float64(j) * cell * sc
			wx := pl.fbm(x*0.6+31.4, y*0.6, p.Time*0.5, p.Octaves) * 0.6 * p.Warp
			wy := pl.fbm(x*0.6, y*0.6+17.2, p.Time*0.5, p.Octaves) * 0.6 * p.Warp
			heights[j*cols+i] = pl.fbm(x+wx, y+wy, p.Time, p.Octaves)
		}
	}

	img := generators.NewCanvas(w, h, pal.bg)
	if p.Hillshade {
		paintRelief(img, heights, cols, rows, cell, pal)
	}

	half := cell * 0.5
	for l := 1; l < p.Levels; l++ {
		level := float64(l) / float64(p.Levels)
		isIndex := p.IndexEvery > 0 && l%p.IndexEvery == 0
		lineCol := pal.minor
		alpha := pal.minorA
		passes := 1
		if isIndex {
			lineCol = pal.index
			alpha = pal.indexA
			passes = 2
		}
		marching(heights, cols, rows, cell, half, level, func(x0, y0, x1, y1 float64) {
			for k := 0; k < passes; k++ {
				off := float64(k) * 0.6
				generators.Line(img, x0+off, y0, x1+off, y1, alpha, lineCol)
			}
		})
	}
	return img, nil
}

func paintRelief(img *image.RGBA, heights []float64, cols, rows int, cell float64, pal palette) {
	shade := make([][3]float64, cols*rows)
	lx, ly, lz := -0.6, -0.7, 0.55
	b2i := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	for j := 0; j < rows; j++ {
		for i := 0; i < cols; i++ {
			k := j*cols + i
			hl := heights[k-b2i(i > 0)]
			hr := heights[k+b2i(i < cols-1)]
			hu := heights[k-cols*b2i(j > 0)]
			hd := heights[k+cols*b2i(j < rows-1)]
			gx := (hr - hl) * 14
			gy := (hd - hu) * 14
			ln := math.Sqrt(gx*gx + gy*gy + 1)
			s := clampf(((-gx/ln)*lx+(-gy/ln)*ly+(1/ln)*lz)*1.1, 0, 1)
			shade[k] = pal.shade(heights[k], s)
		}
	}
	w, h := img.Rect.Dx(), img.Rect.Dy()
	half := cell * 0.5
	for py := 0; py < h; py++ {
		gy := (float64(py) + half) / cell
		j0 := int(gy)
		fy := gy - float64(j0)
		j0 = clampi(j0, 0, rows-2)
		for px := 0; px < w; px++ {
			gx := (float64(px) + half) / cell
			i0 := int(gx)
			fx := gx - float64(i0)
			i0 = clampi(i0, 0, cols-2)
			c00 := shade[j0*cols+i0]
			c10 := shade[j0*cols+i0+1]
			c01 := shade[(j0+1)*cols+i0]
			c11 := shade[(j0+1)*cols+i0+1]
			o := img.PixOffset(px, py)
			for ch := 0; ch < 3; ch++ {
				v := (c00[ch]*(1-fx)+c10[ch]*fx)*(1-fy) + (c01[ch]*(1-fx)+c11[ch]*fx)*fy
				img.Pix[o+ch] = u8b(v)
			}
		}
	}
}

func marching(heights []float64, cols, rows int, cell, half, level float64, add func(x0, y0, x1, y1 float64)) {
	for j := 0; j < rows-1; j++ {
		for i := 0; i < cols-1; i++ {
			k := j*cols + i
			a, b, c, d := heights[k], heights[k+1], heights[k+cols+1], heights[k+cols]
			idx := 0
			if a > level {
				idx |= 8
			}
			if b > level {
				idx |= 4
			}
			if c > level {
				idx |= 2
			}
			if d > level {
				idx |= 1
			}
			if idx == 0 || idx == 15 {
				continue
			}
			x := float64(i)*cell - half
			y := float64(j)*cell - half
			tx, ty := x+cell*safeDiv(level-a, b-a), y
			rx, ry := x+cell, y+cell*safeDiv(level-b, c-b)
			bx, by := x+cell*safeDiv(level-d, c-d), y+cell
			lxp, ly := x, y+cell*safeDiv(level-a, d-a)
			mid := (a + b + c + d) / 4
			switch idx {
			case 1, 14:
				add(lxp, ly, bx, by)
			case 2, 13:
				add(bx, by, rx, ry)
			case 3, 12:
				add(lxp, ly, rx, ry)
			case 4, 11:
				add(tx, ty, rx, ry)
			case 5:
				if mid > level {
					add(lxp, ly, tx, ty)
					add(bx, by, rx, ry)
				} else {
					add(lxp, ly, bx, by)
					add(tx, ty, rx, ry)
				}
			case 6, 9:
				add(tx, ty, bx, by)
			case 7, 8:
				add(lxp, ly, tx, ty)
			case 10:
				if mid > level {
					add(tx, ty, rx, ry)
					add(lxp, ly, bx, by)
				} else {
					add(tx, ty, lxp, ly)
					add(rx, ry, bx, by)
				}
			}
		}
	}
}

func safeDiv(n, d float64) float64 {
	if math.Abs(d) < 1e-12 {
		return 0.5
	}
	return n / d
}

type palette struct {
	bg             color.RGBA
	minor, index   color.RGBA
	minorA, indexA float64
	shade          func(hgt, s float64) [3]float64
}

func paletteFor(name string) palette {
	switch name {
	case "blueprint":
		return palette{
			bg:    color.RGBA{10, 26, 51, 255},
			minor: color.RGBA{127, 209, 255, 255}, minorA: 0.42,
			index: color.RGBA{217, 236, 255, 255}, indexA: 0.95,
			shade: func(hgt, s float64) [3]float64 {
				v := 0.55 + s*0.9 + hgt*0.5
				return [3]float64{10 + 22*v, 26 + 34*v, 51 + 50*v}
			},
		}
	case "terrain":
		stops := [][4]float64{
			{0, 62, 110, 74}, {0.25, 122, 150, 82}, {0.45, 200, 178, 122},
			{0.65, 150, 112, 78}, {0.82, 120, 104, 92}, {1, 242, 239, 230},
		}
		return palette{
			bg:    color.RGBA{233, 228, 210, 255},
			minor: color.RGBA{58, 46, 34, 255}, minorA: 0.4,
			index: color.RGBA{58, 46, 34, 255}, indexA: 0.9,
			shade: func(hgt, s float64) [3]float64 {
				i := 0
				for i < len(stops)-2 && hgt > stops[i+1][0] {
					i++
				}
				a, bb := stops[i], stops[i+1]
				t := clampf((hgt-a[0])/(bb[0]-a[0]), 0, 1)
				l := 0.72 + s*0.56
				return [3]float64{
					(a[1] + (bb[1]-a[1])*t) * l,
					(a[2] + (bb[2]-a[2])*t) * l,
					(a[3] + (bb[3]-a[3])*t) * l,
				}
			},
		}
	default: // paper
		return palette{
			bg:    color.RGBA{243, 238, 227, 255},
			minor: color.RGBA{43, 42, 40, 255}, minorA: 0.42,
			index: color.RGBA{43, 42, 40, 255}, indexA: 0.92,
			shade: func(hgt, s float64) [3]float64 {
				v := 232 + (s-0.5)*34 - hgt*14
				return [3]float64{v + 8, v + 2, v - 12}
			},
		}
	}
}

func u8b(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 255:
		return 255
	}
	return uint8(v + 0.5)
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

func init() { generators.Register("contours", render) }
