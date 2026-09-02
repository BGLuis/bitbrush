package colorspace

import (
	"math"
	"testing"
)

// grid yields a spread of 8-bit sRGB triples for round-trip testing.
func grid(t *testing.T, fn func(r, g, b uint8)) {
	t.Helper()
	steps := []uint8{0, 1, 37, 64, 128, 129, 200, 254, 255}
	for _, r := range steps {
		for _, g := range steps {
			for _, b := range steps {
				fn(r, g, b)
			}
		}
	}
}

func absDiff(a, b uint8) int {
	d := int(a) - int(b)
	if d < 0 {
		return -d
	}
	return d
}

func TestOKLabRoundTrip(t *testing.T) {
	grid(t, func(r, g, b uint8) {
		gr, gg, gb := OKLabToRGB(RGBToOKLab(r, g, b))
		if absDiff(gr, r) > 1 || absDiff(gg, g) > 1 || absDiff(gb, b) > 1 {
			t.Fatalf("OKLab round trip (%d,%d,%d) -> (%d,%d,%d)", r, g, b, gr, gg, gb)
		}
	})
}

func TestOKLChRoundTrip(t *testing.T) {
	grid(t, func(r, g, b uint8) {
		ch := OKLabToOKLCh(RGBToOKLab(r, g, b))
		gr, gg, gb := OKLabToRGB(OKLChToOKLab(ch))
		if absDiff(gr, r) > 1 || absDiff(gg, g) > 1 || absDiff(gb, b) > 1 {
			t.Fatalf("OKLCh round trip (%d,%d,%d) -> (%d,%d,%d)", r, g, b, gr, gg, gb)
		}
		if ch.H < 0 || ch.H >= 360 {
			t.Fatalf("OKLCh hue out of range for (%d,%d,%d): %v", r, g, b, ch.H)
		}
		if ch.C < 0 {
			t.Fatalf("OKLCh negative chroma for (%d,%d,%d): %v", r, g, b, ch.C)
		}
	})
}

func TestLabRoundTrip(t *testing.T) {
	grid(t, func(r, g, b uint8) {
		gr, gg, gb := LabToRGB(RGBToLab(r, g, b))
		if absDiff(gr, r) > 1 || absDiff(gg, g) > 1 || absDiff(gb, b) > 1 {
			t.Fatalf("Lab round trip (%d,%d,%d) -> (%d,%d,%d)", r, g, b, gr, gg, gb)
		}
	})
}

func TestLabLChRoundTrip(t *testing.T) {
	grid(t, func(r, g, b uint8) {
		lab := RGBToLab(r, g, b)
		back := LChToLab(LabToLCh(lab))
		if math.Abs(back.L-lab.L) > 1e-9 ||
			math.Abs(back.A-lab.A) > 1e-9 ||
			math.Abs(back.B-lab.B) > 1e-9 {
			t.Fatalf("Lab<->LCh mismatch for (%d,%d,%d): %+v vs %+v", r, g, b, lab, back)
		}
		ch := LabToLCh(lab)
		if ch.H < 0 || ch.H >= 360 {
			t.Fatalf("LCh hue out of range for (%d,%d,%d): %v", r, g, b, ch.H)
		}
	})
}

func TestHSLRoundTrip(t *testing.T) {
	grid(t, func(r, g, b uint8) {
		hsl := RGBToHSL(r, g, b)
		if hsl.H < 0 || hsl.H >= 360 {
			t.Fatalf("HSL hue out of range for (%d,%d,%d): %v", r, g, b, hsl.H)
		}
		gr, gg, gb := HSLToRGB(hsl)
		if absDiff(gr, r) > 1 || absDiff(gg, g) > 1 || absDiff(gb, b) > 1 {
			t.Fatalf("HSL round trip (%d,%d,%d) -> (%d,%d,%d)", r, g, b, gr, gg, gb)
		}
	})
}

func TestAnchors(t *testing.T) {
	if l := RGBToLab(255, 255, 255); math.Abs(l.L-100) > 1e-3 || math.Abs(l.A) > 1e-3 || math.Abs(l.B) > 1e-3 {
		t.Fatalf("white Lab = %+v, want L~100 a~0 b~0", l)
	}
	if l := RGBToLab(0, 0, 0); math.Abs(l.L) > 1e-6 {
		t.Fatalf("black Lab.L = %v, want ~0", l.L)
	}
	if c := OKLabToOKLCh(RGBToOKLab(255, 255, 255)); math.Abs(c.L-1) > 1e-3 {
		t.Fatalf("white OKLCh.L = %v, want ~1", c.L)
	}
	if c := OKLabToOKLCh(RGBToOKLab(0, 0, 0)); math.Abs(c.L) > 1e-6 {
		t.Fatalf("black OKLCh.L = %v, want ~0", c.L)
	}

	for _, v := range []uint8{16, 64, 128, 200, 240} {
		if h := RGBToHSL(v, v, v); math.Abs(h.S) > 1e-9 {
			t.Fatalf("grey %d HSL.S = %v, want 0", v, h.S)
		}
		if c := OKLabToOKLCh(RGBToOKLab(v, v, v)); c.C > 1e-4 {
			t.Fatalf("grey %d OKLCh.C = %v, want ~0", v, c.C)
		}
		if c := OKLabToOKLCh(RGBToOKLab(v, v, v)); c.H != 0 {
			t.Fatalf("grey %d OKLCh.H = %v, want 0 (undefined)", v, c.H)
		}
	}
}

func TestOKLabToLinearRGBOutOfGamut(t *testing.T) {
	// A highly saturated OKLCh that sRGB cannot represent: some linear
	// channel must land outside [0,1].
	lin := struct{ r, g, b float64 }{}
	lin.r, lin.g, lin.b = OKLabToLinearRGB(OKLChToOKLab(OKLCh{L: 0.7, C: 0.4, H: 30}))
	if InGamutLinear(lin.r, lin.g, lin.b) {
		t.Fatalf("expected out-of-gamut, got (%v,%v,%v)", lin.r, lin.g, lin.b)
	}
}
