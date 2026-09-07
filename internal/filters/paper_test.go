package filters

import (
	"bytes"
	"testing"
)

func TestPaperKeepsShapeAndAlpha(t *testing.T) {
	src := gradientImg(24, 18)
	for i := 3; i < len(src.Pix); i += 4 {
		src.Pix[i] = byte(i % 251)
	}
	out, err := Apply("paper", src, Params{"seed": 3.0})
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != src.Bounds() {
		t.Fatalf("bounds %v, want %v", out.Bounds(), src.Bounds())
	}
	for i := 3; i < len(out.Pix); i += 4 {
		if out.Pix[i] != src.Pix[i] {
			t.Fatalf("pixel %d alpha = %d, want %d", i/4, out.Pix[i], src.Pix[i])
		}
	}
}

func TestPaperDoesNotMutateSource(t *testing.T) {
	src := gradientImg(20, 13)
	before := append([]byte(nil), src.Pix...)
	if _, err := Apply("paper", src, Params{"seed": 1.0}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src.Pix, before) {
		t.Fatal("paper mutated its source image")
	}
}

func TestPaperDeterministic(t *testing.T) {
	assertDeterministic(t, "paper", gradientImg(20, 13), Params{"seed": 7.0})
}

func TestPaperSeedChangesTexture(t *testing.T) {
	src := solid(48, 48, 255, 255, 255, 255)
	a, err := Apply("paper", src, Params{"seed": 1.0, "grain": 1.0, "mottle": 1.0})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Apply("paper", src, Params{"seed": 2.0, "grain": 1.0, "mottle": 1.0})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("different seeds produced identical paper texture")
	}
}

// With every texture, ageing and vignette switched off, a white input must
// come out as the flat paper colour.
func TestPaperWhiteResolvesToPaperColour(t *testing.T) {
	src := solid(16, 16, 255, 255, 255, 255)
	out, err := Apply("paper", src, Params{
		"age": 0.0, "grain": 0.0, "mottle": 0.0, "fibers": 0.0, "vignette": 0.0,
		"paper": "#f3ecd8",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := out.Pix[out.PixOffset(8, 8):][:3]
	want := []byte{0xf3, 0xec, 0xd8}
	for c := 0; c < 3; c++ {
		if diff := int(got[c]) - int(want[c]); diff < -1 || diff > 1 {
			t.Fatalf("centre pixel = %v, want ~%v", got, want)
		}
	}
}

func BenchmarkPaper1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"seed": 1.0}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("paper", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
