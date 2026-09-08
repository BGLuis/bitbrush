package filters

import (
	"bytes"
	"testing"
)

func TestPixelArtBits8IsPixelate(t *testing.T) {
	// bits=8, no palette, no outline -> identical to a plain cell average.
	src := gradientImg(32, 32)
	got, err := Apply("pixelart", src, Params{"cell": 8.0, "bits": 8.0})
	if err != nil {
		t.Fatal(err)
	}
	want, err := Apply("pixelate", src, Params{"blockSize": 8.0})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("pixelart bits=8 must match pixelate")
	}
}

func TestPixelArt1BitIsTwoLevels(t *testing.T) {
	src := gradientImg(16, 16)
	out, err := Apply("pixelart", src, Params{"cell": 1.0, "bits": 1.0})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(out.Pix); i += 4 {
		for c := 0; c < 3; c++ {
			if v := out.Pix[i+c]; v != 0 && v != 255 {
				t.Fatalf("channel value %d is not 1-bit", v)
			}
		}
	}
}

func TestPixelArtPaletteAndOutlineRun(t *testing.T) {
	src := gradientImg(35, 27)
	assertBoundsFull(t, "pixelart", src, Params{"cell": 5.0, "palette": "nes"})
	assertBoundsFull(t, "pixelart", src, Params{"cell": 5.0, "bits": 3.0, "outline": true})
}

func TestPixelArtNoMutateAndDeterministic(t *testing.T) {
	src := gradientImg(30, 20)
	p := Params{"cell": 6.0, "palette": "gameboy", "outline": true}
	assertNoMutate(t, "pixelart", src, p)
	assertDeterministic(t, "pixelart", src, p)
}

func BenchmarkPixelArt1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"cell": 6.0, "palette": "pico8", "outline": true}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("pixelart", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
