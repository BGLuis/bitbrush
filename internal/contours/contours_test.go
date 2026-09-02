package contours

import (
	"bytes"
	"encoding/json"
	"image"
	"testing"

	"bitbrush/internal/generators"
)

func run(t *testing.T, p string, w, h int) *image.RGBA {
	t.Helper()
	img, err := generators.Render("contours", json.RawMessage(p), w, h)
	if err != nil {
		t.Fatalf("render %s: %v", p, err)
	}
	if img.Bounds() != image.Rect(0, 0, w, h) {
		t.Fatalf("bounds %v", img.Bounds())
	}
	return img
}

func distinct(img *image.RGBA) int {
	seen := map[uint32]struct{}{}
	for i := 0; i < len(img.Pix); i += 4 {
		k := uint32(img.Pix[i])<<16 | uint32(img.Pix[i+1])<<8 | uint32(img.Pix[i+2])
		seen[k] = struct{}{}
	}
	return len(seen)
}

func TestContoursDefault(t *testing.T) {
	img := run(t, "", 240, 180)
	if distinct(img) < 20 {
		t.Fatalf("expected a shaded, contoured image, got %d distinct colours", distinct(img))
	}
}

func TestContoursDeterministicAndSeedVaries(t *testing.T) {
	a := run(t, `{"seed":3}`, 200, 160)
	b := run(t, `{"seed":3}`, 200, 160)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("same seed diverged")
	}
	c := run(t, `{"seed":4}`, 200, 160)
	if bytes.Equal(a.Pix, c.Pix) {
		t.Fatal("different seed identical")
	}
}

func TestContoursTimeAnimates(t *testing.T) {
	a := run(t, `{"seed":1,"time":0}`, 200, 160)
	b := run(t, `{"seed":1,"time":3.5}`, 200, 160)
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("time had no effect")
	}
}

func TestContoursPalettes(t *testing.T) {
	for _, pal := range []string{"paper", "blueprint", "terrain"} {
		run(t, `{"seed":2,"palette":"`+pal+`"}`, 160, 120)
	}
}

func TestContoursNoHillshade(t *testing.T) {
	img := run(t, `{"seed":5,"hillshade":false,"palette":"paper"}`, 160, 120)
	// flat background + lines => far fewer distinct colours than the shaded version
	if distinct(img) < 2 {
		t.Fatal("no lines drawn without hillshade")
	}
}

func TestContoursBadParams(t *testing.T) {
	if _, err := generators.Render("contours", json.RawMessage(`{oops`), 32, 32); err == nil {
		t.Fatal("want error on malformed JSON")
	}
}
