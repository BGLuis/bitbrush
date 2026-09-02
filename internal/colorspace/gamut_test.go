package colorspace

import (
	"math"
	"testing"
)

func TestInGamutLinear(t *testing.T) {
	if !InGamutLinear(0, 0.5, 1) {
		t.Fatal("mid cube reported out of gamut")
	}
	if !InGamutLinear(-5e-5, 1+5e-5, 0.2) {
		t.Fatal("within epsilon reported out of gamut")
	}
	if InGamutLinear(0, 0, 1.01) {
		t.Fatal("clearly out-of-range value reported in gamut")
	}
	if InGamutLinear(-0.2, 0.5, 0.5) {
		t.Fatal("negative channel reported in gamut")
	}
}

func TestClampToGamutReducesChroma(t *testing.T) {
	in := OKLCh{L: 0.7, C: 0.4, H: 30}
	out := ClampToGamut(in)

	if out.L != in.L || out.H != in.H {
		t.Fatalf("ClampToGamut moved L or H: %+v -> %+v", in, out)
	}
	if !(out.C < in.C) {
		t.Fatalf("ClampToGamut did not reduce chroma: %v -> %v", in.C, out.C)
	}
	r, g, b := OKLabToLinearRGB(OKLChToOKLab(out))
	if !InGamutLinear(r, g, b) {
		t.Fatalf("ClampToGamut result still out of gamut: (%v,%v,%v)", r, g, b)
	}
}

func TestClampToGamutKeepsInGamutColour(t *testing.T) {
	// A muted colour well inside sRGB.
	in := OKLabToOKLCh(RGBToOKLab(120, 90, 60))
	out := ClampToGamut(in)
	if math.Abs(out.C-in.C) > 1e-9 || math.Abs(out.L-in.L) > 1e-9 || math.Abs(out.H-in.H) > 1e-9 {
		t.Fatalf("in-gamut colour changed: %+v -> %+v", in, out)
	}
}

func TestClampToGamutClampsLightness(t *testing.T) {
	if out := ClampToGamut(OKLCh{L: 1.5, C: 0.3, H: 200}); out.L != 1 || out.C != 0 {
		t.Fatalf("L>1 clamp: got %+v, want L=1 C=0", out)
	}
	if out := ClampToGamut(OKLCh{L: -0.2, C: 0.3, H: 200}); out.L != 0 || out.C != 0 {
		t.Fatalf("L<0 clamp: got %+v, want L=0 C=0", out)
	}
}
