package filters

import (
	"bytes"
	"image"
	"math/rand"
	"testing"
)

func anyNonZero(shifts []int) bool {
	for _, s := range shifts {
		if s != 0 {
			return true
		}
	}
	return false
}

// --- glitchBandShifts: the RNG-facing logic, tested in isolation ---

func TestGlitchBandShiftsShareValueWithinBand(t *testing.T) {
	const h, sliceCount, maxShift = 20, 4, 15
	rng := rand.New(rand.NewSource(7))
	shifts := glitchBandShifts(h, sliceCount, maxShift, 1.0, rng) // always displace
	bandHeight := (h + sliceCount - 1) / sliceCount
	for band := 0; band < sliceCount; band++ {
		start, end := band*bandHeight, min((band+1)*bandHeight, h)
		want := shifts[start]
		if want < -maxShift || want > maxShift {
			t.Fatalf("band %d shift %d out of range [-%d,%d]", band, want, maxShift, maxShift)
		}
		for y := start; y < end; y++ {
			if shifts[y] != want {
				t.Fatalf("band %d: row %d shift %d != row %d shift %d", band, y, shifts[y], start, want)
			}
		}
	}
}

func TestGlitchBandShiftsDisabledBySliceCount(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	if s := glitchBandShifts(10, 0, 20, 0.5, rng); anyNonZero(s) {
		t.Fatal("sliceCount=0 must produce no shifts")
	}
}

func TestGlitchBandShiftsDisabledByMaxShift(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	if s := glitchBandShifts(10, 5, 0, 0.5, rng); anyNonZero(s) {
		t.Fatal("maxShift=0 must produce no shifts")
	}
}

func TestGlitchBandShiftsRespectZeroProbability(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	if s := glitchBandShifts(20, 4, 20, 0, rng); anyNonZero(s) {
		t.Fatal("sliceProbability=0 must produce no shifts")
	}
}

func TestGlitchBandShiftsVaryAcrossSeeds(t *testing.T) {
	found := false
	for seed := int64(0); seed < 50; seed++ {
		rng := rand.New(rand.NewSource(seed))
		if anyNonZero(glitchBandShifts(8, 4, 20, 1.0, rng)) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected at least one nonzero shift across 50 seeds with sliceProbability=1")
	}
}

// --- Glitch: the filter, using Apply ---

func TestGlitchIdentityWhenDisabled(t *testing.T) {
	src := gradientImg(24, 17)
	out, err := Apply("glitch", src, Params{
		"rgbShift": 0.0, "sliceCount": 0.0, "scanlines": false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Pix, src.Pix) {
		t.Fatal("rgbShift=0, sliceCount=0, scanlines=false must be an identity transform")
	}
}

func TestGlitchDeterministic(t *testing.T) {
	assertDeterministic(t, "glitch", gradientImg(30, 20), Params{
		"seed": 42.0, "rgbShift": 5.0, "sliceCount": 6.0, "maxSliceShift": 12.0,
	})
}

func TestGlitchSameSeedFreshParamsReproducible(t *testing.T) {
	src := gradientImg(30, 20)
	mk := func() Params {
		return Params{"seed": 42.0, "rgbShift": 5.0, "sliceCount": 6.0, "maxSliceShift": 12.0}
	}
	a, err := Apply("glitch", src, mk())
	if err != nil {
		t.Fatal(err)
	}
	b, err := Apply("glitch", src, mk())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("same seed with independently built params must reproduce the same image")
	}
}

func TestGlitchDifferentSeedsDiffer(t *testing.T) {
	src := gradientImg(30, 20)
	params := func(seed float64) Params {
		return Params{"seed": seed, "rgbShift": 5.0, "sliceCount": 6.0, "maxSliceShift": 12.0}
	}
	a, err := Apply("glitch", src, params(1))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Apply("glitch", src, params(2))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("different seeds produced identical output")
	}
}

func TestGlitchPreservesDimensions(t *testing.T) {
	src := gradientImg(13, 27) // odd size, doesn't divide evenly into bands
	out, err := Apply("glitch", src, Params{"sliceCount": 5.0, "maxSliceShift": 4.0})
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != image.Rect(0, 0, 13, 27) {
		t.Fatalf("bounds changed: %v", out.Bounds())
	}
}

func TestGlitchDoesNotMutateSource(t *testing.T) {
	src := gradientImg(24, 17)
	before := append([]byte(nil), src.Pix...)
	if _, err := Apply("glitch", src, Params{"seed": 9.0, "rgbShift": 6.0, "sliceCount": 4.0, "maxSliceShift": 10.0}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src.Pix, before) {
		t.Fatal("glitch mutated its source image")
	}
}

// TestGlitchRGBShiftSemantics builds a single row where each channel
// encodes its x position distinctly, and checks the shift directly:
// R comes from x-rgbShift, G is untouched, B comes from x+rgbShift, all
// clamped at the edges. sliceCount=0 removes the band-displacement pass
// from the picture entirely.
func TestGlitchRGBShiftSemantics(t *testing.T) {
	const w = 16
	src := image.NewRGBA(image.Rect(0, 0, w, 1))
	for x := 0; x < w; x++ {
		i := src.PixOffset(x, 0)
		src.Pix[i] = byte(x * 15)       // R encodes x
		src.Pix[i+1] = 100              // G constant
		src.Pix[i+2] = byte(255 - x*15) // B encodes x too, distinctly
		src.Pix[i+3] = 255
	}

	const shift = 3
	out, err := Apply("glitch", src, Params{"rgbShift": float64(shift), "sliceCount": 0.0})
	if err != nil {
		t.Fatal(err)
	}

	for x := 0; x < w; x++ {
		rx := clampInt(x-shift, 0, w-1)
		bx := clampInt(x+shift, 0, w-1)
		i := out.PixOffset(x, 0)
		if want := src.Pix[src.PixOffset(rx, 0)]; out.Pix[i] != want {
			t.Fatalf("x=%d: R=%d, want src.R(%d)=%d", x, out.Pix[i], rx, want)
		}
		if out.Pix[i+1] != 100 {
			t.Fatalf("x=%d: G=%d, want untouched 100", x, out.Pix[i+1])
		}
		if want := src.Pix[src.PixOffset(bx, 0)+2]; out.Pix[i+2] != want {
			t.Fatalf("x=%d: B=%d, want src.B(%d)=%d", x, out.Pix[i+2], bx, want)
		}
	}
}

func TestGlitchScanlinesDarkenOddRowsOnly(t *testing.T) {
	src := solid(6, 6, 200, 200, 200, 255)
	out, err := Apply("glitch", src, Params{
		"rgbShift": 0.0, "sliceCount": 0.0, "scanlines": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	const want200 = 200
	const wantDark = 164 // uint8(200 * 0.82)
	for y := 0; y < 6; y++ {
		for x := 0; x < 6; x++ {
			i := out.PixOffset(x, y)
			want := byte(want200)
			if y%2 == 1 {
				want = wantDark
			}
			if out.Pix[i] != want {
				t.Fatalf("row %d (odd=%v): R=%d, want %d", y, y%2 == 1, out.Pix[i], want)
			}
			if out.Pix[i+3] != 255 {
				t.Fatalf("row %d: alpha=%d, want untouched 255", y, out.Pix[i+3])
			}
		}
	}
}

// TestGlitchAppliesBandWrapShift replays the same RNG draw the filter will
// make (same seed, same call order) to build the expected image by hand,
// then checks the real Apply output matches exactly.
func TestGlitchAppliesBandWrapShift(t *testing.T) {
	const w, h, seed = 16, 8, int64(42)
	src := gradientImg(w, h)

	rng := rand.New(rand.NewSource(seed))
	shifts := glitchBandShifts(h, 1, 6, 1.0, rng)
	dx := shifts[0]

	out, err := Apply("glitch", src, Params{
		"seed": float64(seed), "sliceCount": 1.0, "maxSliceShift": 6.0,
		"sliceProbability": 1.0, "rgbShift": 0.0, "scanlines": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	want := newLike(src)
	b := src.Bounds()
	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := want.PixOffset(0, y)
		for x := 0; x < w; x++ {
			sx := ((x-dx)%w + w) % w
			s, d := si+sx*4, di+x*4
			copy(want.Pix[d:d+4], src.Pix[s:s+4])
		}
	}
	if !bytes.Equal(out.Pix, want.Pix) {
		t.Fatalf("dx=%d: glitch output doesn't match the hand-computed wrap shift", dx)
	}
}

func BenchmarkGlitch1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"seed": 1.0, "rgbShift": 6.0, "sliceCount": 8.0, "maxSliceShift": 20.0}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("glitch", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
