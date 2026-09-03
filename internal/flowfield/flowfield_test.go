package flowfield

import (
	"bytes"
	"encoding/json"
	"image"
	"testing"

	"bitbrush/internal/generators"
)

func run(t *testing.T, p string, w, h int) *image.RGBA {
	t.Helper()
	img, err := generators.Render("flowfield", json.RawMessage(p), w, h)
	if err != nil {
		t.Fatalf("render %s: %v", p, err)
	}
	if img.Bounds() != image.Rect(0, 0, w, h) {
		t.Fatalf("bounds %v", img.Bounds())
	}
	return img
}

func darkCount(img *image.RGBA) int {
	var n int
	for i := 0; i < len(img.Pix); i += 4 {
		if int(img.Pix[i])+int(img.Pix[i+1])+int(img.Pix[i+2]) < 300 {
			n++
		}
	}
	return n
}

func TestFlowFieldDefault(t *testing.T) {
	img := run(t, "", 200, 150)
	if darkCount(img) < 100 {
		t.Fatal("expected many ink strokes, got almost none")
	}
}

func TestFlowFieldDeterministic(t *testing.T) {
	a := run(t, `{"seed":9,"density":0.5}`, 180, 140)
	b := run(t, `{"seed":9,"density":0.5}`, 180, 140)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("same seed diverged")
	}
	c := run(t, `{"seed":10,"density":0.5}`, 180, 140)
	if bytes.Equal(a.Pix, c.Pix) {
		t.Fatal("different seed identical")
	}
}

func TestFlowFieldCurlDiffersFromAngle(t *testing.T) {
	a := run(t, `{"seed":1,"density":0.6,"curl":false}`, 180, 140)
	b := run(t, `{"seed":1,"density":0.6,"curl":true}`, 180, 140)
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("curl flag had no effect")
	}
}

func TestFlowFieldPalettes(t *testing.T) {
	for _, pal := range []string{"ink", "indigo", "vermilion"} {
		run(t, `{"seed":2,"density":0.5,"palette":"`+pal+`"}`, 140, 110)
	}
}

func TestFlowFieldDensityScales(t *testing.T) {
	lo := run(t, `{"seed":4,"density":0.3,"grain":0}`, 160, 120)
	hi := run(t, `{"seed":4,"density":2.0,"grain":0}`, 160, 120)
	if darkCount(hi) <= darkCount(lo) {
		t.Fatalf("higher density should ink more: lo=%d hi=%d", darkCount(lo), darkCount(hi))
	}
}

func TestFlowFieldBadParams(t *testing.T) {
	if _, err := generators.Render("flowfield", json.RawMessage(`{`), 32, 32); err == nil {
		t.Fatal("want error on malformed JSON")
	}
}

func TestFlowFieldTimeAnimates(t *testing.T) {
	t0 := run(t, `{"seed":42,"density":0.6,"time":0}`, 160, 120)
	t1 := run(t, `{"seed":42,"density":0.6,"time":5.0}`, 160, 120)
	t0Repeat := run(t, `{"seed":42,"density":0.6,"time":0}`, 160, 120)
	if !bytes.Equal(t0.Pix, t0Repeat.Pix) {
		t.Fatal("same time should be deterministic")
	}
	if bytes.Equal(t0.Pix, t1.Pix) {
		t.Fatal("different time should animate flow field")
	}
}
