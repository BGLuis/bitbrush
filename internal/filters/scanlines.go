package filters

import "image"

// Scanlines darkens evenly spaced horizontal lines, the CRT / VHS look.
//
// Params:
//
//	spacing    int,   px between the centre of one dark line and the next (default 3)
//	thickness  int,   px height of each dark line (default 1)
//	darkness   float, 0..1 how much a line row is dimmed (default 0.5)
//	opacity    float, 0..1 overall mix of the effect (default 1.0)
func Scanlines(src *image.RGBA, p Params) (*image.RGBA, error) {
	spacing := clampInt(p.Int("spacing", 3), 2, 1<<20)
	thickness := clampInt(p.Int("thickness", 1), 1, spacing)
	darkness := clampF(p.Float("darkness", 0.5), 0, 1)
	opacity := clampF(p.Float("opacity", 1.0), 0, 1)

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := cloneRGBA(src)
	if w == 0 || h == 0 {
		return dst, nil
	}
	mul := 1 - darkness*opacity

	for y := 0; y < h; y++ {
		if y%spacing >= thickness {
			continue
		}
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			dst.Pix[di] = uint8(float64(dst.Pix[di]) * mul)
			dst.Pix[di+1] = uint8(float64(dst.Pix[di+1]) * mul)
			dst.Pix[di+2] = uint8(float64(dst.Pix[di+2]) * mul)
			di += 4
		}
	}
	return dst, nil
}

func init() { Register("scanlines", Scanlines) }
