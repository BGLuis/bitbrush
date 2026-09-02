package filters

import (
	"bytes"
	"image"
	"testing"
)

// distinctColors counts unique RGBA tuples in img.
func distinctColors(img *image.RGBA) int {
	seen := make(map[[4]byte]struct{})
	for i := 0; i+3 < len(img.Pix); i += 4 {
		seen[[4]byte{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}] = struct{}{}
	}
	return len(seen)
}

// bands builds a w x h opaque image of vertical colour bands.
func bands(w, h int, cs ...[3]byte) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	band := w / len(cs)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cs[min(x/band, len(cs)-1)]
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c[0], c[1], c[2], 255
		}
	}
	return img
}

func TestQuantizeReducesColourCount(t *testing.T) {
	out, err := Apply("quantize", gradientImg(64, 64), Params{"colors": 4.0})
	if err != nil {
		t.Fatal(err)
	}
	if n := distinctColors(out); n > 4 {
		t.Fatalf("output has %d colours, want <= 4", n)
	}
}

func TestQuantizeIdentityWhenPaletteLargeEnough(t *testing.T) {
	src := bands(12, 6, [3]byte{20, 20, 200}, [3]byte{200, 20, 20}, [3]byte{20, 200, 20})
	for _, space := range []string{"oklab", "srgb"} {
		out, err := Apply("quantize", src, Params{"colors": 16.0, "space": space})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out.Pix, src.Pix) {
			t.Fatalf("space %s: 3 well-separated colours with colors=16 should be unchanged", space)
		}
	}
}

func TestQuantizeAlphaPreserved(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2] = byte(i), byte(255-i), 90
		src.Pix[i+3] = byte((i / 4 * 8) % 256)
	}
	out, err := Apply("quantize", src, Params{"colors": 6.0})
	if err != nil {
		t.Fatal(err)
	}
	for i := 3; i < len(out.Pix); i += 4 {
		if out.Pix[i] != src.Pix[i] {
			t.Fatalf("pixel %d alpha = %d, want %d", i/4, out.Pix[i], src.Pix[i])
		}
	}
}

func TestQuantizeClampsColours(t *testing.T) {
	src := gradientImg(32, 32)
	// colors below 2 clamps to 2.
	out, err := Apply("quantize", src, Params{"colors": 1.0})
	if err != nil {
		t.Fatal(err)
	}
	if n := distinctColors(out); n > 2 {
		t.Fatalf("colors=1 clamped to 2, but output has %d colours", n)
	}
}

func TestQuantizeDoesNotMutateSource(t *testing.T) {
	src := gradientImg(24, 18)
	before := append([]byte(nil), src.Pix...)
	if _, err := Apply("quantize", src, Params{"colors": 8.0}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src.Pix, before) {
		t.Fatal("quantize mutated its source image")
	}
}

func TestQuantizeDeterministic(t *testing.T) {
	assertDeterministic(t, "quantize", gradientImg(40, 27), Params{"colors": 12.0})
	assertDeterministic(t, "quantize", gradientImg(40, 27), Params{"colors": 12.0, "space": "srgb"})
}

func BenchmarkQuantize1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"colors": 16.0}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("quantize", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
