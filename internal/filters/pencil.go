package filters

import (
	"image"
	"image/color"
	"math"
)

// Pencil renders src as a black-graphite pencil drawing. It is the classic
// dodge sketch: grey the image, blur an inverted copy, then colour-dodge the
// grey by that blur so flat areas burn out to paper white and only the
// tonal edges survive as strokes. Optional diagonal graphite hatching fills
// the mid and shadow tones, and the line art is laid onto a paper colour.
//
// Params:
//
//	blur      float,  1..40 dodge blur radius — larger = softer, broader
//	          strokes (default 8)
//	strength  float,  0..1 mix between the flat grey and the full sketch
//	          (default 1)
//	darkness  float,  0.3..4 gamma on the stroke tone; >1 deepens lines
//	          (default 1)
//	hatch     float,  0..1 diagonal graphite hatching in the darker tones
//	          (default 0.3)
//	seed      int,    hatching grain seed (default 0)
//	graphite  string, hex darkest pencil tone (default "#1b1b1b")
//	paper     string, hex paper colour (default "#f6f3ea")
//
// Deterministic: no rand, no time. Alpha is carried through; src is not
// mutated.
func Pencil(src *image.RGBA, p Params) (*image.RGBA, error) {
	blur := clampF(p.Float("blur", 8), 0, 40)
	strength := clampF(p.Float("strength", 1), 0, 1)
	darkness := clampF(p.Float("darkness", 1), 0.3, 4)
	hatch := clampF(p.Float("hatch", 0.3), 0, 1)
	seed := p.Int("seed", 0)
	graphite := hexOr(p.String("graphite", "#1b1b1b"), color.RGBA{27, 27, 27, 255})
	paper := hexOr(p.String("paper", "#f6f3ea"), color.RGBA{246, 243, 234, 255})

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	gray := make([]float64, w*h)
	inv := make([]float64, w*h)
	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		for x := 0; x < w; x++ {
			g := luma(src.Pix[si], src.Pix[si+1], src.Pix[si+2]) / 255
			gray[y*w+x] = g
			inv[y*w+x] = 1 - g
			si += 4
		}
	}
	blurred := gaussianBlurPlane(inv, w, h, blur/2)

	gr, gg, gb := float64(graphite.R)/255, float64(graphite.G)/255, float64(graphite.B)/255
	pr, pg, pb := float64(paper.R)/255, float64(paper.G)/255, float64(paper.B)/255

	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			idx := y*w + x
			g := gray[idx]

			// Colour-dodge: grey / (1 - blurred-inverted).
			d := 1.0
			if den := 1 - blurred[idx]; den > 1e-4 {
				d = g / den
			}
			d = math.Pow(clamp01(d), darkness)
			d = 1 - strength*(1-d)

			// Diagonal graphite hatching in the darker tones.
			if hatch > 0 {
				if shade := 1 - g; shade > 0.15 {
					n := valueNoise2D(float64(x)/40, float64(y)/40, seed+5)
					l1 := 0.5 + 0.5*math.Cos((float64(x+y)+12*n)*0.7)
					l2 := 0.5 + 0.5*math.Cos((float64(x-y)+12*n)*0.5)
					line := math.Min(l1, l2)
					grit := 0.85 + 0.15*hashNoise(x, y, seed+31)
					d = clamp01(d - hatch*(shade-0.15)*1.15*(1-line)*grit)
				}
			}

			dst.Pix[di] = clampU8((gr + d*(pr-gr)) * 255)
			dst.Pix[di+1] = clampU8((gg + d*(pg-gg)) * 255)
			dst.Pix[di+2] = clampU8((gb + d*(pb-gb)) * 255)
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			di += 4
		}
	}
	return dst, nil
}

func init() { Register("pencil", Pencil) }

// gaussianBlurPlane returns src (a row-major w*h plane) convolved with a
// separable Gaussian of the given sigma, replicating the border. sigma <= 0
// returns a copy unchanged.
func gaussianBlurPlane(src []float64, w, h int, sigma float64) []float64 {
	out := make([]float64, len(src))
	if sigma <= 0 || w == 0 || h == 0 {
		copy(out, src)
		return out
	}

	radius := int(math.Ceil(sigma * 3))
	if radius < 1 {
		radius = 1
	}
	k := make([]float64, radius+1) // k[0] is the centre tap; symmetric
	var sum float64
	for i := 0; i <= radius; i++ {
		v := math.Exp(-float64(i*i) / (2 * sigma * sigma))
		k[i] = v
		if i == 0 {
			sum += v
		} else {
			sum += 2 * v
		}
	}
	for i := range k {
		k[i] /= sum
	}

	tmp := make([]float64, w*h)
	for y := 0; y < h; y++ {
		row := y * w
		for x := 0; x < w; x++ {
			acc := src[row+x] * k[0]
			for i := 1; i <= radius; i++ {
				xl := x - i
				if xl < 0 {
					xl = 0
				}
				xr := x + i
				if xr >= w {
					xr = w - 1
				}
				acc += (src[row+xl] + src[row+xr]) * k[i]
			}
			tmp[row+x] = acc
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			acc := tmp[y*w+x] * k[0]
			for i := 1; i <= radius; i++ {
				yl := y - i
				if yl < 0 {
					yl = 0
				}
				yr := y + i
				if yr >= h {
					yr = h - 1
				}
				acc += (tmp[yl*w+x] + tmp[yr*w+x]) * k[i]
			}
			out[y*w+x] = acc
		}
	}
	return out
}
