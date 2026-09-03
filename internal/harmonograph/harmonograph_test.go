package harmonograph

import (
	"bytes"
	"encoding/json"
	"image"
	"testing"

	"bitbrush/internal/generators"
)

func run(t *testing.T, p string, w, h int) *image.RGBA {
	t.Helper()
	img, err := generators.Render("harmonograph", json.RawMessage(p), w, h)
	if err != nil {
		t.Fatalf("render %s: %v", p, err)
	}
	if img.Bounds() != image.Rect(0, 0, w, h) {
		t.Fatalf("bounds %v", img.Bounds())
	}
	return img
}

func drew(img *image.RGBA, r, g, b uint8) bool {
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] != r || img.Pix[i+1] != g || img.Pix[i+2] != b {
			return true
		}
	}
	return false
}

func TestHarmonographDefaults(t *testing.T) {
	img := run(t, "", 240, 240)
	if !drew(img, 0x0e, 0x0e, 0x12) {
		t.Fatal("nothing drawn")
	}
}

func TestHarmonographDeterministic(t *testing.T) {
	a := run(t, `{"seed":5,"steps":8000}`, 200, 200)
	b := run(t, `{"seed":5,"steps":8000}`, 200, 200)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("same seed diverged")
	}
	c := run(t, `{"seed":6,"steps":8000}`, 200, 200)
	if bytes.Equal(a.Pix, c.Pix) {
		t.Fatal("different seed identical")
	}
}

func TestHarmonographColorful(t *testing.T) {
	img := run(t, `{"seed":2,"steps":10000,"colorful":true}`, 220, 220)
	// colourful output should contain a pixel whose channels are not all equal
	tinted := false
	for i := 0; i < len(img.Pix); i += 4 {
		r, g, b := img.Pix[i], img.Pix[i+1], img.Pix[i+2]
		if r != g || g != b {
			tinted = true
			break
		}
	}
	if !tinted {
		t.Fatal("colorful=true produced a monochrome image")
	}
}

func TestHarmonographClampsAndBadParams(t *testing.T) {
	if _, err := generators.Render("harmonograph", json.RawMessage(`{"steps":10}`), 64, 64); err != nil {
		t.Fatalf("tiny steps should clamp, not error: %v", err)
	}
	if _, err := generators.Render("harmonograph", json.RawMessage(`{bad`), 64, 64); err == nil {
		t.Fatal("want error on malformed JSON")
	}
}

func TestHarmonographTimeAnimates(t *testing.T) {
	t0 := run(t, `{"seed":12,"steps":8000,"time":0}`, 180, 180)
	t1 := run(t, `{"seed":12,"steps":8000,"time":2.5}`, 180, 180)
	t0Repeat := run(t, `{"seed":12,"steps":8000,"time":0}`, 180, 180)
	if !bytes.Equal(t0.Pix, t0Repeat.Pix) {
		t.Fatal("same time should be deterministic")
	}
	if bytes.Equal(t0.Pix, t1.Pix) {
		t.Fatal("different time should animate harmonograph")
	}
}
