package palette

import (
	"image"
	"testing"
)

func eqPal(a, b []RGB) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestExtractMedianCutDispatch(t *testing.T) {
	img := rampImg(40, 32)
	got := Extract(img, ExtractOptions{Count: 6, Method: MethodMedianCut, Sort: SortLuma})
	want := SortPalette(MedianCut(img, 6), SortLuma, img)
	if !eqPal(got, want) {
		t.Fatalf("median-cut dispatch\n got %+v\nwant %+v", got, want)
	}
}

func TestExtractKMeansDispatch(t *testing.T) {
	img := rampImg(40, 32)
	got := Extract(img, ExtractOptions{
		Count: 5, Method: MethodKMeans, Space: SpaceOKLab, Sort: SortNone, Seed: 11,
	})
	want := SortPalette(
		KMeans(img, 5, KMeansOptions{Space: SpaceOKLab, Seed: 11}),
		SortNone, img,
	)
	if !eqPal(got, want) {
		t.Fatalf("k-means dispatch\n got %+v\nwant %+v", got, want)
	}
}

func TestExtractAlphaThreshold(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			i := img.PixOffset(x, y)
			if x < 10 {
				img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = 200, 30, 30, 255
			} else {
				img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = 30, 30, 200, 100
			}
		}
	}

	kept := Extract(img, ExtractOptions{Count: 4, Method: MethodMedianCut, AlphaThreshold: 128})
	if len(kept) == 0 {
		t.Fatal("threshold dropped every pixel")
	}
	for _, c := range kept {
		if c.B > c.R {
			t.Fatalf("transparent blue leaked past threshold 128: %+v", c)
		}
	}

	all := Extract(img, ExtractOptions{Count: 4, Method: MethodMedianCut, AlphaThreshold: 0})
	sawBlue := false
	for _, c := range all {
		if c.B > c.R {
			sawBlue = true
		}
	}
	if !sawBlue {
		t.Fatal("threshold 0 should keep the semi-transparent blue")
	}
}
