package truchet

import (
	"bytes"
	"encoding/json"
	"image"
	"testing"

	"bitbrush/internal/generators"
)

func run(t *testing.T, p string, w, h int) *image.RGBA {
	t.Helper()
	img, err := generators.Render("truchet", json.RawMessage(p), w, h)
	if err != nil {
		t.Fatalf("render %s: %v", p, err)
	}
	if img.Bounds() != image.Rect(0, 0, w, h) {
		t.Fatalf("bounds %v, want %dx%d", img.Bounds(), w, h)
	}
	return img
}

func nonBg(img *image.RGBA, r, g, b uint8) int {
	var n int
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] != r || img.Pix[i+1] != g || img.Pix[i+2] != b {
			n++
		}
	}
	return n
}

func TestTruchetEmptyParamsAndSize(t *testing.T) {
	img := run(t, "", 200, 150)
	// default bg is #141414
	if nonBg(img, 0x14, 0x14, 0x14) == 0 {
		t.Fatal("nothing drawn with default params")
	}
}

func TestTruchetDeterministic(t *testing.T) {
	a := run(t, `{"seed":7,"tiles":10}`, 180, 180)
	b := run(t, `{"seed":7,"tiles":10}`, 180, 180)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("same seed produced different pixels")
	}
	c := run(t, `{"seed":8,"tiles":10}`, 180, 180)
	if bytes.Equal(a.Pix, c.Pix) {
		t.Fatal("different seed produced identical pixels")
	}
}

func TestTruchetStyles(t *testing.T) {
	for _, s := range []string{"arcs", "lines", "maze", "triangles"} {
		img := run(t, `{"seed":3,"tiles":8,"style":"`+s+`"}`, 160, 160)
		if nonBg(img, 0x14, 0x14, 0x14) == 0 {
			t.Fatalf("style %s drew nothing", s)
		}
	}
}

func TestTruchetMultiScaleAndColorful(t *testing.T) {
	img := run(t, `{"seed":11,"tiles":6,"multiScale":true,"colorful":true}`, 200, 200)
	if nonBg(img, 0x14, 0x14, 0x14) == 0 {
		t.Fatal("multiscale/colorful drew nothing")
	}
}

func TestTruchetBadParams(t *testing.T) {
	if _, err := generators.Render("truchet", json.RawMessage(`{"tiles":`), 32, 32); err == nil {
		t.Fatal("want error on malformed JSON")
	}
}
