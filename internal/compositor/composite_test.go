package compositor

import (
	"bytes"
	"image"
	"testing"
)

func solid(w, h int, r, g, b, a uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = r, g, b, a
	}
	return img
}

func TestCompositeOpacityZeroKeepsBase(t *testing.T) {
	base := solid(4, 4, 200, 100, 50, 255)
	over := solid(4, 4, 0, 0, 0, 255)
	out, err := Composite(base, over, BlendNormal, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Pix, base.Pix) {
		t.Fatalf("opacity 0: out %v != base %v", out.Pix[:8], base.Pix[:8])
	}
}

func TestCompositeOpaqueNormalReplacesBase(t *testing.T) {
	base := solid(3, 3, 10, 20, 30, 255)
	over := solid(3, 3, 200, 150, 100, 255)
	out, err := Composite(base, over, BlendNormal, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Pix, over.Pix) {
		t.Fatalf("opaque normal: out %v != over %v", out.Pix[:8], over.Pix[:8])
	}
}

func TestCompositeOverTransparentKeepsBase(t *testing.T) {
	base := solid(2, 2, 123, 45, 67, 255)
	over := solid(2, 2, 255, 255, 255, 0)
	out, err := Composite(base, over, BlendMultiply, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Pix, base.Pix) {
		t.Fatalf("transparent over: out %v != base %v", out.Pix[:8], base.Pix[:8])
	}
}

func TestCompositeHalfOpacityLerps(t *testing.T) {
	base := solid(2, 2, 0, 0, 0, 255)
	over := solid(2, 2, 255, 255, 255, 255)
	out, err := Composite(base, over, BlendNormal, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(out.Pix); i += 4 {
		for c := 0; c < 3; c++ {
			if v := out.Pix[i+c]; v < 127 || v > 129 {
				t.Fatalf("pixel %d ch %d = %d, want ~128", i/4, c, v)
			}
		}
		if out.Pix[i+3] != 255 {
			t.Fatalf("pixel %d alpha = %d, want 255", i/4, out.Pix[i+3])
		}
	}
}

func TestCompositeSemiOverOpaque(t *testing.T) {
	// base opaque grey 100; over grey 200 at alpha 128 (~0.5019), normal.
	// Co = as*Cs + (1-as)*Cb with as ~= 0.5019 -> ~150.
	base := solid(1, 1, 100, 100, 100, 255)
	over := solid(1, 1, 200, 200, 200, 128)
	out, err := Composite(base, over, BlendNormal, 1)
	if err != nil {
		t.Fatal(err)
	}
	if v := out.Pix[0]; v < 149 || v > 151 {
		t.Fatalf("channel = %d, want ~150", v)
	}
	if out.Pix[3] != 255 {
		t.Fatalf("alpha = %d, want 255 (base opaque)", out.Pix[3])
	}
}

func TestCompositeSemiOverSemiAlpha(t *testing.T) {
	// alpha_b = alpha_s ~= 0.5019 -> ao = as + ab(1-as) ~= 0.7519 -> ~192.
	base := solid(1, 1, 0, 0, 0, 128)
	over := solid(1, 1, 0, 0, 0, 128)
	out, err := Composite(base, over, BlendNormal, 1)
	if err != nil {
		t.Fatal(err)
	}
	if v := out.Pix[3]; v < 191 || v > 193 {
		t.Fatalf("composited alpha = %d, want ~192", v)
	}
}

func TestCompositeSizeMismatch(t *testing.T) {
	base := solid(4, 4, 0, 0, 0, 255)
	over := solid(4, 3, 0, 0, 0, 255)
	if _, err := Composite(base, over, BlendNormal, 1); err == nil {
		t.Fatal("expected an error for mismatched sizes")
	}
}

func TestCompositeDoesNotMutateInputs(t *testing.T) {
	base := solid(2, 2, 10, 20, 30, 255)
	over := solid(2, 2, 40, 50, 60, 200)
	baseCopy := append([]byte(nil), base.Pix...)
	overCopy := append([]byte(nil), over.Pix...)
	if _, err := Composite(base, over, BlendScreen, 0.7); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(base.Pix, baseCopy) || !bytes.Equal(over.Pix, overCopy) {
		t.Fatal("Composite mutated an input image")
	}
}
