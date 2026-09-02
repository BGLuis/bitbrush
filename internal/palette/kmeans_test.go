package palette

import (
	"image"
	"testing"
)

func absi(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func TestKMeansDeterministic(t *testing.T) {
	img := rampImg(48, 40)
	opt := KMeansOptions{Space: SpaceOKLab, Iterations: 15, Seed: 42}
	a := KMeans(img, 6, opt)
	b := KMeans(img, 6, opt)
	if len(a) == 0 {
		t.Fatal("empty palette")
	}
	if len(a) != len(b) {
		t.Fatalf("length mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("entry %d differs: %+v vs %+v", i, a[i], b[i])
		}
	}
}

func TestKMeansSeedsStayValid(t *testing.T) {
	// Different seeds may converge to different palettes on a non-trivial
	// image; do not over-assert, just check both runs stay bounded.
	img := rampImg(50, 37)
	for _, seed := range []int64{1, 999} {
		pal := KMeans(img, 8, KMeansOptions{Space: SpaceOKLab, Seed: seed})
		if len(pal) == 0 || len(pal) > 8 {
			t.Fatalf("seed %d: len(pal)=%d, want 1..8", seed, len(pal))
		}
	}
}

func TestKMeansFlatColoursExact(t *testing.T) {
	c0, c1, c2 := RGB{10, 20, 200}, RGB{200, 20, 10}, RGB{20, 200, 30}
	pal := KMeans(stripes(12, 4, c0, c1, c2), 8, KMeansOptions{Space: SpaceOKLab, Seed: 7})
	if len(pal) != 3 {
		t.Fatalf("len(pal)=%d, want 3", len(pal))
	}
	want := map[RGB]bool{c0: true, c1: true, c2: true}
	for _, p := range pal {
		if !want[p] {
			t.Fatalf("unexpected colour %+v in %+v", p, pal)
		}
	}
}

func TestKMeansClustersSeparatedColours(t *testing.T) {
	cs := []RGB{{240, 20, 20}, {20, 240, 20}, {20, 20, 240}, {240, 240, 20}}
	img := stripes(64, 8, cs...)
	// Perturb alternate columns so the distinct count exceeds k and the
	// Lloyd refinement path runs rather than the exact-colours shortcut.
	for y := 0; y < 8; y++ {
		for x := 1; x < 64; x += 2 {
			i := img.PixOffset(x, y)
			img.Pix[i] ^= 1
		}
	}
	pal := KMeans(img, 4, KMeansOptions{Space: SpaceOKLab, Seed: 3})
	if len(pal) == 0 || len(pal) > 4 {
		t.Fatalf("len(pal)=%d, want 1..4", len(pal))
	}
	for _, want := range cs {
		near := false
		for _, p := range pal {
			if absi(int(p.R)-int(want.R)) < 50 &&
				absi(int(p.G)-int(want.G)) < 50 &&
				absi(int(p.B)-int(want.B)) < 50 {
				near = true
			}
		}
		if !near {
			t.Fatalf("no palette entry near %+v; got %+v", want, pal)
		}
	}
}

func TestKMeansK1IsMean(t *testing.T) {
	img := stripes(8, 4, RGB{0, 0, 0}, RGB{255, 255, 255})
	pal := KMeans(img, 1, KMeansOptions{})
	if len(pal) != 1 {
		t.Fatalf("len(pal)=%d, want 1", len(pal))
	}
	want := channelMean(samplePixels(img))
	if pal[0] != want {
		t.Fatalf("k=1 -> %+v, want mean %+v", pal[0], want)
	}
}

func TestKMeansK0Nil(t *testing.T) {
	if KMeans(rampImg(8, 8), 0, KMeansOptions{}) != nil {
		t.Fatal("k=0 must return nil")
	}
	if KMeans(rampImg(8, 8), -2, KMeansOptions{}) != nil {
		t.Fatal("k<0 must return nil")
	}
}

func TestKMeansAllTransparent(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	if KMeans(img, 4, KMeansOptions{}) != nil {
		t.Fatal("a fully transparent image has no colours")
	}
}
