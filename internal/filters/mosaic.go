package filters

import (
	"image"
	"image/color"

	"bitbrush/internal/palette"
)

// Mosaic rebuilds src as a grid of flat tiles, each filled with the mean
// colour beneath it, with optional grout between tiles, an optional retro
// palette snap and an optional Bayer texture.
//
// Params:
//
//	cellWidth     int,    tile width px, default 16 (clamped >= 1)
//	cellHeight    int,    tile height px, default 16 (clamped >= 1)
//	gap           int,    grout thickness px between tiles, default 0
//	gapColor      string, hex grout colour, default "#101010"
//	palette       string, retro palette name or "none" (default)
//	bayerOverlay  bool,   lay a 4x4 Bayer texture over each tile (default false)
func Mosaic(src *image.RGBA, p Params) (*image.RGBA, error) {
	cw := clampInt(p.Int("cellWidth", 16), 1, 1<<20)
	ch := clampInt(p.Int("cellHeight", 16), 1, 1<<20)
	gap := clampInt(p.Int("gap", 0), 0, 1<<20)
	grout := hexOr(p.String("gapColor", "#101010"), color.RGBA{R: 16, G: 16, B: 16, A: 255})
	bayerOverlay := p.Bool("bayerOverlay", false)

	var mapper *palette.Mapper
	if pal := namedPalette(p.String("palette", "none")); pal != nil {
		mapper = palette.NewMapper(pal, palette.SpaceOKLab)
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}
	bayer := bayerMatrix(4)
	const bayerAmp = 40.0

	for cy := 0; cy < h; cy += ch {
		y1 := min(cy+ch, h)
		for cx := 0; cx < w; cx += cw {
			x1 := min(cx+cw, w)

			avg := blockAverage(src, b.Min.X+cx, b.Min.Y+cy, x1-cx, y1-cy)
			tile := color.RGBA{R: avg.R, G: avg.G, B: avg.B, A: 255}
			if mapper != nil {
				m := mapper.At(palette.RGB{R: tile.R, G: tile.G, B: tile.B})
				tile = color.RGBA{R: m.R, G: m.G, B: m.B, A: 255}
			}

			gl := gap / 2
			gr := gap - gl
			inX0, inY0 := cx+gl, cy+gl
			inX1, inY1 := x1-gr, y1-gr

			for y := cy; y < y1; y++ {
				di := dst.PixOffset(cx, y)
				si := src.PixOffset(b.Min.X+cx, b.Min.Y+y)
				for x := cx; x < x1; x++ {
					col := tile
					if gap > 0 && (x < inX0 || x >= inX1 || y < inY0 || y >= inY1) {
						col = grout
					} else if bayerOverlay {
						d := (bayer[(y&3)*4+(x&3)] - 0.5) * bayerAmp
						col = color.RGBA{
							R: clampU8(float64(tile.R) + d),
							G: clampU8(float64(tile.G) + d),
							B: clampU8(float64(tile.B) + d),
							A: 255,
						}
					}
					dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2] = col.R, col.G, col.B
					dst.Pix[di+3] = src.Pix[si+3]
					di += 4
					si += 4
				}
			}
		}
	}
	return dst, nil
}

func init() { Register("mosaic", Mosaic) }
