package filters

import "image"

// Grain adds deterministic film grain: a per-pixel (or per size×size block)
// value drawn from a seeded hash, added to the image.
//
// Params:
//
//	amount  float, 0..1 grain strength (default 0.12)
//	size    int,   grain cell in px, larger = coarser (default 1)
//	seed    int,   hash seed (default 1)
//	mono    bool,  one grain value for all channels vs per-channel (default true)
func Grain(src *image.RGBA, p Params) (*image.RGBA, error) {
	amount := clampF(p.Float("amount", 0.12), 0, 1)
	size := clampInt(p.Int("size", 1), 1, 1<<20)
	seed := p.Int("seed", 1)
	mono := p.Bool("mono", true)

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}
	amp := amount * 255

	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		gy := y / size
		for x := 0; x < w; x++ {
			gx := x / size
			n := (hashNoise(gx, gy, seed) - 0.5) * amp
			nr, ng, nb := n, n, n
			if !mono {
				ng = (hashNoise(gx, gy, seed+101) - 0.5) * amp
				nb = (hashNoise(gx, gy, seed+202) - 0.5) * amp
			}
			dst.Pix[di] = clampU8(float64(src.Pix[si]) + nr)
			dst.Pix[di+1] = clampU8(float64(src.Pix[si+1]) + ng)
			dst.Pix[di+2] = clampU8(float64(src.Pix[si+2]) + nb)
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			di += 4
		}
	}
	return dst, nil
}

func init() { Register("grain", Grain) }
