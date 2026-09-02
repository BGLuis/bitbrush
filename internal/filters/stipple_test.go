package filters

import (
	"bytes"
	"image"
	"testing"
)

func nonPaperCount(img *image.RGBA, paperR, paperG, paperB uint8) int {
	var n int
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] != paperR || img.Pix[i+1] != paperG || img.Pix[i+2] != paperB {
			n++
		}
	}
	return n
}

func TestStippleSizeAndAlpha(t *testing.T) {
	src := hGradient(40, 30)
	src.Pix[7] = 99
	out, err := Stipple(src, Params{"points": 500.0, "iterations": 4.0})
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != src.Bounds() {
		t.Fatalf("bounds %v != %v", out.Bounds(), src.Bounds())
	}
	if out.Pix[7] != 99 {
		t.Fatalf("alpha not copied through: got %d", out.Pix[7])
	}
}

func TestStippleDeterministic(t *testing.T) {
	src := hGradient(48, 36)
	p := Params{"points": 800.0, "iterations": 6.0, "seed": 42.0}
	a, _ := Stipple(src, p)
	b, _ := Stipple(src, p)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("same seed+params produced different pixels")
	}
	c, _ := Stipple(src, Params{"points": 800.0, "iterations": 6.0, "seed": 43.0})
	if bytes.Equal(a.Pix, c.Pix) {
		t.Fatal("different seed produced identical pixels")
	}
}

func TestStippleMorePointsMoreInk(t *testing.T) {
	src := solidRGBA(64, 64, 0, 0, 0, 255) // black -> uniformly dense
	few, _ := Stipple(src, Params{"points": 300.0, "iterations": 8.0, "paper": "#ffffff", "ink": "#000000"})
	many, _ := Stipple(src, Params{"points": 3000.0, "iterations": 8.0, "paper": "#ffffff", "ink": "#000000"})
	fn := nonPaperCount(few, 255, 255, 255)
	mn := nonPaperCount(many, 255, 255, 255)
	if mn <= fn {
		t.Fatalf("more points should ink more pixels: few=%d many=%d", fn, mn)
	}
}

func TestStippleDarkerInputInksMore(t *testing.T) {
	black := solidRGBA(64, 64, 0, 0, 0, 255)
	white := solidRGBA(64, 64, 255, 255, 255, 255)
	p := Params{"points": 1500.0, "iterations": 10.0, "paper": "#ffffff", "ink": "#000000"}
	bo, _ := Stipple(black, p)
	wo, _ := Stipple(white, p)
	if nonPaperCount(bo, 255, 255, 255) <= nonPaperCount(wo, 255, 255, 255) {
		t.Fatalf("black input should ink more than white: black=%d white=%d",
			nonPaperCount(bo, 255, 255, 255), nonPaperCount(wo, 255, 255, 255))
	}
}

func TestStippleZeroIterationsStillRenders(t *testing.T) {
	src := hGradient(40, 40)
	out, err := Stipple(src, Params{"points": 600.0, "iterations": 0.0})
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != src.Bounds() {
		t.Fatal("bad bounds at iterations=0")
	}
	if nonPaperCount(out, 245, 242, 234) == 0 {
		t.Fatal("iterations=0 should still draw the initial point set")
	}
}

func TestStippleBadHexFallsBack(t *testing.T) {
	src := hGradient(32, 32)
	if _, err := Stipple(src, Params{"points": 300.0, "iterations": 3.0, "ink": "not-a-colour", "paper": "#zzz"}); err != nil {
		t.Fatalf("invalid hex should fall back to defaults, got %v", err)
	}
}

func TestStippleDoesNotMutateSource(t *testing.T) {
	src := hGradient(40, 28)
	before := append([]byte(nil), src.Pix...)
	if _, err := Stipple(src, Params{"points": 500.0, "iterations": 5.0}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src.Pix, before) {
		t.Fatal("Stipple mutated its source image")
	}
}
