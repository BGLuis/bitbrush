package filters

import (
	"bytes"
	"testing"
)

func TestBlocksQuadrantSolidUnchanged(t *testing.T) {
	// In quadrant mode every sub-cell equals the cell mean, so all quads are
	// "on" and painted with the cell's own colour: a solid image round-trips.
	src := solid(20, 20, 60, 120, 180, 255)
	out, err := Apply("blocks", src, Params{"cellWidth": 5.0, "cellHeight": 5.0, "mode": "quadrant"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Pix, src.Pix) {
		t.Fatal("solid image must be unchanged by blocks/quadrant")
	}
}

func TestBlocksModesRun(t *testing.T) {
	src := gradientImg(31, 23)
	for _, m := range []string{"quadrant", "halves", "shade"} {
		assertBoundsFull(t, "blocks", src, Params{"cellWidth": 4.0, "cellHeight": 6.0, "mode": m})
	}
}

func TestBlocksNoMutateAndDeterministic(t *testing.T) {
	src := gradientImg(30, 18)
	p := Params{"cellWidth": 6.0, "cellHeight": 6.0, "mode": "shade", "colored": true}
	assertNoMutate(t, "blocks", src, p)
	assertDeterministic(t, "blocks", src, p)
}

func TestBlocksDefault(t *testing.T) {
	assertDefaultMatches(t, "blocks", gradientImg(24, 24),
		Params{"cellWidth": 6.0, "cellHeight": 6.0, "mode": "quadrant", "colored": true, "invert": false, "background": "#000000"})
}

func BenchmarkBlocks1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"cellWidth": 6.0, "cellHeight": 6.0, "mode": "quadrant"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("blocks", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
