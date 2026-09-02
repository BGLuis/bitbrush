package attractor

import (
	"bytes"
	"encoding/json"
	"image"
	"testing"

	"bitbrush/internal/generators"
)

func run(t *testing.T, p string, w, h int) *image.RGBA {
	t.Helper()
	img, err := generators.Render("attractor", json.RawMessage(p), w, h)
	if err != nil {
		t.Fatalf("render %s: %v", p, err)
	}
	if img.Bounds() != image.Rect(0, 0, w, h) {
		t.Fatalf("bounds %v", img.Bounds())
	}
	return img
}

func litPixels(img *image.RGBA, r, g, b uint8) int {
	var n int
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] != r || img.Pix[i+1] != g || img.Pix[i+2] != b {
			n++
		}
	}
	return n
}

func TestAttractorTypes(t *testing.T) {
	// One vetted coefficient set per map that is known to produce a real
	// (space-filling) attractor rather than a fixed point or short cycle.
	cases := map[string]string{
		"clifford": `{"type":"clifford","a":-1.4,"b":1.6,"c":1.0,"d":0.7,"iterations":300000}`,
		"dejong":   `{"type":"dejong","a":-2.24,"b":0.43,"c":-0.65,"d":-2.43,"iterations":300000}`,
		"svensson": `{"type":"svensson","a":1.4,"b":1.56,"c":1.4,"d":-6.56,"iterations":300000}`,
	}
	for k, p := range cases {
		img := run(t, p, 160, 160)
		if litPixels(img, 7, 7, 11) < 200 {
			t.Fatalf("%s: almost nothing plotted", k)
		}
	}
}

func TestAttractorDeterministic(t *testing.T) {
	p := `{"type":"clifford","seed":9,"iterations":150000}`
	a := run(t, p, 150, 150)
	b := run(t, p, 150, 150)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("same seed diverged")
	}
	c := run(t, `{"type":"clifford","seed":10,"iterations":150000}`, 150, 150)
	if bytes.Equal(a.Pix, c.Pix) {
		t.Fatal("different seed identical")
	}
}

func TestAttractorColorBySpeed(t *testing.T) {
	img := run(t, `{"type":"clifford","seed":4,"iterations":300000,"colorBySpeed":true}`, 160, 160)
	tinted := false
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] != img.Pix[i+1] || img.Pix[i+1] != img.Pix[i+2] {
			tinted = true
			break
		}
	}
	if !tinted {
		t.Fatal("colorBySpeed produced a monochrome image")
	}
}

func TestAttractorLargeCoeffsStillRender(t *testing.T) {
	// De Jong / Clifford / Svensson are bounded maps, so even big
	// coefficients don't blow up — autoscale should still frame them.
	img := run(t, `{"type":"svensson","a":12,"b":9,"c":7,"d":15,"iterations":200000}`, 96, 96)
	if litPixels(img, 7, 7, 11) == 0 {
		t.Fatal("large-coefficient attractor plotted nothing")
	}
}

func TestAttractorBadParams(t *testing.T) {
	if _, err := generators.Render("attractor", json.RawMessage(`{`), 32, 32); err == nil {
		t.Fatal("want error on malformed JSON")
	}
}
