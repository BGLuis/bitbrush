package filters

import (
	"image"
	"math"
)

// CRT wraps src onto a curved cathode-ray tube: barrel distortion, an RGB
// phosphor mask, scanlines and an edge vignette. Pixels that curve off the
// tube are black.
//
// Params:
//
//	curvature  float, 0..0.5 barrel amount (default 0.15)
//	zoom       float, 0.5..1.5 scale before warp, <1 shows the bezel (default 1.0)
//	scanline   float, 0..1 scanline darkening (default 0.4)
//	mask       float, 0..1 RGB phosphor-stripe strength (default 0.2)
//	vignette   float, 0..1 corner darkening (default 0.3)
func CRT(src *image.RGBA, p Params) (*image.RGBA, error) {
	curvature := clampF(p.Float("curvature", 0.15), 0, 0.5)
	zoom := clampF(p.Float("zoom", 1.0), 0.5, 1.5)
	scan := clampF(p.Float("scanline", 0.4), 0, 1)
	mask := clampF(p.Float("mask", 0.2), 0, 1)
	vig := clampF(p.Float("vignette", 0.3), 0, 1)

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	for y := 0; y < h; y++ {
		di := dst.PixOffset(0, y)
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		// Normalised [-1,1] coords.
		v := (float64(y)+0.5)/float64(h)*2 - 1
		for x := 0; x < w; x++ {
			u := (float64(x)+0.5)/float64(w)*2 - 1

			// Barrel: push coords outward by r^2.
			uu := u / zoom
			vv := v / zoom
			r2 := uu*uu + vv*vv
			wu := uu * (1 + curvature*r2)
			wv := vv * (1 + curvature*r2)

			dst.Pix[di+3] = src.Pix[si+3]
			if wu < -1 || wu > 1 || wv < -1 || wv > 1 {
				dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2] = 0, 0, 0
				di += 4
				si += 4
				continue
			}

			fx := (wu + 1) / 2 * float64(w-1)
			fy := (wv + 1) / 2 * float64(h-1)
			c := bilinearSample(src, fx, fy)
			cr, cg, cb := float64(c.R), float64(c.G), float64(c.B)

			// Scanlines from the sampled row.
			if scan > 0 {
				line := 0.5 + 0.5*math.Cos(fy*math.Pi)
				m := 1 - scan*line
				cr, cg, cb = cr*m, cg*m, cb*m
			}
			// RGB phosphor stripes across x.
			if mask > 0 {
				switch x % 3 {
				case 0:
					cg, cb = cg*(1-mask), cb*(1-mask)
				case 1:
					cr, cb = cr*(1-mask), cb*(1-mask)
				default:
					cr, cg = cr*(1-mask), cg*(1-mask)
				}
			}
			// Edge vignette on the warped radius.
			if vig > 0 {
				f := 1 - vig*smoothstep(0.5, 1.6, r2)
				cr, cg, cb = cr*f, cg*f, cb*f
			}

			dst.Pix[di] = clampU8(cr)
			dst.Pix[di+1] = clampU8(cg)
			dst.Pix[di+2] = clampU8(cb)
			di += 4
			si += 4
		}
	}
	return dst, nil
}

func init() { Register("crt", CRT) }
