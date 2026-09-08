package filters

import (
	"image"
	"image/color"
	"math"
)

// Vignette darkens (or tints) src toward the frame edges by a smooth radial
// falloff.
//
// Params:
//
//	strength   float,  0..1 amount of darkening at the outer edge (default 0.6)
//	inner      float,  0..1 normalised radius where the falloff starts (default 0.4)
//	outer      float,  0..1 normalised radius where it reaches full strength (default 1.0)
//	roundness  float,  >1 pinches the vignette vertically, <1 horizontally (default 1.0)
//	color      string, hex tint the edges fade toward (default "#000000")
func Vignette(src *image.RGBA, p Params) (*image.RGBA, error) {
	strength := clampF(p.Float("strength", 0.6), 0, 1)
	inner := clampF(p.Float("inner", 0.4), 0, 2)
	outer := clampF(p.Float("outer", 1.0), 0, 2)
	roundness := clampF(p.Float("roundness", 1.0), 0.1, 4)
	tint := hexOr(p.String("color", "#000000"), color.RGBA{A: 255})

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	for y := 0; y < h; y++ {
		ny := ((float64(y)+0.5)/float64(h)*2 - 1) * roundness
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			nx := (float64(x)+0.5)/float64(w)*2 - 1
			d := math.Hypot(nx, ny) / math.Sqrt2
			f := 1 - strength*smoothstep(inner, outer, d)
			dst.Pix[di] = lerpByte(tint.R, src.Pix[si], f)
			dst.Pix[di+1] = lerpByte(tint.G, src.Pix[si+1], f)
			dst.Pix[di+2] = lerpByte(tint.B, src.Pix[si+2], f)
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			di += 4
		}
	}
	return dst, nil
}

func init() { Register("vignette", Vignette) }
