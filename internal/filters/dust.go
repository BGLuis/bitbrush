package filters

import (
	"image"
	"math/rand"
)

// Dust scatters deterministic light and dark specks over the image, plus a
// few faint vertical scratches — the aged-film-print look. All randomness
// comes from `seed`.
//
// Params:
//
//	density    float, specks per pixel, 0..0.02 (default 0.002)
//	seed       int,   RNG seed (default 1)
//	scratches  bool,  add a handful of vertical scratch lines (default true)
func Dust(src *image.RGBA, p Params) (*image.RGBA, error) {
	density := clampF(p.Float("density", 0.002), 0, 0.05)
	seed := int64(p.Int("seed", 1))
	scratches := p.Bool("scratches", true)

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := cloneRGBA(src)
	if w == 0 || h == 0 {
		return dst, nil
	}
	rng := rand.New(rand.NewSource(seed))

	paint := func(x, y int, v uint8, cov float64) {
		if x < 0 || x >= w || y < 0 || y >= h {
			return
		}
		o := dst.PixOffset(x, y)
		dst.Pix[o] = lerpByte(dst.Pix[o], v, cov)
		dst.Pix[o+1] = lerpByte(dst.Pix[o+1], v, cov)
		dst.Pix[o+2] = lerpByte(dst.Pix[o+2], v, cov)
	}

	n := int(density * float64(w*h))
	for i := 0; i < n; i++ {
		x := rng.Intn(w)
		y := rng.Intn(h)
		rad := 1 + rng.Intn(3)
		var v uint8 = 255
		if rng.Float64() < 0.5 {
			v = 0
		}
		for dy := -rad; dy <= rad; dy++ {
			for dx := -rad; dx <= rad; dx++ {
				if dx*dx+dy*dy > rad*rad {
					continue
				}
				paint(x+dx, y+dy, v, 0.85)
			}
		}
	}

	if scratches {
		count := 2 + rng.Intn(3)
		for i := 0; i < count; i++ {
			x := rng.Intn(w)
			dark := rng.Float64() < 0.5
			var v uint8 = 235
			if dark {
				v = 20
			}
			cov := 0.15 + rng.Float64()*0.25
			for y := 0; y < h; y++ {
				paint(x, y, v, cov)
			}
		}
	}
	return dst, nil
}

func init() { Register("dust", Dust) }
