package filters

import (
	"image"
	"image/color"
	"math"

	"bitbrush/internal/palette"
)

// PixelArt is Pixelate with a colour reduction: after averaging each cell it
// snaps the tile either to a named retro palette or to a per-channel
// bit-depth, and can draw a 1px dark seam between differently-coloured
// tiles.
//
// Params:
//
//	cell     int,    cell edge px, default 8 (clamped >= 1)
//	palette  string, retro palette name or "none" (default)
//	bits     int,    per-channel bit depth when palette is "none", 1..8 (default 8)
//	outline  bool,   dark seam between differing tiles (default false)
func PixelArt(src *image.RGBA, p Params) (*image.RGBA, error) {
	cell := clampInt(p.Int("cell", 8), 1, 1<<20)
	bits := clampInt(p.Int("bits", 8), 1, 8)
	outline := p.Bool("outline", false)

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

	cols := (w + cell - 1) / cell
	rows := (h + cell - 1) / cell
	tiles := make([]color.RGBA, cols*rows)
	levels := 1 << bits
	steps := float64(levels - 1)
	snapChan := func(v uint8) uint8 {
		if bits >= 8 {
			return v
		}
		return clampU8(math.Round(float64(v)/255*steps) / steps * 255)
	}

	for ry := 0; ry < rows; ry++ {
		cy := ry * cell
		y1 := min(cy+cell, h)
		for rx := 0; rx < cols; rx++ {
			cx := rx * cell
			x1 := min(cx+cell, w)

			avg := blockAverage(src, b.Min.X+cx, b.Min.Y+cy, x1-cx, y1-cy)
			t := color.RGBA{R: avg.R, G: avg.G, B: avg.B, A: 255}
			if mapper != nil {
				m := mapper.At(palette.RGB{R: t.R, G: t.G, B: t.B})
				t = color.RGBA{R: m.R, G: m.G, B: m.B, A: 255}
			} else {
				t = color.RGBA{R: snapChan(t.R), G: snapChan(t.G), B: snapChan(t.B), A: 255}
			}
			tiles[ry*cols+rx] = t

			for y := cy; y < y1; y++ {
				di := dst.PixOffset(cx, y)
				si := src.PixOffset(b.Min.X+cx, b.Min.Y+y)
				for x := cx; x < x1; x++ {
					dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2] = t.R, t.G, t.B
					dst.Pix[di+3] = src.Pix[si+3]
					di += 4
					si += 4
				}
			}
		}
	}

	if outline {
		for ry := 0; ry < rows; ry++ {
			cy := ry * cell
			y1 := min(cy+cell, h)
			for rx := 0; rx < cols; rx++ {
				cx := rx * cell
				x1 := min(cx+cell, w)
				me := tiles[ry*cols+rx]
				seamLeft := rx > 0 && tiles[ry*cols+rx-1] != me
				seamTop := ry > 0 && tiles[(ry-1)*cols+rx] != me
				if seamLeft {
					for y := cy; y < y1; y++ {
						darkenPx(dst, cx, y)
					}
				}
				if seamTop {
					for x := cx; x < x1; x++ {
						darkenPx(dst, x, cy)
					}
				}
			}
		}
	}
	return dst, nil
}

func init() { Register("pixelart", PixelArt) }

// darkenPx multiplies the RGB of dst at (x,y) by 0.45, leaving alpha alone.
func darkenPx(dst *image.RGBA, x, y int) {
	o := dst.PixOffset(x, y)
	dst.Pix[o] = uint8(float64(dst.Pix[o]) * 0.45)
	dst.Pix[o+1] = uint8(float64(dst.Pix[o+1]) * 0.45)
	dst.Pix[o+2] = uint8(float64(dst.Pix[o+2]) * 0.45)
}
