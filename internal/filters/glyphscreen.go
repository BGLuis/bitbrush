package filters

import (
	"image"
	"image/color"
	"math"
)

// GlyphScreen replaces each cell of src with a single procedural mark whose
// size or ink density tracks the cell's darkness. It covers the "cross",
// "diagonal", "diamond" and "lines" art styles in one filter, drawn without
// a font (the glyphs — ◆ × ╱ — are outside basicfont's ASCII range).
//
// Params:
//
//	cell        int,    px cell edge, default 10 (clamped >= 1)
//	shape       string, "cross" (default), "diagonal", "backslash",
//	            "vertical", "horizontal", "plus", "diamond" or "dot"
//	colored     bool,   tint the mark by the cell's mean colour (default true)
//	background  string, hex fill behind the mark (default "#000000")
//	weight      float,  stroke width in px for the line shapes, default 2
//	invert      bool,   drive the mark from brightness instead of darkness (default false)
func GlyphScreen(src *image.RGBA, p Params) (*image.RGBA, error) {
	cell := clampInt(p.Int("cell", 10), 1, 1<<20)
	shape := p.String("shape", "cross")
	colored := p.Bool("colored", true)
	invert := p.Bool("invert", false)
	weight := clampF(p.Float("weight", 2), 0.5, 64)
	bg := hexOr(p.String("background", "#000000"), color.RGBA{A: 255})
	flatFG := color.RGBA{R: 224, G: 224, B: 224, A: 255}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}
	half := weight/2 + 0.5

	for cy := 0; cy < h; cy += cell {
		y1 := min(cy+cell, h)
		for cx := 0; cx < w; cx += cell {
			x1 := min(cx+cell, w)
			cw, chh := x1-cx, y1-cy

			avg := blockAverage(src, b.Min.X+cx, b.Min.Y+cy, cw, chh)
			t := 1 - luma(avg.R, avg.G, avg.B)/255 // ink coverage: darker -> more
			if invert {
				t = 1 - t
			}
			fg := flatFG
			if colored {
				fg = color.RGBA{R: avg.R, G: avg.G, B: avg.B, A: 255}
			}

			fcx := float64(cx) + float64(cw)/2
			fcy := float64(cy) + float64(chh)/2
			rad := t * math.Min(float64(cw), float64(chh)) / 2

			for y := cy; y < y1; y++ {
				di := dst.PixOffset(cx, y)
				si := src.PixOffset(b.Min.X+cx, b.Min.Y+y)
				for x := cx; x < x1; x++ {
					px, py := float64(x)+0.5, float64(y)+0.5
					cov := 0.0
					switch shape {
					case "dot":
						if math.Hypot(px-fcx, py-fcy) <= rad {
							cov = 1
						}
					case "diamond":
						if math.Abs(px-fcx)+math.Abs(py-fcy) <= rad {
							cov = 1
						}
					case "vertical":
						if math.Abs(px-fcx) <= half {
							cov = t
						}
					case "horizontal":
						if math.Abs(py-fcy) <= half {
							cov = t
						}
					case "plus":
						if math.Abs(px-fcx) <= half || math.Abs(py-fcy) <= half {
							cov = t
						}
					case "diagonal": // '/'
						if distToSeg(px, py, float64(cx), float64(y1), float64(x1), float64(cy)) <= half {
							cov = t
						}
					case "backslash": // '\'
						if distToSeg(px, py, float64(cx), float64(cy), float64(x1), float64(y1)) <= half {
							cov = t
						}
					default: // "cross" — an X
						d := math.Min(
							distToSeg(px, py, float64(cx), float64(cy), float64(x1), float64(y1)),
							distToSeg(px, py, float64(cx), float64(y1), float64(x1), float64(cy)),
						)
						if d <= half {
							cov = t
						}
					}
					dst.Pix[di] = lerpByte(bg.R, fg.R, cov)
					dst.Pix[di+1] = lerpByte(bg.G, fg.G, cov)
					dst.Pix[di+2] = lerpByte(bg.B, fg.B, cov)
					dst.Pix[di+3] = src.Pix[si+3]
					di += 4
					si += 4
				}
			}
		}
	}
	return dst, nil
}

func init() { Register("glyphscreen", GlyphScreen) }

// distToSeg is the Euclidean distance from (px,py) to the segment
// (ax,ay)-(bx,by).
func distToSeg(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	u := ((px-ax)*dx + (py-ay)*dy) / l2
	u = clampF(u, 0, 1)
	return math.Hypot(px-(ax+u*dx), py-(ay+u*dy))
}
