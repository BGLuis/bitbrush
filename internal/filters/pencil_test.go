package filters

import (
	"bytes"
	"image"
	"math"
	"testing"
)

// A flat white field has no tonal edges, so the dodge burns out everywhere
// and the result is the bare paper colour.
func TestPencilFlatInputIsPaper(t *testing.T) {
	out, err := Apply("pencil", solid(20, 20, 255, 255, 255, 255), Params{"paper": "#f6f3ea"})
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0xf6, 0xf3, 0xea}
	for i := 0; i < len(out.Pix); i += 4 {
		for c := 0; c < 3; c++ {
			if diff := int(out.Pix[i+c]) - int(want[c]); diff < -1 || diff > 1 {
				t.Fatalf("pixel %d = %v, want ~%v", i/4, out.Pix[i:i+3], want)
			}
		}
	}
}

func TestPencilEdgeDrawsAStroke(t *testing.T) {
	out, err := Apply("pencil", verticalEdge(48, 12, 24), Params{"hatch": 0.0})
	if err != nil {
		t.Fatal(err)
	}
	var minV byte = 255
	for i := 0; i < len(out.Pix); i += 4 {
		if out.Pix[i] < minV {
			minV = out.Pix[i]
		}
	}
	if minV > 120 {
		t.Fatalf("darkest pixel = %d, want a distinct graphite stroke along the edge", minV)
	}
}

func TestPencilAlphaPreserved(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2] = 120, 200, 60
		src.Pix[i+3] = byte(i)
	}
	out, err := Apply("pencil", src, Params{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 3; i < len(out.Pix); i += 4 {
		if out.Pix[i] != src.Pix[i] {
			t.Fatalf("pixel %d alpha = %d, want %d", i/4, out.Pix[i], src.Pix[i])
		}
	}
}

func TestPencilDoesNotMutateSource(t *testing.T) {
	src := gradientImg(20, 13)
	before := append([]byte(nil), src.Pix...)
	if _, err := Apply("pencil", src, Params{"hatch": 0.5}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src.Pix, before) {
		t.Fatal("pencil mutated its source image")
	}
}

func TestPencilDeterministic(t *testing.T) {
	assertDeterministic(t, "pencil", gradientImg(24, 17), Params{"hatch": 0.4, "seed": 5.0})
}

func TestGaussianBlurPlaneConstantIsUnchanged(t *testing.T) {
	const w, h = 12, 9
	src := make([]float64, w*h)
	for i := range src {
		src[i] = 0.42
	}
	out := gaussianBlurPlane(src, w, h, 3)
	for i, v := range out {
		if math.Abs(v-0.42) > 1e-9 {
			t.Fatalf("index %d = %v, want 0.42 (blur of a constant)", i, v)
		}
	}
}

func TestGaussianBlurPlaneZeroSigmaCopies(t *testing.T) {
	src := []float64{1, 2, 3, 4, 5, 6}
	out := gaussianBlurPlane(src, 3, 2, 0)
	for i := range src {
		if out[i] != src[i] {
			t.Fatalf("index %d = %v, want %v (sigma 0 is a copy)", i, out[i], src[i])
		}
	}
	out[0] = 99
	if src[0] == 99 {
		t.Fatal("sigma-0 blur aliased the input slice")
	}
}

func BenchmarkPencil1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"blur": 8.0, "hatch": 0.3}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("pencil", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
