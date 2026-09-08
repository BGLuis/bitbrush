package filters

import (
	"image"
	"image/color"
	"math"
)

// Blocks renders src in the "block characters" style: each cell becomes a
// Unicode-block-like glyph drawn procedurally as filled rectangles (no font
// — basicfont has no block glyphs), a foreground colour taken from the
// cell's ink and a flat background colour.
//
// Params:
//
//	cellWidth   int,    px, default 6 (clamped >= 1)
//	cellHeight  int,    px, default 6 (clamped >= 1)
//	mode        string, "quadrant" (default), "halves" or "shade"
//	colored     bool,   tint the glyph by the cell's mean colour vs flat grey (default true)
//	background  string, hex fill behind the glyph (default "#000000")
//	invert      bool,   swap which luma end is "ink" (default false)
func Blocks(src *image.RGBA, p Params) (*image.RGBA, error) {
	cw := clampInt(p.Int("cellWidth", 6), 1, 1<<20)
	ch := clampInt(p.Int("cellHeight", 6), 1, 1<<20)
	mode := p.String("mode", "quadrant")
	colored := p.Bool("colored", true)
	invert := p.Bool("invert", false)
	bg := hexOr(p.String("background", "#000000"), color.RGBA{A: 255})
	flatFG := color.RGBA{R: 224, G: 224, B: 224, A: 255}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}
	bayer := bayerMatrix(4)

	// paintRect fills dst[x0:x1, y0:y1] (cell-local, already clipped) with
	// col's RGB, carrying src's alpha through pixel for pixel.
	paintRect := func(x0, y0, x1, y1 int, col color.RGBA) {
		for y := y0; y < y1; y++ {
			di := dst.PixOffset(x0, y)
			si := src.PixOffset(b.Min.X+x0, b.Min.Y+y)
			for x := x0; x < x1; x++ {
				dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2] = col.R, col.G, col.B
				dst.Pix[di+3] = src.Pix[si+3]
				di += 4
				si += 4
			}
		}
	}

	for cy := 0; cy < h; cy += ch {
		y1 := min(cy+ch, h)
		for cx := 0; cx < w; cx += cw {
			x1 := min(cx+cw, w)
			cellW, cellH := x1-cx, y1-cy

			cellAvg := blockAverage(src, b.Min.X+cx, b.Min.Y+cy, cellW, cellH)
			cellLuma := luma(cellAvg.R, cellAvg.G, cellAvg.B)
			fg := flatFG
			if colored {
				fg = color.RGBA{R: cellAvg.R, G: cellAvg.G, B: cellAvg.B, A: 255}
			}

			// "ink" is the brighter end by default (matches the ascii ramp
			// mapping white -> dense glyph); invert flips it.
			on := func(l float64) bool {
				if invert {
					return l < cellLuma
				}
				return l >= cellLuma
			}

			switch mode {
			case "shade":
				t := cellLuma / 255
				if invert {
					t = 1 - t
				}
				for y := cy; y < y1; y++ {
					di := dst.PixOffset(cx, y)
					si := src.PixOffset(b.Min.X+cx, b.Min.Y+y)
					for x := cx; x < x1; x++ {
						col := bg
						if t > bayer[(y&3)*4+(x&3)] {
							col = fg
						}
						dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2] = col.R, col.G, col.B
						dst.Pix[di+3] = src.Pix[si+3]
						di += 4
						si += 4
					}
				}

			case "halves":
				mx, my := cx+cellW/2, cy+cellH/2
				lL := luma3(blockAverage(src, b.Min.X+cx, b.Min.Y+cy, cellW/2, cellH))
				lR := luma3(blockAverage(src, b.Min.X+mx, b.Min.Y+cy, cellW-cellW/2, cellH))
				lT := luma3(blockAverage(src, b.Min.X+cx, b.Min.Y+cy, cellW, cellH/2))
				lB := luma3(blockAverage(src, b.Min.X+cx, b.Min.Y+my, cellW, cellH-cellH/2))
				paintRect(cx, cy, x1, y1, bg)
				if math.Abs(lL-lR) >= math.Abs(lT-lB) {
					if on(lL) {
						paintRect(cx, cy, mx, y1, fg)
					}
					if on(lR) {
						paintRect(mx, cy, x1, y1, fg)
					}
				} else {
					if on(lT) {
						paintRect(cx, cy, x1, my, fg)
					}
					if on(lB) {
						paintRect(cx, my, x1, y1, fg)
					}
				}

			default: // "quadrant"
				mx, my := cx+cellW/2, cy+cellH/2
				paintRect(cx, cy, x1, y1, bg)
				quads := [4][4]int{
					{cx, cy, mx, my},
					{mx, cy, x1, my},
					{cx, my, mx, y1},
					{mx, my, x1, y1},
				}
				for _, q := range quads {
					if q[2] <= q[0] || q[3] <= q[1] {
						continue
					}
					ql := luma3(blockAverage(src, b.Min.X+q[0], b.Min.Y+q[1], q[2]-q[0], q[3]-q[1]))
					if on(ql) {
						paintRect(q[0], q[1], q[2], q[3], fg)
					}
				}
			}
		}
	}
	return dst, nil
}

func init() { Register("blocks", Blocks) }

func luma3(c color.RGBA) float64 { return luma(c.R, c.G, c.B) }
