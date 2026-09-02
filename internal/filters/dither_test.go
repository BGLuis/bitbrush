package filters

import (
	"bytes"
	"image"
	"testing"
)

func TestDitherLevelsGrayscaleIsBinaryAndAveragesOut(t *testing.T) {
	src := solid(40, 40, 128, 128, 128, 255)
	out, err := Apply("dither", src, Params{"levels": 2.0, "grayscale": true})
	if err != nil {
		t.Fatal(err)
	}
	var sum int
	for i := 0; i < len(out.Pix); i += 4 {
		r, g, b, a := out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3]
		if r != 0 && r != 255 {
			t.Fatalf("pixel %d R=%d, want 0 or 255", i/4, r)
		}
		if r != g || g != b {
			t.Fatalf("pixel %d = (%d,%d,%d), want grayscale (R=G=B)", i/4, r, g, b)
		}
		if a != 255 {
			t.Fatalf("pixel %d alpha = %d, want 255", i/4, a)
		}
		sum += int(r)
	}
	mean := sum / (40 * 40)
	if mean < 110 || mean > 146 {
		t.Fatalf("mean = %d, want close to the source value 128 (error diffusion should conserve it)", mean)
	}
}

func TestDitherLevelsPureBlackWhiteUnchanged(t *testing.T) {
	for _, v := range []uint8{0, 255} {
		out, err := Apply("dither", solid(8, 8, v, v, v, 255), Params{"levels": 2.0, "grayscale": true})
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < len(out.Pix); i += 4 {
			if out.Pix[i] != v {
				t.Fatalf("solid %d: pixel %d = %d, want unchanged", v, i/4, out.Pix[i])
			}
		}
	}
}

func TestDitherLevelsOutputOnlyContainsSteps(t *testing.T) {
	out, err := Apply("dither", gradientImg(48, 32), Params{"levels": 4.0, "grayscale": true})
	if err != nil {
		t.Fatal(err)
	}
	// 4 levels -> steps of 255/3 = 85: {0, 85, 170, 255}.
	valid := map[byte]bool{0: true, 85: true, 170: true, 255: true}
	for i := 0; i < len(out.Pix); i += 4 {
		if !valid[out.Pix[i]] {
			t.Fatalf("pixel %d R=%d, not one of the 4 allowed steps", i/4, out.Pix[i])
		}
	}
}

func TestDitherLevelsClamped(t *testing.T) {
	// levels=1 would divide by zero if not clamped to >= 2; levels=999
	// must clamp to <= 16. Neither should panic or error.
	if _, err := Apply("dither", gradientImg(6, 6), Params{"levels": 1.0, "grayscale": true}); err != nil {
		t.Fatalf("levels=1: %v", err)
	}
	if _, err := Apply("dither", gradientImg(6, 6), Params{"levels": 999.0, "grayscale": true}); err != nil {
		t.Fatalf("levels=999: %v", err)
	}
}

func TestDitherFirstPixelIsExactQuantization(t *testing.T) {
	// The scan always starts at (0,0) left-to-right (row 0 is never
	// reversed by serpentine), so no error has reached it yet: its output
	// must equal a direct quantisation of the source pixel.
	src := gradientImg(20, 20)
	cases := []Params{
		{"mode": "levels", "grayscale": true, "levels": 3.0},
		{"mode": "levels", "grayscale": false, "levels": 2.0},
	}
	for _, params := range cases {
		out, err := Apply("dither", src, params)
		if err != nil {
			t.Fatal(err)
		}
		wantR := levelQuant(int(params["levels"].(float64)))(float64(src.Pix[0]))
		if float64(out.Pix[0]) != wantR {
			t.Fatalf("%v: first pixel R=%d, want %v", params, out.Pix[0], wantR)
		}
	}
}

func TestDitherPaletteUsesOnlyPaletteColours(t *testing.T) {
	src := bands(30, 10, [3]byte{20, 20, 200}, [3]byte{200, 20, 20}, [3]byte{20, 200, 20}, [3]byte{200, 200, 20})
	out, err := Apply("dither", src, Params{"mode": "palette", "colors": 8.0})
	if err != nil {
		t.Fatal(err)
	}
	if n := distinctColors(out); n > 8 {
		t.Fatalf("output has %d colours, want <= 8 (the requested palette size)", n)
	}
}

func TestDitherPaletteAlphaPreserved(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2] = byte(i), byte(200-i), 60
		src.Pix[i+3] = byte((i / 4 * 7) % 256)
	}
	out, err := Apply("dither", src, Params{"mode": "palette", "colors": 6.0})
	if err != nil {
		t.Fatal(err)
	}
	for i := 3; i < len(out.Pix); i += 4 {
		if out.Pix[i] != src.Pix[i] {
			t.Fatalf("pixel %d alpha = %d, want %d", i/4, out.Pix[i], src.Pix[i])
		}
	}
}

func TestDitherSerpentineChangesOutput(t *testing.T) {
	src := gradientImg(30, 20)
	on, err := Apply("dither", src, Params{"levels": 3.0, "grayscale": true, "serpentine": true})
	if err != nil {
		t.Fatal(err)
	}
	off, err := Apply("dither", src, Params{"levels": 3.0, "grayscale": true, "serpentine": false})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(on.Pix, off.Pix) {
		t.Fatal("serpentine on/off produced identical output")
	}
}

func TestDitherDoesNotMutateSource(t *testing.T) {
	for _, params := range []Params{
		{"mode": "levels", "grayscale": true},
		{"mode": "levels", "grayscale": false},
		{"mode": "palette", "colors": 8.0},
	} {
		src := gradientImg(24, 17)
		before := append([]byte(nil), src.Pix...)
		if _, err := Apply("dither", src, params); err != nil {
			t.Fatalf("%v: %v", params, err)
		}
		if !bytes.Equal(src.Pix, before) {
			t.Fatalf("%v: dither mutated its source image", params)
		}
	}
}

func TestDitherDeterministic(t *testing.T) {
	img := gradientImg(28, 19)
	for _, params := range []Params{
		{"mode": "levels", "grayscale": true, "levels": 3.0},
		{"mode": "levels", "grayscale": false, "levels": 2.0},
		{"mode": "palette", "colors": 10.0},
	} {
		assertDeterministic(t, "dither", img, params)
	}
}

func BenchmarkDitherLevelsGrayscale1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"levels": 2.0, "grayscale": true}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("dither", src, p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDitherPalette1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"mode": "palette", "colors": 16.0}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("dither", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
