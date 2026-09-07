package compositor

import (
	"image"
	"testing"
)

func at(img *image.RGBA, x, y int) (r, g, b, a uint8) {
	o := img.PixOffset(x, y)
	return img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3]
}

func TestFitDimensionsAlwaysMatchFrame(t *testing.T) {
	src := solid(4, 2, 255, 0, 0, 255)
	for _, m := range []FitMode{FitCover, FitContain, FitStretch, FitCenter, FitTile, "bogus"} {
		out := Fit(src, 8, 8, m)
		if out.Bounds().Dx() != 8 || out.Bounds().Dy() != 8 {
			t.Fatalf("%s: got %v, want 8x8", m, out.Bounds())
		}
	}
}

func TestFitContainLetterboxes(t *testing.T) {
	src := solid(4, 2, 255, 0, 0, 255) // 2:1 into 8x8 -> scaled 8x4, centred rows 2..5
	out := Fit(src, 8, 8, FitContain)
	if _, _, _, a := at(out, 4, 0); a != 0 {
		t.Fatalf("top row alpha = %d, want 0 (letterbox)", a)
	}
	if r, _, _, a := at(out, 4, 3); a != 255 || r < 200 {
		t.Fatalf("centre = r%d a%d, want opaque red", r, a)
	}
	if _, _, _, a := at(out, 4, 7); a != 0 {
		t.Fatalf("bottom row alpha = %d, want 0 (letterbox)", a)
	}
}

func TestFitCoverFillsFrame(t *testing.T) {
	src := solid(4, 2, 255, 0, 0, 255)
	out := Fit(src, 8, 8, FitCover)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if r, _, _, a := at(out, x, y); a != 255 || r < 200 {
				t.Fatalf("(%d,%d) = r%d a%d, want opaque red everywhere", x, y, r, a)
			}
		}
	}
}

func TestFitCenterPadsWithoutScaling(t *testing.T) {
	src := solid(4, 2, 255, 0, 0, 255) // placed at ox=2, oy=3
	out := Fit(src, 8, 8, FitCenter)
	if _, _, _, a := at(out, 0, 0); a != 0 {
		t.Fatalf("corner alpha = %d, want 0", a)
	}
	if r, _, _, a := at(out, 3, 4); a != 255 || r < 200 {
		t.Fatalf("centre = r%d a%d, want opaque red", r, a)
	}
}

func TestFitCenterClipsOversizeSource(t *testing.T) {
	src := solid(8, 8, 0, 255, 0, 255) // bigger than frame -> fully covers 4x4
	out := Fit(src, 4, 4, FitCenter)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if _, g, _, a := at(out, x, y); a != 255 || g < 200 {
				t.Fatalf("(%d,%d) = g%d a%d, want opaque green", x, y, g, a)
			}
		}
	}
}

func TestFitTileRepeats(t *testing.T) {
	src := solid(3, 3, 0, 0, 255, 255)
	out := Fit(src, 8, 8, FitTile)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if _, _, b, a := at(out, x, y); a != 255 || b < 200 {
				t.Fatalf("(%d,%d) = b%d a%d, want opaque blue tile", x, y, b, a)
			}
		}
	}
}
