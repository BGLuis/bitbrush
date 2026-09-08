package filters

import (
	"bytes"
	"testing"
)

func TestGlyphScreenWhiteIsBlank(t *testing.T) {
	// A white image has zero ink coverage (t=0), so every cell stays the
	// background colour regardless of shape.
	src := solid(24, 24, 255, 255, 255, 255)
	out, err := Apply("glyphscreen", src, Params{"cell": 6.0, "shape": "cross", "background": "#000000"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(out.Pix); i += 4 {
		if out.Pix[i] != 0 || out.Pix[i+1] != 0 || out.Pix[i+2] != 0 {
			t.Fatalf("pixel %d = %v,%v,%v; want black background", i/4, out.Pix[i], out.Pix[i+1], out.Pix[i+2])
		}
	}
}

func TestGlyphScreenShapesRun(t *testing.T) {
	src := gradientImg(29, 21)
	for _, s := range []string{"cross", "diagonal", "backslash", "vertical", "horizontal", "plus", "diamond", "dot"} {
		assertBoundsFull(t, "glyphscreen", src, Params{"cell": 5.0, "shape": s})
	}
}

func TestGlyphScreenNoMutateAndDeterministic(t *testing.T) {
	src := gradientImg(28, 20)
	p := Params{"cell": 7.0, "shape": "diamond", "colored": true}
	assertNoMutate(t, "glyphscreen", src, p)
	assertDeterministic(t, "glyphscreen", src, p)
}

func TestGlyphScreenDefault(t *testing.T) {
	assertDefaultMatches(t, "glyphscreen", gradientImg(24, 24),
		Params{"cell": 10.0, "shape": "cross", "colored": true, "background": "#000000", "weight": 2.0, "invert": false})
}

func TestGlyphScreenInvertDiffers(t *testing.T) {
	src := gradientImg(24, 24)
	a, _ := Apply("glyphscreen", src, Params{"cell": 6.0, "shape": "dot"})
	b, _ := Apply("glyphscreen", src, Params{"cell": 6.0, "shape": "dot", "invert": true})
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("invert must change the result")
	}
}

func BenchmarkGlyphScreen1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"cell": 8.0, "shape": "diamond"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("glyphscreen", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
