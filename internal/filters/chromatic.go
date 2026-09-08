package filters

import (
	"image"
	"math"
)

// Chromatic simulates lens chromatic aberration: the red and blue channels
// are sampled with a small opposing offset, growing toward the frame edge
// in the radial mode.
//
// Params:
//
//	strength  float, px offset between the R and B samples (default 3)
//	radial    bool,  offset along the centre->pixel direction, scaled by
//	          distance, vs a flat horizontal offset (default true)
func Chromatic(src *image.RGBA, p Params) (*image.RGBA, error) {
	strength := clampF(p.Float("strength", 3), 0, 256)
	radial := p.Bool("radial", true)

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}
	cx, cy := float64(w-1)/2, float64(h-1)/2
	maxD := math.Hypot(cx, cy)
	if maxD == 0 {
		maxD = 1
	}

	for y := 0; y < h; y++ {
		di := dst.PixOffset(0, y)
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		for x := 0; x < w; x++ {
			fx, fy := float64(x), float64(y)
			var ox, oy float64
			if radial {
				dx, dy := fx-cx, fy-cy
				d := math.Hypot(dx, dy)
				if d > 1e-6 {
					scale := strength * (d / maxD)
					ox, oy = dx/d*scale, dy/d*scale
				}
			} else {
				ox = strength
			}
			r := bilinearSample(src, fx+ox, fy+oy)
			bl := bilinearSample(src, fx-ox, fy-oy)
			dst.Pix[di] = r.R
			dst.Pix[di+1] = src.Pix[si+1]
			dst.Pix[di+2] = bl.B
			dst.Pix[di+3] = src.Pix[si+3]
			di += 4
			si += 4
		}
	}
	return dst, nil
}

func init() { Register("chromatic", Chromatic) }
