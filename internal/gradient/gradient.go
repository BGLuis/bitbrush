// Package gradient builds multi-stop gradients and CSS output, sampling
// colours through the spaces in package colorspace with configurable
// easing between stops. Pure Go; deterministic; covered by go test.
//
// The engine is three concerns kept separate:
//
//   - sampling: [Gradient.At] finds the two stops bracketing a position,
//     applies the local [Easing] curve, and interpolates in the configured
//     [Space] (honouring the [HueArc] for cylindrical spaces).
//   - rendering: [Gradient.Render] rasterises a linear gradient along a CSS
//     angle into an *image.RGBA.
//   - CSS: [Gradient.CSS] emits a linear/radial/conic gradient value, either
//     with a native colour-interpolation-method or as baked hard stops.
//
// [Generate] is the entry point the WASM adapter calls; [Params] /
// [ParseParams] mirror the whole configuration as plain JSON data.
package gradient

import "math"

// RGB is an 8-bit colour without alpha, matching internal/palette.RGB.
type RGB struct{ R, G, B uint8 }

// Stop is one colour anchored at a position in [0,1]. Stops are sorted by
// Pos; the first and last stop clamp the two ends of the gradient.
type Stop struct {
	Color RGB
	Pos   float64
}

// Space selects the colour space the interpolation runs in.
type Space int

const (
	// SpaceSRGB lerps the gamma-encoded 8-bit channels directly, like most
	// naive gradient tools.
	SpaceSRGB Space = iota
	// SpaceLinearRGB lerps in linear light, then re-encodes.
	SpaceLinearRGB
	// SpaceHSL lerps L and S linearly and the hue along the arc.
	SpaceHSL
	// SpaceLab lerps L, a and b in CIE Lab.
	SpaceLab
	// SpaceLCh lerps L and C linearly and the hue along the arc (CIE LCh).
	SpaceLCh
	// SpaceOKLab lerps L, a and b in OKLab. Default and recommended.
	SpaceOKLab
	// SpaceOKLCh lerps L and C linearly and the hue along the arc, then
	// clamps the result back into the sRGB gamut.
	SpaceOKLCh
)

func (s Space) cylindrical() bool {
	return s == SpaceHSL || s == SpaceLCh || s == SpaceOKLCh
}

// cssInterp is the CSS Color 4 colour-interpolation-method keyword.
func (s Space) cssInterp() string {
	switch s {
	case SpaceLinearRGB:
		return "srgb-linear"
	case SpaceHSL:
		return "hsl"
	case SpaceLab:
		return "lab"
	case SpaceLCh:
		return "lch"
	case SpaceOKLab:
		return "oklab"
	case SpaceOKLCh:
		return "oklch"
	default:
		return "srgb"
	}
}

// HueArc controls hue interpolation direction for cylindrical spaces
// (HSL, LCh, OKLCh). It matches CSS Color 4 hue-interpolation semantics.
type HueArc int

const (
	// HueShorter takes the smaller of the two arcs between the hues.
	HueShorter HueArc = iota
	// HueLonger takes the larger of the two arcs.
	HueLonger
	// HueIncreasing forces the hue to increase (mod 360).
	HueIncreasing
	// HueDecreasing forces the hue to decrease (mod 360).
	HueDecreasing
)

func (h HueArc) String() string {
	switch h {
	case HueLonger:
		return "longer"
	case HueIncreasing:
		return "increasing"
	case HueDecreasing:
		return "decreasing"
	default:
		return "shorter"
	}
}

// Gradient is a sorted stop list plus the interpolation configuration.
type Gradient struct {
	Stops  []Stop
	Space  Space
	Hue    HueArc
	Easing Easing // nil is treated as Linear{}
}

// New copies and sorts stops by Pos (stable), clamps each Pos to [0,1],
// fills a nil easing with Linear{}, and returns the gradient. It panics if
// stops is empty.
func New(stops []Stop, space Space, hue HueArc, easing Easing) *Gradient {
	if len(stops) == 0 {
		panic("gradient.New: need at least one stop")
	}
	cp := make([]Stop, len(stops))
	copy(cp, stops)
	for i := range cp {
		cp[i].Pos = clamp01(cp[i].Pos)
	}
	// Stable insertion sort — stop lists are short.
	for i := 1; i < len(cp); i++ {
		for j := i; j > 0 && cp[j-1].Pos > cp[j].Pos; j-- {
			cp[j-1], cp[j] = cp[j], cp[j-1]
		}
	}
	if easing == nil {
		easing = Linear{}
	}
	return &Gradient{Stops: cp, Space: space, Hue: hue, Easing: easing}
}

// At returns the gradient colour at position t (clamped to [0,1]). It finds
// the bracketing stops, eases the local parameter, and interpolates in the
// gradient's Space.
func (g *Gradient) At(t float64) RGB {
	t = clamp01(t)
	s := g.Stops
	if len(s) == 1 || t <= s[0].Pos {
		return s[0].Color
	}
	last := len(s) - 1
	if t >= s[last].Pos {
		return s[last].Color
	}

	i := 0
	for i < last && s[i+1].Pos < t {
		i++
	}
	lo, hi := s[i], s[i+1]
	if hi.Pos <= lo.Pos {
		return hi.Color // coincident stops => hard stop, later wins
	}

	u := (t - lo.Pos) / (hi.Pos - lo.Pos)
	ease := g.Easing
	if ease == nil {
		ease = Linear{}
	}
	u = ease.Ease(u)
	return interp(lo.Color, hi.Color, u, g.Space, g.Hue)
}

// Sample returns n colours evenly spaced over [0,1] (n is raised to 2 if
// smaller).
func (g *Gradient) Sample(n int) []RGB {
	if n < 2 {
		n = 2
	}
	out := make([]RGB, n)
	for i := range out {
		out[i] = g.At(float64(i) / float64(n-1))
	}
	return out
}

// SampleStops returns n stops with Pos = i/(n-1) and Color = At(Pos),
// suitable for baking a gradient into fixed CSS stops.
func (g *Gradient) SampleStops(n int) []Stop {
	if n < 2 {
		n = 2
	}
	out := make([]Stop, n)
	for i := range out {
		p := float64(i) / float64(n-1)
		out[i] = Stop{Color: g.At(p), Pos: p}
	}
	return out
}

// interp blends c0..c1 by u in the given space.
func interp(c0, c1 RGB, u float64, space Space, arc HueArc) RGB {
	switch space {
	case SpaceLinearRGB:
		return RGB{
			encLerp(c0.R, c1.R, u),
			encLerp(c0.G, c1.G, u),
			encLerp(c0.B, c1.B, u),
		}

	case SpaceHSL:
		a := rgbToHSL(c0)
		b := rgbToHSL(c1)
		h := interpHue(a.H, b.H, a.S, b.S, u, arc)
		r, g, bl := hslToRGB(h, lerp(a.S, b.S, u), lerp(a.L, b.L, u))
		return RGB{r, g, bl}

	case SpaceLab:
		a := rgbToLab(c0)
		b := rgbToLab(c1)
		r, g, bl := labToRGB(lerp(a.L, b.L, u), lerp(a.A, b.A, u), lerp(a.B, b.B, u))
		return RGB{r, g, bl}

	case SpaceLCh:
		a := rgbToLCh(c0)
		b := rgbToLCh(c1)
		h := interpHue(a.H, b.H, a.C, b.C, u, arc)
		r, g, bl := lchToRGB(lerp(a.L, b.L, u), lerp(a.C, b.C, u), h)
		return RGB{r, g, bl}

	case SpaceOKLab:
		a := rgbToOKLab(c0)
		b := rgbToOKLab(c1)
		r, g, bl := okLabToRGB(lerp(a.L, b.L, u), lerp(a.A, b.A, u), lerp(a.B, b.B, u))
		return RGB{r, g, bl}

	case SpaceOKLCh:
		a := rgbToOKLCh(c0)
		b := rgbToOKLCh(c1)
		h := interpHue(a.H, b.H, a.C, b.C, u, arc)
		r, g, bl := okLChToRGB(lerp(a.L, b.L, u), lerp(a.C, b.C, u), h)
		return RGB{r, g, bl}

	default: // SpaceSRGB
		return RGB{lerp8(c0.R, c1.R, u), lerp8(c0.G, c1.G, u), lerp8(c0.B, c1.B, u)}
	}
}

// interpHue blends two hue angles (degrees) along the arc selected by arc.
// c0/c1 are the matching chroma (or saturation) values: if either endpoint
// is effectively achromatic its hue is undefined, so the other endpoint's
// hue is carried instead of interpolating toward 0.
func interpHue(h0, h1, c0, c1, u float64, arc HueArc) float64 {
	const eps = 1e-4
	a0, a1 := c0 > eps, c1 > eps
	switch {
	case !a0 && !a1:
		return norm360(h0)
	case !a0:
		return norm360(h1)
	case !a1:
		return norm360(h0)
	}

	h0, h1 = norm360(h0), norm360(h1)
	d := h1 - h0
	switch arc {
	case HueLonger:
		if 0 < d && d < 180 {
			h0 += 360
		} else if -180 < d && d <= 0 {
			h1 += 360
		}
	case HueIncreasing:
		if h1 < h0 {
			h1 += 360
		}
	case HueDecreasing:
		if h0 < h1 {
			h0 += 360
		}
	default: // HueShorter
		if d > 180 {
			h0 += 360
		} else if d < -180 {
			h1 += 360
		}
	}
	return norm360(lerp(h0, h1, u))
}

func lerp(a, b, u float64) float64 { return a + (b-a)*u }

func lerp8(a, b uint8, u float64) uint8 {
	return clampByte(float64(a) + (float64(b)-float64(a))*u)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clampByte(v float64) uint8 {
	v = math.Round(v)
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v)
}

func norm360(h float64) float64 {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	return h
}
