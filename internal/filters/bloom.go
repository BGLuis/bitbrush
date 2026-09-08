package filters

import "image"

// Bloom extracts the bright parts of src, blurs them and screen-blends the
// halo back — the glow around highlights (and around bright glyphs, the
// "character bloom" look, with a lower threshold).
//
// Params:
//
//	threshold  float, 0..1 luma above which a pixel contributes to the glow (default 0.75)
//	radius     int,   blur radius of the halo px (default 12)
//	intensity  float, 0..2 how strongly the halo is added back (default 0.8)
func Bloom(src *image.RGBA, p Params) (*image.RGBA, error) {
	threshold := clampF(p.Float("threshold", 0.75), 0, 1)
	radius := clampInt(p.Int("radius", 12), 0, 1<<12)
	intensity := clampF(p.Float("intensity", 0.8), 0, 4)

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	// Bright pass -> its own opaque image so the blur doesn't bleed
	// transparency in.
	cut := threshold * 255
	bright := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		bi := bright.PixOffset(0, y)
		for x := 0; x < w; x++ {
			r, g, bl := src.Pix[si], src.Pix[si+1], src.Pix[si+2]
			if luma(r, g, bl) >= cut {
				bright.Pix[bi], bright.Pix[bi+1], bright.Pix[bi+2] = r, g, bl
			}
			bright.Pix[bi+3] = 255
			si += 4
			bi += 4
		}
	}
	halo := boxBlur(bright, radius)

	// screen(base, halo*intensity), alpha carried from src.
	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		hi := halo.PixOffset(0, y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			for c := 0; c < 3; c++ {
				base := float64(src.Pix[si+c]) / 255
				add := float64(halo.Pix[hi+c]) / 255 * intensity
				if add > 1 {
					add = 1
				}
				dst.Pix[di+c] = clampU8((1 - (1-base)*(1-add)) * 255)
			}
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			hi += 4
			di += 4
		}
	}
	return dst, nil
}

func init() { Register("bloom", Bloom) }
