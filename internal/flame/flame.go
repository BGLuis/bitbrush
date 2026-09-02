// Package flame renders fractal flames (Scott Draves): an iterated function
// system of affine maps composed with nonlinear "variations", visited by
// the chaos game and tone-mapped with a log-density curve. The whole
// transform set is derived from the seed, so a seed + params always
// reproduces the same image.
package flame

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"

	"bitbrush/internal/generators"
)

type params struct {
	Seed       int64   `json:"seed"`
	Transforms int     `json:"transforms"` // 2..6
	Iterations int     `json:"iterations"`
	Gamma      float64 `json:"gamma"`
	Vibrancy   float64 `json:"vibrancy"`
	Symmetry   int     `json:"symmetry"` // n-fold rotational, 0/1 = none
	HueSpread  float64 `json:"hueSpread"`
	Background string  `json:"background"`
}

func defaults() params {
	return params{
		Transforms: 3, Iterations: 2_000_000, Gamma: 2.4, Vibrancy: 0.85,
		HueSpread: 0.7, Background: "#050507",
	}
}

var variationNames = []string{
	"linear", "sinusoidal", "spherical", "swirl",
	"horseshoe", "polar", "handkerchief", "disc",
}

type xform struct {
	a, b, c, d, e, f float64
	varW             []float64
	weight           float64
	color            float64 // palette coordinate [0,1]
	baseHue          float64
}

func render(raw json.RawMessage, w, h int) (*image.RGBA, error) {
	p := defaults()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("flame: bad params: %w", err)
		}
	}
	p.Transforms = clampi(p.Transforms, 2, 6)
	p.Iterations = clampi(p.Iterations, 100_000, 20_000_000)
	p.Gamma = clampf(p.Gamma, 0.5, 6)
	p.Vibrancy = clampf(p.Vibrancy, 0, 1)
	p.Symmetry = clampi(p.Symmetry, 0, 12)
	p.HueSpread = clampf(p.HueSpread, 0, 1)

	rng := rand.New(rand.NewSource(p.Seed))
	baseHue := rng.Float64()
	xf := make([]xform, p.Transforms)
	var wsum float64
	for i := range xf {
		r := func() float64 { return rng.Float64()*1.6 - 0.8 }
		vw := make([]float64, len(variationNames))
		// 1–3 active variations per transform
		active := 1 + rng.Intn(3)
		for k := 0; k < active; k++ {
			vw[rng.Intn(len(vw))] += rng.Float64()
		}
		normalize(vw)
		xf[i] = xform{
			a: r(), b: r(), c: rng.Float64()*1.2 - 0.6,
			d: r(), e: r(), f: rng.Float64()*1.2 - 0.6,
			varW:   vw,
			weight: 0.15 + rng.Float64(),
			color:  rng.Float64(),
		}
		xf[i].baseHue = baseHue + xf[i].color*p.HueSpread
		wsum += xf[i].weight
	}
	// cumulative weights for selection
	cum := make([]float64, len(xf))
	acc := 0.0
	for i := range xf {
		acc += xf[i].weight / wsum
		cum[i] = acc
	}

	pick := func() *xform {
		u := rng.Float64()
		for i := range cum {
			if u <= cum[i] {
				return &xf[i]
			}
		}
		return &xf[len(xf)-1]
	}

	step := func(x, y, col float64, t *xform) (float64, float64, float64) {
		px := t.a*x + t.b*y + t.c
		py := t.d*x + t.e*y + t.f
		var vx, vy float64
		for k, wv := range t.varW {
			if wv == 0 {
				continue
			}
			ax, ay := variation(k, px, py)
			vx += wv * ax
			vy += wv * ay
		}
		return vx, vy, (col + t.color) / 2
	}

	// Bounds pass to frame the camera.
	x, y, col := rng.Float64()*2-1, rng.Float64()*2-1, rng.Float64()
	for i := 0; i < 40; i++ {
		x, y, col = step(x, y, col, pick())
	}
	minX, minY, maxX, maxY := x, y, x, y
	bx, by, bc := x, y, col
	good := 0
	for i := 0; i < 40_000; i++ {
		bx, by, bc = step(bx, by, bc, pick())
		if !finite(bx) || !finite(by) {
			bx, by = rng.Float64()*2-1, rng.Float64()*2-1
			continue
		}
		good++
		minX, maxX = math.Min(minX, bx), math.Max(maxX, bx)
		minY, maxY = math.Min(minY, by), math.Max(maxY, by)
	}
	if good < 1000 {
		return nil, fmt.Errorf("flame: transform set does not converge; try another seed")
	}
	spanX := math.Max(maxX-minX, 1e-6)
	spanY := math.Max(maxY-minY, 1e-6)
	cxData, cyData := (minX+maxX)/2, (minY+maxY)/2
	scale := 0.88 * math.Min(float64(w)/spanX, float64(h)/spanY)
	cx, cy := float64(w)/2, float64(h)/2

	sym := p.Symmetry
	if sym < 2 {
		sym = 1
	}
	rot := make([][2]float64, sym) // cos/sin per symmetry copy
	for k := 0; k < sym; k++ {
		ang := 2 * math.Pi * float64(k) / float64(sym)
		rot[k] = [2]float64{math.Cos(ang), math.Sin(ang)}
	}

	accm := generators.NewAccumulator(w, h)
	for i := 0; i < p.Iterations; i++ {
		t := pick()
		x, y, col = step(x, y, col, t)
		if !finite(x) || !finite(y) {
			x, y, col = rng.Float64()*2-1, rng.Float64()*2-1, rng.Float64()
			continue
		}
		if i < 20 {
			continue
		}
		dx, dy := x-cxData, y-cyData
		r, g, b := hsv(t.baseHue+col*0.15, 0.62, 1)
		for _, cs := range rot {
			rx := dx*cs[0] - dy*cs[1]
			ry := dx*cs[1] + dy*cs[0]
			accm.Add(int(cx+rx*scale), int(cy+ry*scale), r, g, b)
		}
	}

	bg := generators.ParseHex(p.Background, color.RGBA{5, 5, 7, 255})
	return accm.ToImage(bg, color.RGBA{255, 255, 255, 255}, p.Gamma, p.Vibrancy), nil
}

func variation(k int, x, y float64) (float64, float64) {
	r2 := x*x + y*y
	r := math.Sqrt(r2)
	switch k {
	case 1: // sinusoidal
		return math.Sin(x), math.Sin(y)
	case 2: // spherical
		if r2 < 1e-9 {
			r2 = 1e-9
		}
		return x / r2, y / r2
	case 3: // swirl
		s, c := math.Sin(r2), math.Cos(r2)
		return x*s - y*c, x*c + y*s
	case 4: // horseshoe
		if r < 1e-9 {
			r = 1e-9
		}
		return (x - y) * (x + y) / r, 2 * x * y / r
	case 5: // polar
		return math.Atan2(x, y) / math.Pi, r - 1
	case 6: // handkerchief
		th := math.Atan2(x, y)
		return r * math.Sin(th+r), r * math.Cos(th-r)
	case 7: // disc
		th := math.Atan2(x, y) / math.Pi
		return th * math.Sin(math.Pi*r), th * math.Cos(math.Pi*r)
	default: // linear
		return x, y
	}
}

func normalize(v []float64) {
	var s float64
	for _, x := range v {
		s += x
	}
	if s == 0 {
		v[0] = 1
		return
	}
	for i := range v {
		v[i] /= s
	}
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && math.Abs(v) < 1e6
}

func hsv(h, s, v float64) (float64, float64, float64) {
	h = math.Mod(math.Mod(h, 1)+1, 1) * 6
	i := math.Floor(h)
	f := h - i
	pp := v * (1 - s)
	q := v * (1 - s*f)
	tt := v * (1 - s*(1-f))
	switch int(i) % 6 {
	case 0:
		return v, tt, pp
	case 1:
		return q, v, pp
	case 2:
		return pp, v, tt
	case 3:
		return pp, q, v
	case 4:
		return tt, pp, v
	default:
		return v, pp, q
	}
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func init() { generators.Register("flame", render) }
