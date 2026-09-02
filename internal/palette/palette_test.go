package palette

import (
	"image"
	"testing"
)

// stripes builds a w x h opaque image made of vertical bands, one per
// colour in cs, left to right.
func stripes(w, h int, cs ...RGB) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	band := w / len(cs)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cs[min(x/band, len(cs)-1)]
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, 255
		}
	}
	return img
}

func rampImg(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := img.PixOffset(x, y)
			img.Pix[i] = uint8((x * 255) / max(w-1, 1))
			img.Pix[i+1] = uint8((y * 255) / max(h-1, 1))
			img.Pix[i+2] = uint8(((x + y) * 255) / max(w+h-2, 1))
			img.Pix[i+3] = 255
		}
	}
	return img
}

func TestMedianCutRespectsMax(t *testing.T) {
	pal := MedianCut(rampImg(64, 64), 5)
	if len(pal) == 0 || len(pal) > 5 {
		t.Fatalf("len(pal) = %d, want 1..5", len(pal))
	}
}

func TestMedianCutZeroOrNegative(t *testing.T) {
	if MedianCut(rampImg(8, 8), 0) != nil || MedianCut(rampImg(8, 8), -3) != nil {
		t.Fatal("n <= 0 must return nil")
	}
}

func TestMedianCutSingleIsMean(t *testing.T) {
	pal := MedianCut(stripes(8, 4, RGB{0, 0, 0}, RGB{255, 255, 255}), 1)
	if len(pal) != 1 {
		t.Fatalf("len(pal) = %d, want 1", len(pal))
	}
	// half black, half white -> mean ~127/128 on every channel.
	p := pal[0]
	if p.R < 120 || p.R > 135 || p.G < 120 || p.G > 135 || p.B < 120 || p.B > 135 {
		t.Fatalf("mean colour = %+v, want ~grey 128", p)
	}
}

func TestMedianCutFewerDistinctColours(t *testing.T) {
	c0, c1, c2 := RGB{20, 20, 200}, RGB{200, 20, 20}, RGB{20, 200, 20}
	pal := MedianCut(stripes(12, 4, c0, c1, c2), 16)
	if len(pal) == 0 || len(pal) > 3 {
		t.Fatalf("len(pal) = %d, want 1..3 for 3 distinct colours", len(pal))
	}
	for _, p := range pal {
		if p != c0 && p != c1 && p != c2 {
			t.Fatalf("palette entry %+v is not one of the source colours", p)
		}
	}
}

func TestMedianCutDeterministic(t *testing.T) {
	img := rampImg(50, 37)
	a, b := MedianCut(img, 12), MedianCut(img, 12)
	if len(a) != len(b) {
		t.Fatalf("lengths differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("entry %d differs: %+v vs %+v", i, a[i], b[i])
		}
	}
}

func TestNearestPicksClosest(t *testing.T) {
	pal := []RGB{{0, 0, 0}, {255, 255, 255}, {255, 0, 0}}
	for _, space := range []Space{SpaceSRGB, SpaceOKLab} {
		if got := Nearest(RGB{12, 12, 12}, pal, space); got != 0 {
			t.Fatalf("space %d: near-black -> %d, want 0", space, got)
		}
		if got := Nearest(RGB{240, 245, 240}, pal, space); got != 1 {
			t.Fatalf("space %d: near-white -> %d, want 1", space, got)
		}
		if got := Nearest(RGB{200, 30, 30}, pal, space); got != 2 {
			t.Fatalf("space %d: near-red -> %d, want 2", space, got)
		}
	}
}

// TestMapperExactAtCellCentres checks Mapper against the byte-for-byte
// Nearest() at points that sit exactly on a lookup-table cell centre —
// where the two are defined to agree (Mapper resolves any other colour to
// its cell's centre answer, trading a little accuracy for O(1) lookups).
func TestMapperExactAtCellCentres(t *testing.T) {
	pal := MedianCut(rampImg(40, 40), 8)
	const cellsPerAxis = 1 << mapperBits // 64
	for _, space := range []Space{SpaceSRGB, SpaceOKLab} {
		m := NewMapper(pal, space)
		for rc := 0; rc < cellsPerAxis; rc += 7 {
			for gc := 0; gc < cellsPerAxis; gc += 11 {
				for bc := 0; bc < cellsPerAxis; bc += 13 {
					c := RGB{cellCentreByte(rc), cellCentreByte(gc), cellCentreByte(bc)}
					want := Nearest(c, pal, space)
					if got := m.Index(c); got != want {
						t.Fatalf("space %d colour %+v: Mapper %d != Nearest %d", space, c, got, want)
					}
				}
			}
		}
	}
}

// cellCentreByte returns the sRGB byte at the centre of lookup-table
// cell index cell (0..2^mapperBits-1).
func cellCentreByte(cell int) uint8 {
	const shift = 8 - mapperBits
	return uint8(cell<<shift) | uint8(1<<(shift-1))
}

func TestMedianCutAllTransparent(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8)) // alpha 0 everywhere
	if MedianCut(img, 4) != nil {
		t.Fatal("a fully transparent image has no colours to extract")
	}
}
