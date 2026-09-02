package filters

import (
	"image"
	"testing"
)

// solidRGBA returns a w*h image filled with (r,g,b,a).
func solidRGBA(w, h int, r, g, b, a uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = r, g, b, a
	}
	return img
}

// hGradient returns a horizontal black->white gradient.
func hGradient(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(x * 255 / max(w-1, 1))
			o := img.PixOffset(x, y)
			img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = v, v, v, 255
		}
	}
	return img
}

func inkFraction(img *image.RGBA) float64 {
	var dark int
	n := len(img.Pix) / 4
	for i := 0; i < len(img.Pix); i += 4 {
		if luma(img.Pix[i], img.Pix[i+1], img.Pix[i+2]) < 128 {
			dark++
		}
	}
	return float64(dark) / float64(n)
}

func TestHalftoneSizeAndAlpha(t *testing.T) {
	src := hGradient(48, 32)
	src.Pix[3] = 123 // a stray alpha value must survive
	out, err := Halftone(src, Params{"cellSize": 6.0})
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != src.Bounds() {
		t.Fatalf("bounds %v != %v", out.Bounds(), src.Bounds())
	}
	if out.Pix[3] != 123 {
		t.Fatalf("alpha not copied through: got %d", out.Pix[3])
	}
}

func TestHalftoneToneExtremes(t *testing.T) {
	black := solidRGBA(60, 60, 0, 0, 0, 255)
	white := solidRGBA(60, 60, 255, 255, 255, 255)

	bo, _ := Halftone(black, Params{"cellSize": 6.0})
	wo, _ := Halftone(white, Params{"cellSize": 6.0})

	if f := inkFraction(bo); f < 0.9 {
		t.Fatalf("black input should be almost fully inked, got %.2f", f)
	}
	if f := inkFraction(wo); f > 0.1 {
		t.Fatalf("white input should be almost fully paper, got %.2f", f)
	}
}

func TestHalftoneInvert(t *testing.T) {
	src := solidRGBA(60, 60, 40, 40, 40, 255) // fairly dark
	plain, _ := Halftone(src, Params{"cellSize": 6.0})
	inv, _ := Halftone(src, Params{"cellSize": 6.0, "invert": true})
	if inkFraction(inv) >= inkFraction(plain) {
		t.Fatalf("invert should reduce ink on a dark input: plain=%.2f inv=%.2f",
			inkFraction(plain), inkFraction(inv))
	}
}

func TestHalftoneDeterministic(t *testing.T) {
	src := hGradient(50, 40)
	p := Params{"cellSize": 5.0, "angleDeg": 30.0, "shape": "diamond", "gamma": 1.3}
	a, _ := Halftone(src, p)
	b, _ := Halftone(src, p)
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			t.Fatalf("non-deterministic at byte %d: %d != %d", i, a.Pix[i], b.Pix[i])
		}
	}
}

func TestHalftoneColorChannels(t *testing.T) {
	src := hGradient(64, 24)
	for _, ch := range []string{"cmyk", "rgb"} {
		out, err := Halftone(src, Params{"cellSize": 6.0, "channels": ch})
		if err != nil {
			t.Fatalf("%s: %v", ch, err)
		}
		if out.Bounds() != src.Bounds() {
			t.Fatalf("%s: bad bounds", ch)
		}
	}
}

func TestHalftoneShapes(t *testing.T) {
	src := solidRGBA(40, 40, 128, 128, 128, 255)
	for _, s := range []string{"circle", "square", "diamond", "line"} {
		if _, err := Halftone(src, Params{"cellSize": 8.0, "shape": s}); err != nil {
			t.Fatalf("shape %s: %v", s, err)
		}
	}
}
