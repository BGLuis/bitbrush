package palette

import (
	"math"

	"bitbrush/internal/colorspace"
)

// HarmonyRule names a classic colour-wheel relationship.
type HarmonyRule int

const (
	// Complementary pairs the base with its opposite (+180 degrees).
	Complementary HarmonyRule = iota
	// Analogous spreads neighbours either side of the base.
	Analogous
	// Triadic uses three hues 120 degrees apart.
	Triadic
	// Tetradic uses four hues 90 degrees apart.
	Tetradic
	// SplitComplementary uses the two hues flanking the complement.
	SplitComplementary
	// Monochromatic holds the hue fixed and varies lightness.
	Monochromatic
)

// Wheel selects which hue circle Harmony rotates in.
type Wheel int

const (
	// WheelHSL rotates hue in gamma-encoded HSL (the familiar web wheel).
	WheelHSL Wheel = iota
	// WheelOKLCh rotates hue in OKLCh, so steps are perceptually even.
	WheelOKLCh
)

// HarmonyOptions configures Harmony. The zero value rotates in HSL with a
// 30-degree spread.
type HarmonyOptions struct {
	// Wheel is the hue circle to rotate in.
	Wheel Wheel
	// SpreadDeg is the hue step for Analogous and SplitComplementary.
	// Zero means the default, 30 degrees.
	SpreadDeg float64
}

// Harmony builds an n-colour palette from base by rotating hue in the
// chosen wheel while holding lightness and chroma/saturation fixed
// (Monochromatic instead holds hue and walks lightness upward). Element 0
// is always the unmodified base. n <= 0 returns nil; n == 1 returns
// [base]. When a rule's core hues are fewer than n, the palette is padded
// with lighter/darker variants of those hues.
func Harmony(base RGB, rule HarmonyRule, n int, opt HarmonyOptions) []RGB {
	if n <= 0 {
		return nil
	}

	spread := opt.SpreadDeg
	if spread == 0 {
		spread = 30
	}
	wc := toWheel(base, opt.Wheel)

	out := make([]RGB, 0, n)
	out = append(out, base) // element 0: literal base, never round-tripped
	if n == 1 {
		return out
	}

	switch rule {
	case Monochromatic:
		lo := wc.light
		hi := lo + (1-lo)*0.6
		if hi <= lo {
			hi = math.Min(lo+0.02, 1)
		}
		for i := 1; i < n; i++ {
			t := lo + (hi-lo)*float64(i)/float64(n-1)
			out = append(out, wc.withLightness(t).rgb())
		}
		return out

	case Analogous:
		// Offsets fan out from the base: -s, +s, -2s, +2s, ...
		for i := 1; i < n; i++ {
			mag := float64((i + 1) / 2)
			off := mag * spread
			if i%2 == 1 {
				off = -off
			}
			out = append(out, wc.rotated(off).rgb())
		}
		return out
	}

	// Hue-anchor rules: offset 0 is the base and is already emitted.
	var offs []float64
	switch rule {
	case Triadic:
		offs = []float64{0, 120, 240}
	case Tetradic:
		offs = []float64{0, 90, 180, 270}
	case SplitComplementary:
		offs = []float64{0, 180 - spread, 180 + spread}
	default: // Complementary
		offs = []float64{0, 180}
	}

	anchors := make([]wheelColor, len(offs))
	for i, o := range offs {
		anchors[i] = wc.rotated(o)
	}

	// Emit anchors 1..end, then keep cycling through every anchor applying
	// an alternating lightness delta (+d, -d, +2d, -2d, ...) until full.
	idx, round := 1, 0
	for len(out) < n {
		a := anchors[idx]
		if round > 0 {
			d := float64((round+1)/2) * 0.12
			if round%2 == 0 {
				d = -d
			}
			a = a.withLightness(a.light + d)
		}
		out = append(out, a.rgb())
		if idx++; idx >= len(anchors) {
			idx, round = 0, round+1
		}
	}
	return out
}

// wheelColor is a colour decomposed for hue rotation: hue in degrees, a
// chroma-like term (HSL saturation or OKLCh chroma), and lightness. It
// remembers its wheel so rgb() reconstructs consistently.
type wheelColor struct {
	wheel  Wheel
	hue    float64
	chroma float64
	light  float64
}

func toWheel(c RGB, w Wheel) wheelColor {
	if w == WheelOKLCh {
		lch := colorspace.OKLabToOKLCh(colorspace.RGBToOKLab(c.R, c.G, c.B))
		return wheelColor{wheel: w, hue: lch.H, chroma: lch.C, light: lch.L}
	}
	hsl := colorspace.RGBToHSL(c.R, c.G, c.B)
	return wheelColor{wheel: w, hue: hsl.H, chroma: hsl.S, light: hsl.L}
}

func (wc wheelColor) rotated(deltaDeg float64) wheelColor {
	n := wc
	n.hue = math.Mod(math.Mod(wc.hue+deltaDeg, 360)+360, 360)
	return n
}

func (wc wheelColor) withLightness(l float64) wheelColor {
	n := wc
	n.light = clamp01(l)
	return n
}

func (wc wheelColor) rgb() RGB {
	if wc.wheel == WheelOKLCh {
		g := colorspace.ClampToGamut(colorspace.OKLCh{L: wc.light, C: wc.chroma, H: wc.hue})
		r, gg, b := colorspace.OKLabToRGB(colorspace.OKLChToOKLab(g))
		return RGB{r, gg, b}
	}
	r, g, b := colorspace.HSLToRGB(colorspace.HSL{H: wc.hue, S: wc.chroma, L: wc.light})
	return RGB{r, g, b}
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
