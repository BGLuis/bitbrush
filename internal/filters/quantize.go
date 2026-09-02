package filters

import (
	"image"

	"bitbrush/internal/palette"
)

// Quantize builds an at-most-`colors` palette from src with median-cut and
// remaps every pixel to its nearest palette entry, giving the flat banded
// "poster" look. Alpha is carried through from src.
//
// Params:
//
//	colors  int,    palette size, clamped to 2..256 (default 8)
//	space   string, "oklab" (perceptual, default) or "srgb" match metric
func Quantize(src *image.RGBA, p Params) (*image.RGBA, error) {
	n := p.Int("colors", 8)
	if n < 2 {
		n = 2
	}
	if n > 256 {
		n = 256
	}

	space := palette.SpaceOKLab
	if p.String("space", "oklab") == "srgb" {
		space = palette.SpaceSRGB
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	pal := palette.MedianCut(src, n)
	if len(pal) == 0 {
		return cloneRGBA(src), nil // fully transparent source, nothing to quantise
	}
	m := palette.NewMapper(pal, space)

	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			out := m.At(palette.RGB{R: src.Pix[si], G: src.Pix[si+1], B: src.Pix[si+2]})
			dst.Pix[di] = out.R
			dst.Pix[di+1] = out.G
			dst.Pix[di+2] = out.B
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			di += 4
		}
	}
	return dst, nil
}

func init() { Register("quantize", Quantize) }
