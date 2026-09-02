package colorspace

import (
	"math"
	"testing"
)

func TestLinearizeEncodeRoundTrip(t *testing.T) {
	for _, x := range []float64{0, 0.01, 0.04045, 0.2, 0.5, 0.8, 1} {
		got := EncodeChannel(LinearizeChannel(x))
		// The sRGB piecewise curve is not perfectly continuous at its
		// 0.04045 knee; a few 1e-8 of slack covers it.
		if math.Abs(got-x) > 1e-6 {
			t.Fatalf("round trip of %v = %v", x, got)
		}
	}
}

func TestRGBToOKLabAnchors(t *testing.T) {
	black := RGBToOKLab(0, 0, 0)
	if math.Abs(black.L) > 1e-6 || math.Abs(black.A) > 1e-6 || math.Abs(black.B) > 1e-6 {
		t.Fatalf("black -> %+v, want ~zero", black)
	}
	white := RGBToOKLab(255, 255, 255)
	if math.Abs(white.L-1) > 1e-3 || math.Abs(white.A) > 1e-3 || math.Abs(white.B) > 1e-3 {
		t.Fatalf("white -> %+v, want L~1, A~0, B~0", white)
	}
}

func TestOKLabGrayIsAchromatic(t *testing.T) {
	for _, v := range []uint8{16, 64, 128, 200, 240} {
		lab := RGBToOKLab(v, v, v)
		if math.Abs(lab.A) > 1e-3 || math.Abs(lab.B) > 1e-3 {
			t.Fatalf("gray %d -> A=%v B=%v, want achromatic", v, lab.A, lab.B)
		}
	}
}

func TestOKLabLightnessMonotonic(t *testing.T) {
	prev := -1.0
	for v := 0; v <= 255; v += 15 {
		l := RGBToOKLab(uint8(v), uint8(v), uint8(v)).L
		if l < prev {
			t.Fatalf("L not monotonic at gray %d: %v < %v", v, l, prev)
		}
		prev = l
	}
}
