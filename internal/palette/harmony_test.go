package palette

import (
	"math"
	"slices"
	"testing"

	"bitbrush/internal/colorspace"
)

func baseColour() RGB {
	r, g, b := colorspace.HSLToRGB(colorspace.HSL{H: 210, S: 0.5, L: 0.5})
	return RGB{r, g, b}
}

func wheelHue(c RGB, w Wheel) float64 {
	if w == WheelOKLCh {
		return colorspace.OKLabToOKLCh(colorspace.RGBToOKLab(c.R, c.G, c.B)).H
	}
	return colorspace.RGBToHSL(c.R, c.G, c.B).H
}

func wheelLight(c RGB, w Wheel) float64 {
	if w == WheelOKLCh {
		return colorspace.RGBToOKLab(c.R, c.G, c.B).L
	}
	return colorspace.RGBToHSL(c.R, c.G, c.B).L
}

// angDelta is the absolute smallest-arc difference between two hue angles.
func angDelta(a, b float64) float64 {
	d := math.Mod(a-b, 360)
	if d < -180 {
		d += 360
	}
	if d > 180 {
		d -= 360
	}
	return math.Abs(d)
}

func TestHarmonyElementZeroAndCount(t *testing.T) {
	base := baseColour()
	rules := []HarmonyRule{
		Complementary, Analogous, Triadic, Tetradic, SplitComplementary, Monochromatic,
	}
	for _, w := range []Wheel{WheelHSL, WheelOKLCh} {
		for _, r := range rules {
			for n := 1; n <= 7; n++ {
				got := Harmony(base, r, n, HarmonyOptions{Wheel: w})
				if len(got) != n {
					t.Fatalf("wheel %d rule %d n=%d: len=%d", w, r, n, len(got))
				}
				if got[0] != base {
					t.Fatalf("wheel %d rule %d n=%d: element 0 = %+v, want base %+v",
						w, r, n, got[0], base)
				}
			}
		}
	}
}

func TestHarmonyNilAndSingle(t *testing.T) {
	base := baseColour()
	if Harmony(base, Triadic, 0, HarmonyOptions{}) != nil {
		t.Fatal("n=0 -> nil")
	}
	if Harmony(base, Triadic, -1, HarmonyOptions{}) != nil {
		t.Fatal("n<0 -> nil")
	}
	one := Harmony(base, Triadic, 1, HarmonyOptions{})
	if len(one) != 1 || one[0] != base {
		t.Fatalf("n=1 -> %+v, want [base]", one)
	}
}

func TestHarmonyComplementary(t *testing.T) {
	base := baseColour()
	for _, w := range []Wheel{WheelHSL, WheelOKLCh} {
		got := Harmony(base, Complementary, 2, HarmonyOptions{Wheel: w})
		h0, h1 := wheelHue(got[0], w), wheelHue(got[1], w)
		tol := 2.0
		if w == WheelOKLCh {
			tol = 8.0
		}
		if d := angDelta(h1, h0+180); d > tol {
			t.Fatalf("wheel %d: complement hue off by %.2f (h0=%.1f h1=%.1f)", w, d, h0, h1)
		}
	}
}

func TestHarmonyComplementaryVariantsRespectN(t *testing.T) {
	base := baseColour()
	got := Harmony(base, Complementary, 5, HarmonyOptions{Wheel: WheelHSL})
	if len(got) != 5 {
		t.Fatalf("len=%d, want 5", len(got))
	}
	// Element 1 is still the pure complement.
	if d := angDelta(wheelHue(got[1], WheelHSL), wheelHue(got[0], WheelHSL)+180); d > 2 {
		t.Fatalf("complement hue off by %.2f", d)
	}
}

func TestHarmonyTriadic(t *testing.T) {
	base := baseColour()
	got := Harmony(base, Triadic, 3, HarmonyOptions{Wheel: WheelHSL})
	h0 := wheelHue(got[0], WheelHSL)
	for i, off := range []float64{0, 120, 240} {
		if d := angDelta(wheelHue(got[i], WheelHSL), h0+off); d > 2 {
			t.Fatalf("triadic entry %d hue off by %.2f", i, d)
		}
	}
}

func TestHarmonySplitComplementary(t *testing.T) {
	base := baseColour()
	got := Harmony(base, SplitComplementary, 3, HarmonyOptions{Wheel: WheelHSL, SpreadDeg: 30})
	h0 := wheelHue(got[0], WheelHSL)
	for i, off := range []float64{0, 150, 210} {
		if d := angDelta(wheelHue(got[i], WheelHSL), h0+off); d > 2 {
			t.Fatalf("split-complementary entry %d hue off by %.2f", i, d)
		}
	}
}

func TestHarmonyAnalogous(t *testing.T) {
	base := baseColour()
	got := Harmony(base, Analogous, 5, HarmonyOptions{Wheel: WheelHSL, SpreadDeg: 20})
	h0 := wheelHue(got[0], WheelHSL)

	offs := make([]float64, 0, len(got))
	for _, c := range got {
		d := math.Mod(wheelHue(c, WheelHSL)-h0, 360)
		if d > 180 {
			d -= 360
		}
		if d < -180 {
			d += 360
		}
		offs = append(offs, d)
	}
	slices.Sort(offs)

	for i, want := range []float64{-40, -20, 0, 20, 40} {
		if math.Abs(offs[i]-want) > 2 {
			t.Fatalf("analogous offsets = %v, want ~[-40 -20 0 20 40]", offs)
		}
	}
}

func TestHarmonyMonochromatic(t *testing.T) {
	base := baseColour()
	for _, w := range []Wheel{WheelHSL, WheelOKLCh} {
		got := Harmony(base, Monochromatic, 6, HarmonyOptions{Wheel: w})
		h0 := wheelHue(got[0], w)
		tol := 2.0
		if w == WheelOKLCh {
			tol = 5.0
		}
		for i, c := range got {
			if d := angDelta(wheelHue(c, w), h0); d > tol {
				t.Fatalf("wheel %d entry %d hue drifted %.2f", w, i, d)
			}
		}
		for i := 1; i < len(got); i++ {
			if wheelLight(got[i], w) <= wheelLight(got[i-1], w) {
				t.Fatalf("wheel %d: lightness not strictly increasing at %d: %+v", w, i, got)
			}
		}
	}
}

func TestHarmonyDeterministic(t *testing.T) {
	base := baseColour()
	opt := HarmonyOptions{Wheel: WheelOKLCh, SpreadDeg: 25}
	a := Harmony(base, Tetradic, 6, opt)
	b := Harmony(base, Tetradic, 6, opt)
	if !eqPal(a, b) {
		t.Fatalf("non-deterministic: %+v vs %+v", a, b)
	}
}
