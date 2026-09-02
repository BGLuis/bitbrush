package filters

import (
	"bytes"
	"image"
	"testing"
)

// assertDeterministic runs name twice with the same input and params and
// fails if the output bytes differ. Every effect must pass this.
func assertDeterministic(t *testing.T, name string, src *image.RGBA, params Params) {
	t.Helper()
	a, err := Apply(name, src, params)
	if err != nil {
		t.Fatalf("%s: first run: %v", name, err)
	}
	b, err := Apply(name, src, params)
	if err != nil {
		t.Fatalf("%s: second run: %v", name, err)
	}
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatalf("%s: output is not deterministic", name)
	}
}

// gradientImg builds a w x h opaque image with a diagonal colour ramp.
func gradientImg(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := img.PixOffset(x, y)
			img.Pix[i] = uint8((x * 255) / max(w-1, 1))
			img.Pix[i+1] = uint8((y * 255) / max(h-1, 1))
			img.Pix[i+2] = uint8(((x + y) * 255) / max(w+h-2, 1))
			img.Pix[i+3] = 255
		}
	}
	return img
}

func TestPixelateSolidUnchanged(t *testing.T) {
	src := solid(16, 16, 40, 80, 120, 255)
	out, err := Apply("pixelate", src, Params{"blockSize": 4.0})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Pix, src.Pix) {
		t.Fatal("a solid image must be unchanged by pixelate")
	}
}

func TestPixelateBlockIsAverage(t *testing.T) {
	// 2x2: one opaque-white pixel, three opaque-black. blockSize 2 -> every
	// output pixel is the block mean: R=G=B=255/4=63, A=255.
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.Pix[0], src.Pix[1], src.Pix[2], src.Pix[3] = 255, 255, 255, 255
	for i := 4; i < 16; i += 4 {
		src.Pix[i+3] = 255
	}
	out, err := Apply("pixelate", src, Params{"blockSize": 2.0})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i += 4 {
		got := [4]byte{out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3]}
		if got != [4]byte{63, 63, 63, 255} {
			t.Fatalf("pixel %d = %v, want [63 63 63 255]", i/4, got)
		}
	}
}

func TestPixelateNonDivisibleDimensions(t *testing.T) {
	// 10x7 with blockSize 4 -> edge blocks of 2 across and 3 down. Must not
	// panic and must write every pixel (alpha 255 everywhere).
	src := solid(10, 7, 10, 20, 30, 255)
	out, err := Apply("pixelate", src, Params{"blockSize": 4.0})
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != image.Rect(0, 0, 10, 7) {
		t.Fatalf("bounds changed: %v", out.Bounds())
	}
	for i := 3; i < len(out.Pix); i += 4 {
		if out.Pix[i] != 255 {
			t.Fatalf("pixel %d was not written (alpha %d)", i/4, out.Pix[i])
		}
	}
}

func TestPixelateBlockSizeOneIsIdentity(t *testing.T) {
	src := gradientImg(8, 8)
	out, err := Apply("pixelate", src, Params{"blockSize": 1.0})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Pix, src.Pix) {
		t.Fatal("blockSize 1 must be an identity transform")
	}
}

func TestPixelateMissingParamUsesDefault(t *testing.T) {
	src := gradientImg(24, 24)
	withDefault, err := Apply("pixelate", src, Params{})
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := Apply("pixelate", src, Params{"blockSize": 8.0})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(withDefault.Pix, explicit.Pix) {
		t.Fatal("omitting blockSize must behave like blockSize=8")
	}
}

func TestPixelateDoesNotMutateSource(t *testing.T) {
	src := gradientImg(20, 13)
	before := append([]byte(nil), src.Pix...)
	if _, err := Apply("pixelate", src, Params{"blockSize": 5.0}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src.Pix, before) {
		t.Fatal("pixelate mutated its source image")
	}
}

func TestPixelateDeterministic(t *testing.T) {
	assertDeterministic(t, "pixelate", gradientImg(20, 13), Params{"blockSize": 5.0})
}

func BenchmarkPixelate1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"blockSize": 8.0}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("pixelate", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
