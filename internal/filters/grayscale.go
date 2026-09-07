package filters

import (
	"image"
	"math"

	"bitbrush/internal/colorspace"
)

// Grayscale desaturates src to a single tone, with an optional brightness /
// contrast trim and a hard threshold for a pure 1-bit black-and-white result.
//
// Params:
//
//	method     string, tone weighting:
//	           "luma" (Rec.709, gamma-encoded — the default),
//	           "luminance" (linear-light, gamma-correct average),
//	           "average" (flat R+G+B mean),
//	           "bt601" (legacy NTSC/TV weights),
//	           "lightness" (HSL midrange, (max+min)/2)
//	brightness float,  -100..100, added after desaturation (default 0)
//	contrast   float,  -100..100, S-curve around mid-grey (default 0)
//	threshold  float,  0 keeps the continuous ramp; >0 snaps every pixel to
//	           black or white at this cut on the 0..255 scale (default 0)
//	invert     bool,   output 255-v (default false)
//
// Alpha is carried through per pixel; src is not mutated.
func Grayscale(src *image.RGBA, p Params) (*image.RGBA, error) {
	method := p.String("method", "luma")
	brightness := clampF(p.Float("brightness", 0), -100, 100)
	contrast := clampF(p.Float("contrast", 0), -100, 100)
	threshold := p.Float("threshold", 0)
	invert := p.Bool("invert", false)

	// Classic GIMP-style contrast factor around 128.
	cf := (259 * (contrast + 255)) / (255 * (259 - contrast))

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)

	// Resolve the tone rule to a plain function once, not per pixel.
	tone := toneFunc(method)

	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			v := tone(src.Pix[si], src.Pix[si+1], src.Pix[si+2])
			v = cf*(v-128) + 128 + brightness

			var out uint8
			if threshold > 0 {
				if v >= threshold {
					out = 255
				}
			} else {
				out = clampU8(v)
			}
			if invert {
				out = 255 - out
			}

			dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2], dst.Pix[di+3] = out, out, out, src.Pix[si+3]
			si += 4
			di += 4
		}
	}
	return dst, nil
}

func init() { Register("grayscale", Grayscale) }

// toneFunc returns the 0..255 tone reducer named by method, resolved once so
// the pixel loop pays an indirect call instead of a string switch per pixel.
func toneFunc(method string) func(r, g, b uint8) float64 {
	switch method {
	case "average":
		return func(r, g, b uint8) float64 {
			return (float64(r) + float64(g) + float64(b)) / 3
		}
	case "bt601":
		return func(r, g, b uint8) float64 {
			return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
		}
	case "lightness":
		return func(r, g, b uint8) float64 {
			hi := math.Max(float64(r), math.Max(float64(g), float64(b)))
			lo := math.Min(float64(r), math.Min(float64(g), float64(b)))
			return (hi + lo) / 2
		}
	case "luminance":
		return func(r, g, b uint8) float64 {
			y := 0.2126*colorspace.Linearize8(r) + 0.7152*colorspace.Linearize8(g) + 0.0722*colorspace.Linearize8(b)
			return float64(colorspace.Encode8(y))
		}
	default: // "luma"
		return luma
	}
}
