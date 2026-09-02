package gradient

import (
	"math"
	"testing"

	"bitbrush/internal/colorspace"
)

func chanDiff(a, b uint8) int {
	d := int(a) - int(b)
	if d < 0 {
		return -d
	}
	return d
}

var allSpaces = []Space{
	SpaceSRGB, SpaceLinearRGB, SpaceHSL, SpaceLab, SpaceLCh, SpaceOKLab, SpaceOKLCh,
}

func TestNewSortsAndDefaults(t *testing.T) {
	g := New([]Stop{
		{RGB{0, 0, 255}, 1},
		{RGB{255, 0, 0}, 0},
		{RGB{0, 255, 0}, 0.5},
	}, SpaceOKLab, HueShorter, nil)

	if g.Stops[0].Pos != 0 || g.Stops[1].Pos != 0.5 || g.Stops[2].Pos != 1 {
		t.Fatalf("stops not sorted: %v", g.Stops)
	}
	if g.Stops[0].Color != (RGB{255, 0, 0}) {
		t.Fatalf("stop 0 = %v", g.Stops[0].Color)
	}
	if g.Easing == nil {
		t.Fatal("nil easing not defaulted")
	}
	if _, ok := g.Easing.(Linear); !ok {
		t.Fatalf("default easing = %T, want Linear", g.Easing)
	}
}

func TestNewClampsPositionsAndPanicsEmpty(t *testing.T) {
	g := New([]Stop{{RGB{1, 2, 3}, -3}, {RGB{4, 5, 6}, 9}}, SpaceSRGB, HueShorter, nil)
	if g.Stops[0].Pos != 0 || g.Stops[1].Pos != 1 {
		t.Fatalf("positions not clamped: %v", g.Stops)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("New(nil) did not panic")
		}
	}()
	New(nil, SpaceSRGB, HueShorter, nil)
}

func TestAtEndpointsExactEverySpace(t *testing.T) {
	pairs := [][2]RGB{
		{{0, 0, 0}, {255, 255, 255}},
		{{255, 0, 0}, {0, 64, 255}},
		{{10, 200, 30}, {200, 10, 120}},
	}
	for _, sp := range allSpaces {
		for _, p := range pairs {
			g := New([]Stop{{p[0], 0}, {p[1], 1}}, sp, HueShorter, nil)
			if g.At(0) != p[0] {
				t.Fatalf("space %d: At(0) = %v, want %v", sp, g.At(0), p[0])
			}
			if g.At(1) != p[1] {
				t.Fatalf("space %d: At(1) = %v, want %v", sp, g.At(1), p[1])
			}
			// Clamping outside [0,1].
			if g.At(-5) != p[0] || g.At(5) != p[1] {
				t.Fatalf("space %d: At clamp failed", sp)
			}
		}
	}
}

func TestAtSingleStop(t *testing.T) {
	g := New([]Stop{{RGB{7, 8, 9}, 0.4}}, SpaceOKLab, HueShorter, nil)
	for _, x := range []float64{0, 0.4, 1} {
		if g.At(x) != (RGB{7, 8, 9}) {
			t.Fatalf("single-stop At(%v) = %v", x, g.At(x))
		}
	}
}

// relLuma is a rough perceived-brightness proxy for a grey.
func relLuma(c RGB) float64 {
	lr := colorspace.LinearizeChannel(float64(c.R) / 255)
	lg := colorspace.LinearizeChannel(float64(c.G) / 255)
	lb := colorspace.LinearizeChannel(float64(c.B) / 255)
	return 0.2126*lr + 0.7152*lg + 0.0722*lb
}

// TestMidpointGammaEffect pins the well-known ordering of the black->white
// midpoint across spaces: gamma sRGB sits between the (darker) perceptual
// OKLab midpoint and the (lighter) linear-light midpoint.
//
// NB: the task brief describes this the other way round ("OKLab lighter
// than sRGB"); that is the linear-RGB result. OKLab L=0.5 maps to ~12.5%
// luminance, below sRGB 128's ~21.6%, so OKLab's midpoint is darker.
func TestMidpointGammaEffect(t *testing.T) {
	black, white := RGB{0, 0, 0}, RGB{255, 255, 255}
	mk := func(sp Space) RGB {
		return New([]Stop{{black, 0}, {white, 1}}, sp, HueShorter, nil).At(0.5)
	}
	oklab := relLuma(mk(SpaceOKLab))
	srgb := relLuma(mk(SpaceSRGB))
	linear := relLuma(mk(SpaceLinearRGB))

	if !(oklab < srgb) {
		t.Fatalf("expected OKLab midpoint darker than sRGB: oklab=%.4f srgb=%.4f", oklab, srgb)
	}
	if !(srgb < linear) {
		t.Fatalf("expected sRGB midpoint darker than linear-RGB: srgb=%.4f linear=%.4f", srgb, linear)
	}
	if mk(SpaceLinearRGB) != (RGB{188, 188, 188}) {
		t.Fatalf("linear-RGB midpoint = %v, want {188,188,188}", mk(SpaceLinearRGB))
	}
}

func hueOf(c RGB) float64 {
	return colorspace.OKLabToOKLCh(colorspace.RGBToOKLab(c.R, c.G, c.B)).H
}

func circDist(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 360)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func TestHueArcShorterVsLonger(t *testing.T) {
	red, blue := RGB{255, 0, 0}, RGB{0, 0, 255}
	short := New([]Stop{{red, 0}, {blue, 1}}, SpaceOKLCh, HueShorter, nil).At(0.5)
	long := New([]Stop{{red, 0}, {blue, 1}}, SpaceOKLCh, HueLonger, nil).At(0.5)

	if short == long {
		t.Fatal("shorter and longer arcs produced the same midpoint")
	}
	hs, hl := hueOf(short), hueOf(long)
	if d := circDist(hs, hl); d < 120 {
		t.Fatalf("midpoint hues not on opposite sides: shorter=%.1f longer=%.1f dist=%.1f", hs, hl, d)
	}
	// Shorter arc from ~29 deg to ~264 deg wraps through magenta (~327);
	// longer stays on the green side (~147).
	if !(hs > 270 || hs < 60) {
		t.Fatalf("shorter-arc midpoint hue = %.1f, expected near the red/magenta side", hs)
	}
	if !(hl > 90 && hl < 210) {
		t.Fatalf("longer-arc midpoint hue = %.1f, expected near the green side", hl)
	}
}

func TestHueArcIncreasingDecreasing(t *testing.T) {
	// Two hues close together: increasing keeps the short step, decreasing
	// forces the long way round.
	a := RGB{255, 0, 0}   // OKLCh hue ~29
	b := RGB{255, 180, 0} // OKLCh hue ~78 (orange)
	inc := New([]Stop{{a, 0}, {b, 1}}, SpaceOKLCh, HueIncreasing, nil).At(0.5)
	dec := New([]Stop{{a, 0}, {b, 1}}, SpaceOKLCh, HueDecreasing, nil).At(0.5)
	hi, hd := hueOf(inc), hueOf(dec)
	if !(hi > 29 && hi < 80) {
		t.Fatalf("increasing midpoint hue = %.1f, want between the endpoints", hi)
	}
	if circDist(hd, 230) > 90 {
		t.Fatalf("decreasing midpoint hue = %.1f, want the long way round (~230)", hd)
	}
}

func TestAchromaticEndpointCarriesHue(t *testing.T) {
	// White (no chroma) -> saturated blue in HSL: the midpoint must not
	// swing through hue 0 (reds); it should stay blue-ish.
	g := New([]Stop{{RGB{255, 255, 255}, 0}, {RGB{0, 0, 255}, 1}}, SpaceHSL, HueShorter, nil)
	mid := g.At(0.5)
	if mid.B <= mid.R || mid.B <= mid.G {
		t.Fatalf("achromatic->blue midpoint = %v, expected blue dominant", mid)
	}
}

func TestSampleAndSampleStops(t *testing.T) {
	g := New([]Stop{
		{RGB{0, 0, 0}, 0}, {RGB{255, 0, 0}, 0.5}, {RGB{255, 255, 255}, 1},
	}, SpaceOKLab, HueShorter, nil)

	for _, n := range []int{2, 3, 8, 33} {
		s := g.Sample(n)
		if len(s) != n {
			t.Fatalf("Sample(%d) len = %d", n, len(s))
		}
		if s[0] != g.At(0) || s[n-1] != g.At(1) {
			t.Fatalf("Sample(%d) endpoints wrong", n)
		}
		ss := g.SampleStops(n)
		if len(ss) != n {
			t.Fatalf("SampleStops(%d) len = %d", n, len(ss))
		}
		for i, st := range ss {
			want := float64(i) / float64(n-1)
			if st.Pos != want {
				t.Fatalf("SampleStops(%d)[%d].Pos = %v, want %v", n, i, st.Pos, want)
			}
			if st.Color != g.At(want) {
				t.Fatalf("SampleStops(%d)[%d].Color mismatch", n, i)
			}
		}
	}
}

func TestEasingAppliedBetweenStops(t *testing.T) {
	// A steps easing between two stops must quantise the output to the
	// stop colours only.
	g := New([]Stop{{RGB{0, 0, 0}, 0}, {RGB{255, 255, 255}, 1}},
		SpaceSRGB, HueShorter, Steps{N: 2, Jump: JumpEnd})
	// jump-end, 2 steps: [0,0.5) -> black, [0.5,1) -> mid, 1 -> white.
	if g.At(0.25) != (RGB{0, 0, 0}) {
		t.Fatalf("At(0.25) = %v, want black", g.At(0.25))
	}
	if g.At(0.75) != (RGB{128, 128, 128}) {
		t.Fatalf("At(0.75) = %v, want mid grey", g.At(0.75))
	}
}
