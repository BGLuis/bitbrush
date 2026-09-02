package lsystem

import (
	"bytes"
	"encoding/json"
	"image"
	"testing"

	"bitbrush/internal/generators"
)

func run(t *testing.T, p string, w, h int) *image.RGBA {
	t.Helper()
	img, err := generators.Render("lsystem", json.RawMessage(p), w, h)
	if err != nil {
		t.Fatalf("render %s: %v", p, err)
	}
	if img.Bounds() != image.Rect(0, 0, w, h) {
		t.Fatalf("bounds %v", img.Bounds())
	}
	return img
}

func inked(img *image.RGBA, r, g, b uint8) int {
	var n int
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] != r || img.Pix[i+1] != g || img.Pix[i+2] != b {
			n++
		}
	}
	return n
}

func TestLSystemPresets(t *testing.T) {
	for _, name := range []string{"koch", "plant", "dragon", "sierpinski", "tree"} {
		img := run(t, `{"preset":"`+name+`"}`, 220, 220)
		if inked(img, 0x12, 0x13, 0x0f) < 100 {
			t.Fatalf("preset %s drew almost nothing", name)
		}
	}
}

func TestLSystemCustom(t *testing.T) {
	img := run(t, `{"preset":"custom","axiom":"F","rules":"F=F+F-F-F+F","angleDeg":90,"iterations":3}`, 200, 200)
	if inked(img, 0x12, 0x13, 0x0f) < 100 {
		t.Fatal("custom Koch drew almost nothing")
	}
}

func TestLSystemJitterDeterministic(t *testing.T) {
	p := `{"preset":"plant","seed":5,"jitter":0.3}`
	a := run(t, p, 200, 200)
	b := run(t, p, 200, 200)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("same seed+jitter diverged")
	}
	c := run(t, `{"preset":"plant","seed":6,"jitter":0.3}`, 200, 200)
	if bytes.Equal(a.Pix, c.Pix) {
		t.Fatal("different seed identical under jitter")
	}
	// jitter changes the drawing vs no jitter
	d := run(t, `{"preset":"plant","seed":5,"jitter":0}`, 200, 200)
	if bytes.Equal(a.Pix, d.Pix) {
		t.Fatal("jitter had no effect")
	}
}

func TestLSystemColorByDepth(t *testing.T) {
	img := run(t, `{"preset":"tree","colorByDepth":true}`, 200, 200)
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] != img.Pix[i+1] || img.Pix[i+1] != img.Pix[i+2] {
			return
		}
	}
	t.Fatal("colorByDepth produced a monochrome image")
}

func TestLSystemErrors(t *testing.T) {
	if _, err := generators.Render("lsystem", json.RawMessage(`{"preset":"custom","axiom":""}`), 64, 64); err == nil {
		t.Fatal("want error for empty axiom")
	}
	if _, err := generators.Render("lsystem", json.RawMessage(`{"preset":"custom","axiom":"+-+-"}`), 64, 64); err == nil {
		t.Fatal("want error when there are no forward moves")
	}
	if _, err := generators.Render("lsystem", json.RawMessage(`{bad`), 64, 64); err == nil {
		t.Fatal("want error on malformed JSON")
	}
}
