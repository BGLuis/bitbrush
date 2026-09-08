package filters

import (
	"bytes"
	"testing"
)

func TestMosaicSolidUnchanged(t *testing.T) {
	src := solid(24, 24, 33, 99, 140, 255)
	out, err := Apply("mosaic", src, Params{"cellWidth": 6.0, "cellHeight": 6.0})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Pix, src.Pix) {
		t.Fatal("solid image must be unchanged by mosaic with no gap/palette")
	}
}

func TestMosaicTileIsAverage(t *testing.T) {
	// 2x2: one white pixel + three black -> tile mean 63.
	src := solid(2, 2, 0, 0, 0, 255)
	src.Pix[0], src.Pix[1], src.Pix[2] = 255, 255, 255
	out, err := Apply("mosaic", src, Params{"cellWidth": 2.0, "cellHeight": 2.0})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i += 4 {
		if out.Pix[i] != 63 || out.Pix[i+1] != 63 || out.Pix[i+2] != 63 {
			t.Fatalf("pixel %d = %v; want 63,63,63", i/4, out.Pix[i:i+3])
		}
	}
}

func TestMosaicGapAndPaletteRun(t *testing.T) {
	src := gradientImg(33, 25)
	assertBoundsFull(t, "mosaic", src, Params{"cellWidth": 7.0, "cellHeight": 5.0, "gap": 2.0, "gapColor": "#111111"})
	assertBoundsFull(t, "mosaic", src, Params{"cellWidth": 6.0, "cellHeight": 6.0, "palette": "gameboy"})
	assertBoundsFull(t, "mosaic", src, Params{"cellWidth": 6.0, "cellHeight": 6.0, "bayerOverlay": true})
}

func TestMosaicNoMutateAndDeterministic(t *testing.T) {
	src := gradientImg(30, 20)
	p := Params{"cellWidth": 6.0, "cellHeight": 6.0, "gap": 1.0, "palette": "pico8"}
	assertNoMutate(t, "mosaic", src, p)
	assertDeterministic(t, "mosaic", src, p)
}

func BenchmarkMosaic1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"cellWidth": 12.0, "cellHeight": 12.0, "palette": "c64"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("mosaic", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
