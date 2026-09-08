package filters

import (
	"bytes"
	"image"
	"testing"
)

// Shared assertions for the ascii-magic-parity filters (blocks, glyphscreen,
// mosaic, pixelart, lego, voxel and the post-processing set). Each filter's
// own _test.go keeps the effect-specific checks.

// assertNoMutate fails if Apply(name) changes the source pixels.
func assertNoMutate(t *testing.T, name string, src *image.RGBA, params Params) {
	t.Helper()
	before := append([]byte(nil), src.Pix...)
	if _, err := Apply(name, src, params); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if !bytes.Equal(src.Pix, before) {
		t.Fatalf("%s mutated its source image", name)
	}
}

// assertBoundsFull fails unless the output keeps src's dimensions and every
// pixel was written opaque (alpha 255) — the source images used here are all
// fully opaque.
func assertBoundsFull(t *testing.T, name string, src *image.RGBA, params Params) {
	t.Helper()
	out, err := Apply(name, src, params)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if out.Bounds() != image.Rect(0, 0, src.Bounds().Dx(), src.Bounds().Dy()) {
		t.Fatalf("%s: bounds changed to %v", name, out.Bounds())
	}
	for i := 3; i < len(out.Pix); i += 4 {
		if out.Pix[i] != 255 {
			t.Fatalf("%s: pixel %d not written (alpha %d)", name, i/4, out.Pix[i])
		}
	}
}

// assertDefaultMatches fails if omitting params differs from passing explicit.
func assertDefaultMatches(t *testing.T, name string, src *image.RGBA, explicit Params) {
	t.Helper()
	a, err := Apply(name, src, Params{})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	b, err := Apply(name, src, explicit)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatalf("%s: empty params differ from explicit defaults", name)
	}
}
