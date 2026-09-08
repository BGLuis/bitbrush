package filters

import (
	"image"
	"image/color"
	"math"

	"bitbrush/internal/palette"
)

// Lego rebuilds src as a grid of studded plastic bricks: each cell is
// posterised to a shared palette, then a raised circular stud is shaded
// with a top-left highlight and a bottom-right shadow.
//
// Params:
//
//	cell          int,   brick size px, default 16 (clamped >= 2)
//	colors        int,   posterise palette size, 2..256 (default 16)
//	studContrast  float, 0..1 strength of the stud's 3-D shading (default 0.35)
//	outline       bool,  1px darker seam around each brick (default true)
func Lego(src *image.RGBA, p Params) (*image.RGBA, error) {
	cell := clampInt(p.Int("cell", 16), 2, 1<<20)
	colors := clampInt(p.Int("colors", 16), 2, 256)
	studContrast := clampF(p.Float("studContrast", 0.35), 0, 1)
	outline := p.Bool("outline", true)

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	pal := palette.MedianCut(src, colors)
	var mapper *palette.Mapper
	if len(pal) > 0 {
		mapper = palette.NewMapper(pal, palette.SpaceOKLab)
	}

	studR := 0.31 * float64(cell)

	for cy := 0; cy < h; cy += cell {
		y1 := min(cy+cell, h)
		for cx := 0; cx < w; cx += cell {
			x1 := min(cx+cell, w)
			avg := blockAverage(src, b.Min.X+cx, b.Min.Y+cy, x1-cx, y1-cy)
			tile := color.RGBA{R: avg.R, G: avg.G, B: avg.B, A: 255}
			if mapper != nil {
				m := mapper.At(palette.RGB{R: tile.R, G: tile.G, B: tile.B})
				tile = color.RGBA{R: m.R, G: m.G, B: m.B, A: 255}
			}
			fcx := float64(cx+x1) / 2
			fcy := float64(cy+y1) / 2

			for y := cy; y < y1; y++ {
				di := dst.PixOffset(cx, y)
				si := src.PixOffset(b.Min.X+cx, b.Min.Y+y)
				for x := cx; x < x1; x++ {
					shade := 1.0
					if outline && (x == cx || y == cy || x == x1-1 || y == y1-1) {
						shade = 0.62
					} else if studR > 0 {
						nx := (float64(x) + 0.5 - fcx) / studR
						ny := (float64(y) + 0.5 - fcy) / studR
						if d2 := nx*nx + ny*ny; d2 <= 1 {
							// Spherical bump: top-left lit, bottom-right shadowed.
							shade = 1 + studContrast*(-(nx+ny)/math.Sqrt2)*math.Sqrt(1-d2)
						}
					}
					dst.Pix[di] = clampU8(float64(tile.R) * shade)
					dst.Pix[di+1] = clampU8(float64(tile.G) * shade)
					dst.Pix[di+2] = clampU8(float64(tile.B) * shade)
					dst.Pix[di+3] = src.Pix[si+3]
					di += 4
					si += 4
				}
			}
		}
	}
	return dst, nil
}

func init() { Register("lego", Lego) }
