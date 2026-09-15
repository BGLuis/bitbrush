package mask

import (
	"image"
	"image/color"
	"testing"
)

func solidRGBA(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = c.R
		img.Pix[i+1] = c.G
		img.Pix[i+2] = c.B
		img.Pix[i+3] = c.A
	}
	return img
}

func TestComposeRect(t *testing.T) {
	orig := solidRGBA(100, 100, color.RGBA{R: 10, G: 10, B: 10, A: 255})
	filt := solidRGBA(100, 100, color.RGBA{R: 200, G: 200, B: 200, A: 255})

	m := &Mask{
		Kind:    KindRect,
		X:       0.25,
		Y:       0.25,
		W:       0.5,
		H:       0.5,
		Feather: 0,
	}

	out := Compose(orig, filt, m)

	// Pixel at (50, 50) is inside rect -> should be ~200
	idxInside := out.PixOffset(50, 50)
	if out.Pix[idxInside] < 190 {
		t.Errorf("center pixel R=%d, want ~200", out.Pix[idxInside])
	}

	// Pixel at (10, 10) is outside rect -> should be ~10
	idxOutside := out.PixOffset(10, 10)
	if out.Pix[idxOutside] > 20 {
		t.Errorf("corner pixel R=%d, want ~10", out.Pix[idxOutside])
	}
}

func TestComposeEllipse(t *testing.T) {
	orig := solidRGBA(100, 100, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	filt := solidRGBA(100, 100, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	m := &Mask{
		Kind:    KindEllipse,
		X:       0.2,
		Y:       0.2,
		W:       0.6,
		H:       0.6,
		Feather: 0.1,
	}

	out := Compose(orig, filt, m)

	// Center at (50, 50)
	idxCenter := out.PixOffset(50, 50)
	if out.Pix[idxCenter] != 255 {
		t.Errorf("center pixel R=%d, want 255", out.Pix[idxCenter])
	}

	// Far corner at (5, 5)
	idxCorner := out.PixOffset(5, 5)
	if out.Pix[idxCorner] != 0 {
		t.Errorf("corner pixel R=%d, want 0", out.Pix[idxCorner])
	}
}

func TestComposeLuma(t *testing.T) {
	// Gradient/pattern: left black, right white
	orig := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			c := uint8(x * 255 / 100)
			off := orig.PixOffset(x, y)
			orig.Pix[off] = c
			orig.Pix[off+1] = c
			orig.Pix[off+2] = c
			orig.Pix[off+3] = 255
		}
	}
	filt := solidRGBA(100, 100, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	m := &Mask{
		Kind:      KindLuma,
		Threshold: 0.5,
		Feather:   0,
	}

	out := Compose(orig, filt, m)

	// Left side (lum < 0.5) should be original (black-ish, not red)
	idxLeft := out.PixOffset(20, 50)
	if out.Pix[idxLeft] == 255 && out.Pix[idxLeft+1] == 0 {
		t.Errorf("left pixel filtered when it should be original")
	}

	// Right side (lum > 0.5) should be filtered (red)
	idxRight := out.PixOffset(80, 50)
	if out.Pix[idxRight] != 255 || out.Pix[idxRight+1] != 0 {
		t.Errorf("right pixel R=%d G=%d, want red (255, 0)", out.Pix[idxRight], out.Pix[idxRight+1])
	}
}

func TestComposePolygonTriangle(t *testing.T) {
	orig := solidRGBA(100, 100, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	filt := solidRGBA(100, 100, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	m := &Mask{Kind: KindPolygon, Points: []Point{
		{X: 0.5, Y: 0.1}, {X: 0.9, Y: 0.9}, {X: 0.1, Y: 0.9},
	}}

	out := Compose(orig, filt, m)

	// Deep inside the triangle
	if idx := out.PixOffset(50, 60); out.Pix[idx] != 255 {
		t.Errorf("inside triangle R=%d, want 255", out.Pix[idx])
	}
	// Above the apex, outside
	if idx := out.PixOffset(5, 5); out.Pix[idx] != 0 {
		t.Errorf("outside triangle R=%d, want 0", out.Pix[idx])
	}
}

func TestComposePolygonFeather(t *testing.T) {
	orig := solidRGBA(100, 100, color.RGBA{R: 10, G: 10, B: 10, A: 255})
	filt := solidRGBA(100, 100, color.RGBA{R: 200, G: 200, B: 200, A: 255})

	m := &Mask{
		Kind:    KindPolygon,
		Points:  []Point{{X: 0.2, Y: 0.2}, {X: 0.8, Y: 0.2}, {X: 0.8, Y: 0.8}, {X: 0.2, Y: 0.8}},
		Feather: 0.2, // featherPx = 0.2 * 100 * 0.5 = 10
	}

	out := Compose(orig, filt, m)

	// (25, 50): 5px inside the left edge (x=20) -> d=5, alpha=0.5 -> strictly between
	r := out.Pix[out.PixOffset(25, 50)]
	if r <= 10 || r >= 200 {
		t.Errorf("feathered edge R=%d, want strictly between 10 and 200", r)
	}
}

func TestComposePolygonDegenerateNoPanic(t *testing.T) {
	orig := solidRGBA(10, 10, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	filt := solidRGBA(10, 10, color.RGBA{R: 4, G: 5, B: 6, A: 255})

	m := &Mask{Kind: KindPolygon, Points: []Point{{X: 0.2, Y: 0.2}, {X: 0.8, Y: 0.8}}} // only 2 points

	out := Compose(orig, filt, m) // must not panic
	if idx := out.PixOffset(5, 5); out.Pix[idx] != 1 {
		t.Errorf("degenerate polygon R=%d, want original 1 (alpha 0)", out.Pix[idx])
	}
}

func TestComposePolygonInvert(t *testing.T) {
	orig := solidRGBA(100, 100, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	filt := solidRGBA(100, 100, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	m := &Mask{
		Kind:   KindPolygon,
		Points: []Point{{X: 0.5, Y: 0.1}, {X: 0.9, Y: 0.9}, {X: 0.1, Y: 0.9}},
		Invert: true,
	}

	out := Compose(orig, filt, m)
	if idx := out.PixOffset(50, 60); out.Pix[idx] != 0 {
		t.Errorf("inverted inside R=%d, want 0", out.Pix[idx])
	}
	if idx := out.PixOffset(5, 5); out.Pix[idx] != 255 {
		t.Errorf("inverted outside R=%d, want 255", out.Pix[idx])
	}
}

func TestComposeNil(t *testing.T) {
	orig := solidRGBA(10, 10, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	filt := solidRGBA(10, 10, color.RGBA{R: 4, G: 5, B: 6, A: 255})
	out := Compose(orig, filt, nil)
	if out != filt {
		t.Errorf("expected nil mask to return filtered directly")
	}
}
