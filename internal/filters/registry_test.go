package filters

import (
	"image"
	"testing"
)

// solid builds a w x h image filled with one colour, for effect tests.
func solid(w, h int, r, g, b, a uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = r, g, b, a
	}
	return img
}

func TestApplyUnknownEffect(t *testing.T) {
	if _, err := Apply("does-not-exist", solid(2, 2, 0, 0, 0, 255), nil); err == nil {
		t.Fatal("expected error for unknown effect")
	}
}
