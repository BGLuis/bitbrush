package filters

import (
	"image"
	"image/color"
	"math"
)

// Paper re-prints src onto a sheet of textured, optionally aged paper: the
// image is toned toward sepia, multiplied through a warm paper colour, then
// modulated by procedural fibre grain, large-scale mottling, directional
// fibres, sparse dark specks and a soft edge vignette.
//
// The texture is pure value noise keyed on (x, y, seed) — no rand, no time —
// so a given seed + params reproduce the same sheet exactly.
//
// Params:
//
//	seed      int,    texture seed (default 0)
//	paper     string, hex paper colour (default "#f3ecd8", a warm cream)
//	age       float,  0..1 sepia toning + ink fade (default 0.5)
//	grain     float,  0..1 fine fibre grain strength (default 0.35)
//	mottle    float,  0..1 large soft blotchiness (default 0.3)
//	fibers    float,  0..1 vertical fibre streaking (default 0.2)
//	scale     float,  0.5..8 texture feature-size multiplier (default 2)
//	vignette  float,  0..1 edge darkening (default 0.25)
//
// Alpha is carried through per pixel; src is not mutated.
func Paper(src *image.RGBA, p Params) (*image.RGBA, error) {
	seed := p.Int("seed", 0)
	paper := hexOr(p.String("paper", "#f3ecd8"), color.RGBA{243, 236, 216, 255})
	age := clampF(p.Float("age", 0.5), 0, 1)
	grain := clampF(p.Float("grain", 0.35), 0, 1)
	mottle := clampF(p.Float("mottle", 0.30), 0, 1)
	fibers := clampF(p.Float("fibers", 0.20), 0, 1)
	scale := clampF(p.Float("scale", 2), 0.5, 8)
	vignette := clampF(p.Float("vignette", 0.25), 0, 1)

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	pr, pg, pb := float64(paper.R)/255, float64(paper.G)/255, float64(paper.B)/255
	invDiag := 1 / math.Hypot(0.5, 0.5)
	fade := age * 0.08

	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		fy := float64(y)
		for x := 0; x < w; x++ {
			r, g, bl := float64(src.Pix[si]), float64(src.Pix[si+1]), float64(src.Pix[si+2])
			fx := float64(x)

			// Sepia toning, mixed in by `age`.
			sr := 0.393*r + 0.769*g + 0.189*bl
			sg := 0.349*r + 0.686*g + 0.168*bl
			sb := 0.272*r + 0.534*g + 0.131*bl
			cr := clamp01((r + age*(sr-r)) / 255)
			cg := clamp01((g + age*(sg-g)) / 255)
			cb := clamp01((bl + age*(sb-bl)) / 255)

			// Paper texture multiplier.
			mot := fractalNoise(fx/(90*scale), fy/(90*scale), 4, seed)
			grn := valueNoise2D(fx/1.7, fy/1.7, seed+7)
			fib := valueNoise2D(fx/(2.5*scale), fy/(70*scale), seed+13)
			tex := 1.0
			tex += 0.30 * mottle * (mot - 0.5) * 2
			tex += 0.12 * grain * (grn - 0.5) * 2
			tex += 0.15 * fibers * (fib - 0.5) * 2
			if hashNoise(x, y, seed+99) > 0.9975 {
				tex -= 0.45 * grain
			}
			tex = clampF(tex, 0.25, 1.15)

			// Radial vignette.
			ndx := (fx+0.5)/float64(w) - 0.5
			ndy := (fy+0.5)/float64(h) - 0.5
			d := math.Hypot(ndx, ndy) * invDiag
			k := tex * (1 - vignette*clampF(d*d*1.15-0.12, 0, 1))

			or := cr*pr*k*(1-fade) + fade*pr
			og := cg*pg*k*(1-fade) + fade*pg
			ob := cb*pb*k*(1-fade) + fade*pb

			dst.Pix[di] = clampU8(or * 255)
			dst.Pix[di+1] = clampU8(og * 255)
			dst.Pix[di+2] = clampU8(ob * 255)
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			di += 4
		}
	}
	return dst, nil
}

func init() { Register("paper", Paper) }
