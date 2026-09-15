package compositor

import (
	"bytes"
	"image"
	"testing"
)

// quad returns a w x h opaque RGBA split into four flat-coloured quadrants:
// top-left red, top-right green, bottom-left blue, bottom-right yellow.
func quad(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			o := img.PixOffset(x, y)
			var r, g, b uint8
			switch {
			case x < w/2 && y < h/2:
				r, g, b = 255, 0, 0 // TL red
			case x >= w/2 && y < h/2:
				r, g, b = 0, 255, 0 // TR green
			case x < w/2 && y >= h/2:
				r, g, b = 0, 0, 255 // BL blue
			default:
				r, g, b = 255, 255, 0 // BR yellow
			}
			img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = r, g, b, 255
		}
	}
	return img
}

func TestTransformIdentityIsNoOp(t *testing.T) {
	src := solid(6, 6, 10, 20, 30, 255)
	out := ApplyTransform(src, Transform{Scale: 1}, 6, 6)
	for y := 0; y < 6; y++ {
		for x := 0; x < 6; x++ {
			r, g, b, a := at(out, x, y)
			if approxDiff(r, 10) > 2 || approxDiff(g, 20) > 2 || approxDiff(b, 30) > 2 || a != 255 {
				t.Fatalf("(%d,%d) = %d,%d,%d,%d, want ~10,20,30,255", x, y, r, g, b, a)
			}
		}
	}
}

func TestTransformOffsetShiftsContentRight(t *testing.T) {
	src := solid(8, 8, 255, 0, 0, 255)
	out := ApplyTransform(src, Transform{OffsetX: 0.3}, 8, 8)
	if _, _, _, a := at(out, 0, 4); a > 20 {
		t.Fatalf("left edge alpha = %d, want ~0 (content moved right)", a)
	}
	if r, _, _, a := at(out, 7, 4); a < 200 || r < 200 {
		t.Fatalf("right edge = r%d a%d, want opaque red", r, a)
	}
}

func TestTransformScaleDownRevealsTransparentMargin(t *testing.T) {
	src := solid(8, 8, 255, 0, 0, 255)
	out := ApplyTransform(src, Transform{Scale: 0.5}, 8, 8)
	if _, _, _, a := at(out, 0, 4); a > 20 {
		t.Fatalf("edge alpha = %d, want ~0 (shrunk content leaves a margin)", a)
	}
	if r, _, _, a := at(out, 4, 4); a < 200 || r < 200 {
		t.Fatalf("centre = r%d a%d, want opaque red", r, a)
	}
}

func TestTransformScaleChangesQuadrantBoundary(t *testing.T) {
	src := quad(16, 16)
	identity := ApplyTransform(src, Transform{Scale: 1}, 16, 16)
	zoomed := ApplyTransform(src, Transform{Scale: 2}, 16, 16)
	// Near the quadrant boundary, zooming pulls from a point closer to the
	// centre, so the two outputs must disagree somewhere along it.
	differs := false
	for y := 6; y < 11; y++ {
		for x := 6; x < 11; x++ {
			ri, gi, bi, _ := at(identity, x, y)
			rz, gz, bz, _ := at(zoomed, x, y)
			if ri != rz || gi != gz || bi != bz {
				differs = true
			}
		}
	}
	if !differs {
		t.Fatal("Scale: 2 produced identical output to Scale: 1 near the quadrant boundary")
	}
}

func TestTransformRotation180SwapsOppositeCorners(t *testing.T) {
	src := quad(8, 8)
	out := ApplyTransform(src, Transform{Rotation: 180}, 8, 8)
	// A 180-degree rotation is a point reflection through the centre
	// regardless of rotation-direction convention, so this holds either way.
	if r, g, b, _ := at(out, 1, 1); r < 200 || g < 200 || b > 60 {
		t.Fatalf("TL after 180: %d,%d,%d, want ~yellow (was BR)", r, g, b)
	}
	if r, g, b, _ := at(out, 6, 6); r < 200 || g > 60 || b > 60 {
		t.Fatalf("BR after 180: %d,%d,%d, want ~red (was TL)", r, g, b)
	}
	if r, g, b, _ := at(out, 6, 1); r > 60 || g > 60 || b < 200 {
		t.Fatalf("TR after 180: %d,%d,%d, want ~blue (was BL)", r, g, b)
	}
	if r, g, b, _ := at(out, 1, 6); r > 60 || g < 200 || b > 60 {
		t.Fatalf("BL after 180: %d,%d,%d, want ~green (was TR)", r, g, b)
	}
}

func TestTransformRotated45CornerGoesTransparentNotClamped(t *testing.T) {
	src := solid(8, 8, 255, 0, 0, 255)
	out := ApplyTransform(src, Transform{Rotation: 45}, 8, 8)
	if _, _, _, a := at(out, 0, 0); a > 20 {
		t.Fatalf("rotated-away corner alpha = %d, want ~0 (transparent, not clamped red)", a)
	}
	if r, _, _, a := at(out, 4, 4); a < 200 || r < 200 {
		t.Fatalf("centre = r%d a%d, want opaque red", r, a)
	}
}

func TestTransformDegenerateScaleDoesNotPanic(t *testing.T) {
	src := solid(4, 4, 10, 20, 30, 255)
	for _, s := range []float64{0, -1, 1e-10, -1e-10} {
		out := ApplyTransform(src, Transform{Scale: s}, 4, 4)
		if out.Bounds().Dx() != 4 || out.Bounds().Dy() != 4 {
			t.Fatalf("scale %v: got %v, want 4x4", s, out.Bounds())
		}
	}
}

func TestTransformDoesNotMutateInput(t *testing.T) {
	src := solid(4, 4, 10, 20, 30, 255)
	cp := append([]byte(nil), src.Pix...)
	ApplyTransform(src, Transform{OffsetX: 0.2, Rotation: 30, Scale: 1.5}, 4, 4)
	if !bytes.Equal(src.Pix, cp) {
		t.Fatal("ApplyTransform mutated its input")
	}
}

func TestTransformIsIdentity(t *testing.T) {
	cases := []struct {
		t    Transform
		want bool
	}{
		{Transform{}, true},
		{Transform{Scale: 1}, true},
		{Transform{Rotation: 360}, true},
		{Transform{Rotation: -360}, true},
		{Transform{OffsetX: 0.001}, false},
		{Transform{Scale: 1.5}, false},
		{Transform{Rotation: 90}, false},
	}
	for _, c := range cases {
		if got := c.t.IsIdentity(); got != c.want {
			t.Errorf("%+v.IsIdentity() = %v, want %v", c.t, got, c.want)
		}
	}
}

func approxDiff(v, want uint8) int {
	d := int(v) - int(want)
	if d < 0 {
		d = -d
	}
	return d
}
