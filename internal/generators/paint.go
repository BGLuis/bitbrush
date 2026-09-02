package generators

import (
	"image"
	"image/color"
	"math"
)

// Small shared drawing helpers for the generator packages. Nothing here is
// clever — just the handful of operations (fill, alpha splat, additive
// accumulation) every generator needs, in one place.

// ParseHex reads "#rgb", "#rrggbb" or "#rrggbbaa" (leading '#' optional),
// returning def on anything it can't parse.
func ParseHex(s string, def color.RGBA) color.RGBA {
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	hx := func(b byte) (int, bool) {
		switch {
		case b >= '0' && b <= '9':
			return int(b - '0'), true
		case b >= 'a' && b <= 'f':
			return int(b-'a') + 10, true
		case b >= 'A' && b <= 'F':
			return int(b-'A') + 10, true
		}
		return 0, false
	}
	get := func(hi, lo byte) (uint8, bool) {
		a, ok1 := hx(hi)
		b, ok2 := hx(lo)
		if !ok1 || !ok2 {
			return 0, false
		}
		return uint8(a<<4 | b), true
	}
	switch len(s) {
	case 3:
		r, ok1 := hx(s[0])
		g, ok2 := hx(s[1])
		b, ok3 := hx(s[2])
		if !ok1 || !ok2 || !ok3 {
			return def
		}
		return color.RGBA{uint8(r * 17), uint8(g * 17), uint8(b * 17), 255}
	case 6, 8:
		r, ok1 := get(s[0], s[1])
		g, ok2 := get(s[2], s[3])
		b, ok3 := get(s[4], s[5])
		if !ok1 || !ok2 || !ok3 {
			return def
		}
		a := uint8(255)
		if len(s) == 8 {
			av, ok := get(s[6], s[7])
			if !ok {
				return def
			}
			a = av
		}
		return color.RGBA{r, g, b, a}
	}
	return def
}

// NewCanvas returns an opaque w x h image flat-filled with bg.
func NewCanvas(w, h int, bg color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	Fill(img, bg)
	return img
}

// Fill paints every pixel of img with c.
func Fill(img *image.RGBA, c color.RGBA) {
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
}

func clampU8f(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 255:
		return 255
	}
	return uint8(v + 0.5)
}

// SplatAA alpha-blends c onto img at fractional (x, y) with weight a in
// [0,1], spread bilinearly over the four surrounding pixels. Out-of-range
// coordinates are clipped. The alpha channel is left opaque.
func SplatAA(img *image.RGBA, x, y, a float64, c color.RGBA) {
	if a <= 0 {
		return
	}
	if a > 1 {
		a = 1
	}
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	fx := x - float64(x0)
	fy := y - float64(y0)
	w := img.Rect.Dx()
	h := img.Rect.Dy()
	put := func(px, py int, wt float64) {
		if px < 0 || py < 0 || px >= w || py >= h || wt <= 0 {
			return
		}
		o := img.PixOffset(px, py)
		k := a * wt
		img.Pix[o] = clampU8f(float64(img.Pix[o])*(1-k) + float64(c.R)*k)
		img.Pix[o+1] = clampU8f(float64(img.Pix[o+1])*(1-k) + float64(c.G)*k)
		img.Pix[o+2] = clampU8f(float64(img.Pix[o+2])*(1-k) + float64(c.B)*k)
	}
	put(x0, y0, (1-fx)*(1-fy))
	put(x0+1, y0, fx*(1-fy))
	put(x0, y0+1, (1-fx)*fy)
	put(x0+1, y0+1, fx*fy)
}

// Line draws an anti-aliased line from (x0,y0) to (x1,y1) by splatting
// along it. width scales the per-sample alpha, not the geometry.
func Line(img *image.RGBA, x0, y0, x1, y1, alpha float64, c color.RGBA) {
	dx := x1 - x0
	dy := y1 - y0
	steps := math.Hypot(dx, dy)
	n := int(steps) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		SplatAA(img, x0+dx*t, y0+dy*t, alpha, c)
	}
}

// Accumulator is a float RGB buffer for density plots (strange attractors,
// fractal flames): hits are summed, then ToImage tone-maps the whole field
// once at the end.
type Accumulator struct {
	W, H       int
	R, G, B, N []float64
}

func NewAccumulator(w, h int) *Accumulator {
	return &Accumulator{W: w, H: h,
		R: make([]float64, w*h), G: make([]float64, w*h),
		B: make([]float64, w*h), N: make([]float64, w*h)}
}

// Add deposits colour (r,g,b in [0,1]) at integer pixel (x,y).
func (a *Accumulator) Add(x, y int, r, g, b float64) {
	if x < 0 || y < 0 || x >= a.W || y >= a.H {
		return
	}
	i := y*a.W + x
	a.R[i] += r
	a.G[i] += g
	a.B[i] += b
	a.N[i]++
}

// ToImage tone-maps the accumulator with a log density curve (the standard
// fractal-flame mapping) over bg. gamma > 0 lifts the shadows; vibrancy in
// [0,1] blends per-hit colour (1) against a flat ink density (0).
func (a *Accumulator) ToImage(bg, ink color.RGBA, gamma, vibrancy float64) *image.RGBA {
	img := NewCanvas(a.W, a.H, bg)
	maxN := 0.0
	for _, n := range a.N {
		if n > maxN {
			maxN = n
		}
	}
	if maxN <= 0 {
		return img
	}
	logMax := math.Log(maxN + 1)
	invGamma := 1.0
	if gamma > 0 {
		invGamma = 1.0 / gamma
	}
	for i := 0; i < len(a.N); i++ {
		n := a.N[i]
		if n <= 0 {
			continue
		}
		d := math.Log(n+1) / logMax // 0..1
		d = math.Pow(d, invGamma)
		var cr, cg, cb float64
		if a.R[i]+a.G[i]+a.B[i] > 0 {
			cr = a.R[i] / n
			cg = a.G[i] / n
			cb = a.B[i] / n
		}
		fr := vibrancy*cr + (1-vibrancy)*float64(ink.R)/255
		fg := vibrancy*cg + (1-vibrancy)*float64(ink.G)/255
		fb := vibrancy*cb + (1-vibrancy)*float64(ink.B)/255
		o := i * 4
		img.Pix[o] = clampU8f(float64(bg.R)*(1-d) + fr*255*d)
		img.Pix[o+1] = clampU8f(float64(bg.G)*(1-d) + fg*255*d)
		img.Pix[o+2] = clampU8f(float64(bg.B)*(1-d) + fb*255*d)
	}
	return img
}
