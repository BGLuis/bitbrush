package filters

import (
	"image"
	"image/color"
	"math"
)

// ColorOverlay blends a single flat colour over the whole image with a
// chosen blend mode and opacity — the classic photo "colour wash".
//
// Params:
//
//	color    string, hex overlay colour (default "#ff8800")
//	mode     string, "normal" (default), "multiply", "screen", "overlay" or "color"
//	opacity  float,  0..1 mix of the blended result over the original (default 0.3)
func ColorOverlay(src *image.RGBA, p Params) (*image.RGBA, error) {
	ov := hexOr(p.String("color", "#ff8800"), color.RGBA{R: 255, G: 136, A: 255})
	mode := p.String("mode", "normal")
	opacity := clampF(p.Float("opacity", 0.3), 0, 1)

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	or, og, ob := float64(ov.R)/255, float64(ov.G)/255, float64(ov.B)/255
	ovLuma := 0.2126*or + 0.7152*og + 0.0722*ob

	blend := func(base, over float64) float64 {
		switch mode {
		case "multiply":
			return base * over
		case "screen":
			return 1 - (1-base)*(1-over)
		case "overlay":
			if base < 0.5 {
				return 2 * base * over
			}
			return 1 - 2*(1-base)*(1-over)
		default: // "normal"
			return over
		}
	}

	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			br := float64(src.Pix[si]) / 255
			bg := float64(src.Pix[si+1]) / 255
			bb := float64(src.Pix[si+2]) / 255

			var rr, rg, rb float64
			if mode == "color" {
				// Keep the base luma, take the overlay's hue: scale the
				// overlay so its luma matches this pixel's.
				bl := 0.2126*br + 0.7152*bg + 0.0722*bb
				k := 0.0
				if ovLuma > 1e-6 {
					k = bl / ovLuma
				}
				rr, rg, rb = or*k, og*k, ob*k
			} else {
				rr, rg, rb = blend(br, or), blend(bg, og), blend(bb, ob)
			}

			dst.Pix[di] = clampU8(math.Round((br + (rr-br)*opacity) * 255))
			dst.Pix[di+1] = clampU8(math.Round((bg + (rg-bg)*opacity) * 255))
			dst.Pix[di+2] = clampU8(math.Round((bb + (rb-bb)*opacity) * 255))
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			di += 4
		}
	}
	return dst, nil
}

func init() { Register("coloroverlay", ColorOverlay) }
