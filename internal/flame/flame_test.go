package flame

import (
	"bytes"
	"encoding/json"
	"image"
	"testing"

	"bitbrush/internal/generators"
)

func run(t *testing.T, p string, w, h int) *image.RGBA {
	t.Helper()
	img, err := generators.Render("flame", json.RawMessage(p), w, h)
	if err != nil {
		t.Fatalf("render %s: %v", p, err)
	}
	if img.Bounds() != image.Rect(0, 0, w, h) {
		t.Fatalf("bounds %v", img.Bounds())
	}
	return img
}

func lit(img *image.RGBA, r, g, b uint8) int {
	var n int
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] != r || img.Pix[i+1] != g || img.Pix[i+2] != b {
			n++
		}
	}
	return n
}

func TestFlameRendersSomething(t *testing.T) {
	// A few seeds — most converge; at least one must produce a real image.
	ok := 0
	for s := 1; s <= 6; s++ {
		img, err := generators.Render("flame", json.RawMessage(`{"seed":`+itoa(s)+`,"iterations":300000}`), 160, 160)
		if err != nil {
			continue
		}
		if lit(img, 5, 5, 7) > 300 {
			ok++
		}
	}
	if ok == 0 {
		t.Fatal("no seed produced a visible flame")
	}
}

func TestFlameDeterministic(t *testing.T) {
	p := `{"seed":3,"iterations":300000}`
	a := run(t, p, 150, 150)
	b := run(t, p, 150, 150)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("same seed diverged")
	}
	c := run(t, `{"seed":99,"iterations":300000}`, 150, 150)
	if bytes.Equal(a.Pix, c.Pix) {
		t.Fatal("different seed identical")
	}
}

func TestFlameSymmetry(t *testing.T) {
	img := run(t, `{"seed":3,"iterations":400000,"symmetry":5}`, 160, 160)
	if lit(img, 5, 5, 7) < 300 {
		t.Fatal("symmetric flame plotted almost nothing")
	}
}

func TestFlameColour(t *testing.T) {
	img := run(t, `{"seed":3,"iterations":400000}`, 160, 160)
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] != img.Pix[i+1] || img.Pix[i+1] != img.Pix[i+2] {
			return
		}
	}
	t.Fatal("flame image is monochrome")
}

func TestFlameBadParams(t *testing.T) {
	if _, err := generators.Render("flame", json.RawMessage(`{`), 32, 32); err == nil {
		t.Fatal("want error on malformed JSON")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
