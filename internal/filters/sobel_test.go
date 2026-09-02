package filters

import (
	"bytes"
	"image"
	"testing"
)

// verticalEdge builds a w x h opaque image: columns [0,split) black,
// columns [split,w) white.
func verticalEdge(w, h, split int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := img.PixOffset(x, y)
			c := byte(0)
			if x >= split {
				c = 255
			}
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c, c, c, 255
		}
	}
	return img
}

// horizontalEdge is verticalEdge transposed: rows [0,split) black, rest white.
func horizontalEdge(w, h, split int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		c := byte(0)
		if y >= split {
			c = 255
		}
		for x := 0; x < w; x++ {
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c, c, c, 255
		}
	}
	return img
}

func TestSobelSolidHasNoEdges(t *testing.T) {
	out, err := Apply("sobel", solid(8, 8, 100, 100, 100, 255), Params{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(out.Pix); i += 4 {
		if out.Pix[i] != 0 || out.Pix[i+1] != 0 || out.Pix[i+2] != 0 {
			t.Fatalf("pixel %d = %v, want black", i/4, out.Pix[i:i+3])
		}
		if out.Pix[i+3] != 255 {
			t.Fatalf("pixel %d alpha = %d, want 255", i/4, out.Pix[i+3])
		}
	}
}

func TestSobelSolidInverted(t *testing.T) {
	out, err := Apply("sobel", solid(8, 8, 100, 100, 100, 255), Params{"invert": true})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(out.Pix); i += 4 {
		if out.Pix[i] != 255 {
			t.Fatalf("pixel %d = %d, want 255 (inverted flat field)", i/4, out.Pix[i])
		}
	}
}

func TestSobelVerticalEdgeLightsUpBoundaryColumns(t *testing.T) {
	const w, h, split = 8, 4, 4
	out, err := Apply("sobel", verticalEdge(w, h, split), Params{})
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := out.Pix[out.PixOffset(x, y)]
			onBoundary := x == split-1 || x == split
			if onBoundary && v < 150 {
				t.Fatalf("col %d row %d = %d, want strong edge response", x, y, v)
			}
			if !onBoundary && v != 0 {
				t.Fatalf("col %d row %d = %d, want 0 away from the edge", x, y, v)
			}
		}
	}
}

func TestSobelIsRotationallySymmetric(t *testing.T) {
	vout, err := Apply("sobel", verticalEdge(8, 6, 4), Params{})
	if err != nil {
		t.Fatal(err)
	}
	hout, err := Apply("sobel", horizontalEdge(6, 8, 4), Params{})
	if err != nil {
		t.Fatal(err)
	}
	maxV, maxH := maxLuma(vout), maxLuma(hout)
	if maxV != maxH {
		t.Fatalf("vertical edge peak %d != horizontal edge peak %d", maxV, maxH)
	}
	if maxV < 150 {
		t.Fatalf("edge peak %d unexpectedly weak", maxV)
	}
}

func TestSobelThresholdBinarises(t *testing.T) {
	out, err := Apply("sobel", gradientImg(32, 32), Params{"threshold": 100.0})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(out.Pix); i += 4 {
		if v := out.Pix[i]; v != 0 && v != 255 {
			t.Fatalf("pixel %d = %d, want 0 or 255 with a threshold set", i/4, v)
		}
	}
}

func TestSobelAlphaPreserved(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2] = 30, 90, 150
		src.Pix[i+3] = byte(i) // 0, 4, 8, ... distinct alphas
	}
	out, err := Apply("sobel", src, Params{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 3; i < len(out.Pix); i += 4 {
		if out.Pix[i] != src.Pix[i] {
			t.Fatalf("pixel %d alpha = %d, want %d", i/4, out.Pix[i], src.Pix[i])
		}
	}
}

func TestSobelDoesNotMutateSource(t *testing.T) {
	src := gradientImg(20, 13)
	before := append([]byte(nil), src.Pix...)
	if _, err := Apply("sobel", src, Params{}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src.Pix, before) {
		t.Fatal("sobel mutated its source image")
	}
}

func TestSobelDeterministic(t *testing.T) {
	assertDeterministic(t, "sobel", gradientImg(20, 13), Params{"threshold": 0.0})
}

func TestConvolve3x3IdentityKernel(t *testing.T) {
	plane, w, h := lumaPlane(gradientImg(9, 7))
	out := convolve3x3(plane, w, h, [9]float64{0, 0, 0, 0, 1, 0, 0, 0, 0})
	for i := range plane {
		if out[i] != plane[i] {
			t.Fatalf("index %d: identity kernel changed %v to %v", i, plane[i], out[i])
		}
	}
}

func BenchmarkSobel1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("sobel", src, p); err != nil {
			b.Fatal(err)
		}
	}
}

// maxLuma returns the largest R value across an RGBA image.
func maxLuma(img *image.RGBA) byte {
	var m byte
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] > m {
			m = img.Pix[i]
		}
	}
	return m
}
