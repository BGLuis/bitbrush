package filters

import (
	"bytes"
	"image"
	"testing"
)

func TestGrayscaleProducesNeutralPixels(t *testing.T) {
	out, err := Apply("grayscale", gradientImg(32, 24), Params{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(out.Pix); i += 4 {
		if out.Pix[i] != out.Pix[i+1] || out.Pix[i+1] != out.Pix[i+2] {
			t.Fatalf("pixel %d = %v, want R==G==B", i/4, out.Pix[i:i+3])
		}
	}
}

func TestGrayscaleMethodsStayNeutral(t *testing.T) {
	for _, m := range []string{"luma", "luminance", "average", "bt601", "lightness"} {
		out, err := Apply("grayscale", gradientImg(16, 16), Params{"method": m})
		if err != nil {
			t.Fatalf("method %q: %v", m, err)
		}
		for i := 0; i < len(out.Pix); i += 4 {
			if out.Pix[i] != out.Pix[i+1] || out.Pix[i+1] != out.Pix[i+2] {
				t.Fatalf("method %q pixel %d not neutral: %v", m, i/4, out.Pix[i:i+3])
			}
		}
	}
}

func TestGrayscaleThresholdBinarises(t *testing.T) {
	out, err := Apply("grayscale", gradientImg(40, 40), Params{"threshold": 128.0})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(out.Pix); i += 4 {
		if v := out.Pix[i]; v != 0 && v != 255 {
			t.Fatalf("pixel %d = %d, want 0 or 255 with a threshold set", i/4, v)
		}
	}
}

func TestGrayscaleInvert(t *testing.T) {
	out, err := Apply("grayscale", solid(8, 8, 0, 0, 0, 255), Params{"invert": true})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(out.Pix); i += 4 {
		if out.Pix[i] != 255 {
			t.Fatalf("pixel %d = %d, want 255 (inverted black)", i/4, out.Pix[i])
		}
	}
}

func TestGrayscaleAlphaPreserved(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2] = 200, 40, 90
		src.Pix[i+3] = byte(i)
	}
	out, err := Apply("grayscale", src, Params{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 3; i < len(out.Pix); i += 4 {
		if out.Pix[i] != src.Pix[i] {
			t.Fatalf("pixel %d alpha = %d, want %d", i/4, out.Pix[i], src.Pix[i])
		}
	}
}

func TestGrayscaleDoesNotMutateSource(t *testing.T) {
	src := gradientImg(20, 13)
	before := append([]byte(nil), src.Pix...)
	if _, err := Apply("grayscale", src, Params{"contrast": 30.0, "brightness": -10.0}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src.Pix, before) {
		t.Fatal("grayscale mutated its source image")
	}
}

func TestGrayscaleDeterministic(t *testing.T) {
	assertDeterministic(t, "grayscale", gradientImg(20, 13), Params{"method": "luminance", "contrast": 20.0})
}

func BenchmarkGrayscale1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"method": "luminance"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("grayscale", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
