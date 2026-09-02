// Package palette extracts colour palettes from an image with median-cut
// and maps arbitrary colours onto a fixed palette. The same code backs the
// Colour Quantization filter and the standalone "palette from image"
// feature. Pure Go; deterministic for a given input.
package palette

import (
	"image"
	"slices"

	"bitbrush/internal/colorspace"
)

// RGB is an 8-bit colour without alpha. Palettes are alpha-agnostic; the
// caller keeps the source alpha.
type RGB struct{ R, G, B uint8 }

// Space selects the metric used when matching a colour to a palette entry.
type Space int

const (
	// SpaceSRGB measures distance in gamma-encoded sRGB — fast, but darker
	// hues get over-weighted.
	SpaceSRGB Space = iota
	// SpaceOKLab measures distance in OKLab — matches perception better.
	SpaceOKLab
)

// maxSamples caps how many pixels feed median-cut. Larger images are
// stride-sampled; median-cut is box statistics over the sample, so a
// 32K-pixel sample represents a photo's colour distribution about as well
// as the full frame while keeping the splitting sorts fast.
const maxSamples = 1 << 15

// MedianCut reduces img to a palette of at most n colours by repeatedly
// splitting the colour box with the widest channel range at its median
// along that channel. n <= 0 returns nil; n == 1 returns the single mean
// colour. The result is sorted, so it is deterministic for a given image.
func MedianCut(img *image.RGBA, n int) []RGB {
	if n <= 0 {
		return nil
	}
	return medianCutPixels(samplePixels(img), n)
}

// medianCutPixels runs the median-cut split loop over an already-sampled
// pixel set. Extract uses it to honour an alpha threshold that the plain
// samplePixels path does not apply.
func medianCutPixels(pixels []RGB, n int) []RGB {
	if n <= 0 || len(pixels) == 0 {
		return nil
	}

	boxes := []box{{pixels: pixels}}
	for len(boxes) < n {
		target, span := -1, 0
		for i := range boxes {
			if len(boxes[i].pixels) < 2 {
				continue
			}
			if _, s := boxes[i].widestAxis(); s > span {
				target, span = i, s
			}
		}
		if target < 0 {
			break // every remaining box is a single colour
		}
		lo, hi := boxes[target].split()
		boxes = append(boxes, box{})
		copy(boxes[target+1:], boxes[target:])
		boxes[target], boxes[target+1] = lo, hi
	}

	pal := make([]RGB, len(boxes))
	for i := range boxes {
		pal[i] = boxes[i].mean()
	}
	// Splitting a box that already holds a single colour still produces
	// two boxes with the same mean, so sort and dedupe the result.
	return sortDedupeRGB(pal)
}

// sortDedupeRGB sorts pal in place by (R,G,B) ascending and drops adjacent
// duplicates, returning a deterministic, duplicate-free palette. Shared by
// MedianCut and KMeans.
func sortDedupeRGB(pal []RGB) []RGB {
	if len(pal) == 0 {
		return pal
	}
	slices.SortFunc(pal, func(a, b RGB) int {
		if a.R != b.R {
			return int(a.R) - int(b.R)
		}
		if a.G != b.G {
			return int(a.G) - int(b.G)
		}
		return int(a.B) - int(b.B)
	})
	out := pal[:1]
	for _, c := range pal[1:] {
		if c != out[len(out)-1] {
			out = append(out, c)
		}
	}
	return out
}

// mapperBits is the per-channel resolution of a Mapper's lookup table:
// 6 bits -> a 64x64x64 (2^18) table, cells 4 units wide. Coarse enough to
// stay fast and small, fine enough that misclassified pixels sit within a
// palette-visible tolerance of the true nearest entry.
const mapperBits = 6

// Mapper maps arbitrary colours to the nearest entry of a fixed palette.
// It precomputes the answer for every cell of a coarse RGB lookup table
// (built lazily, one nearest-neighbour scan per cell) so that mapping a
// whole image is O(pixels) array reads instead of O(pixels * len(pal))
// distance computations. Build one per image/palette and reuse it.
type Mapper struct {
	pal   []RGB
	space Space
	labs  []colorspace.OKLab
	shift uint
	lut   []int32 // 0 = unset; stored value is (index + 1)
}

// NewMapper precomputes what the chosen space needs and returns a reusable
// mapper over pal.
func NewMapper(pal []RGB, space Space) *Mapper {
	m := &Mapper{
		pal:   pal,
		space: space,
		shift: 8 - mapperBits,
		lut:   make([]int32, 1<<(3*mapperBits)),
		labs:  labsFor(pal, space),
	}
	return m
}

// Index returns the palette index nearest to c.
func (m *Mapper) Index(c RGB) int {
	k := m.cellKey(c)
	if v := m.lut[k]; v != 0 {
		return int(v - 1)
	}
	i := m.nearest(m.cellCentre(c))
	m.lut[k] = int32(i) + 1
	return i
}

// At returns the palette colour nearest to c.
func (m *Mapper) At(c RGB) RGB { return m.pal[m.Index(c)] }

func (m *Mapper) cellKey(c RGB) int {
	return int(c.R>>m.shift)<<(2*mapperBits) | int(c.G>>m.shift)<<mapperBits | int(c.B>>m.shift)
}

// cellCentre snaps c to the middle of its lookup-table cell, so every
// colour in the cell resolves to the same, cell-representative answer.
func (m *Mapper) cellCentre(c RGB) RGB {
	half := uint8(1) << (m.shift - 1)
	return RGB{
		R: (c.R >> m.shift << m.shift) | half,
		G: (c.G >> m.shift << m.shift) | half,
		B: (c.B >> m.shift << m.shift) | half,
	}
}

func (m *Mapper) nearest(c RGB) int {
	best, bestD := 0, 0.0
	if m.space == SpaceOKLab {
		cl := colorspace.RGBToOKLab(c.R, c.G, c.B)
		for i, pl := range m.labs {
			dl, da, db := cl.L-pl.L, cl.A-pl.A, cl.B-pl.B
			d := dl*dl + da*da + db*db
			if i == 0 || d < bestD {
				best, bestD = i, d
			}
		}
		return best
	}
	for i, p := range m.pal {
		dr := float64(c.R) - float64(p.R)
		dg := float64(c.G) - float64(p.G)
		db := float64(c.B) - float64(p.B)
		d := dr*dr + dg*dg + db*db
		if i == 0 || d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

// Nearest returns the index of the entry in pal closest to c under space,
// computed exactly (no lookup-table approximation). It is the right tool
// for one-off lookups; for a whole image use a Mapper.
func Nearest(c RGB, pal []RGB, space Space) int {
	return (&Mapper{pal: pal, space: space, labs: labsFor(pal, space)}).nearest(c)
}

func labsFor(pal []RGB, space Space) []colorspace.OKLab {
	if space != SpaceOKLab {
		return nil
	}
	out := make([]colorspace.OKLab, len(pal))
	for i, p := range pal {
		out[i] = colorspace.RGBToOKLab(p.R, p.G, p.B)
	}
	return out
}

type box struct{ pixels []RGB }

func chanAt(c RGB, axis int) uint8 {
	switch axis {
	case 0:
		return c.R
	case 1:
		return c.G
	default:
		return c.B
	}
}

// widestAxis reports which channel (0=R,1=G,2=B) has the largest range in
// the box and the size of that range.
func (b box) widestAxis() (axis, span int) {
	lo := [3]uint8{255, 255, 255}
	hi := [3]uint8{0, 0, 0}
	for _, p := range b.pixels {
		for a := 0; a < 3; a++ {
			v := chanAt(p, a)
			if v < lo[a] {
				lo[a] = v
			}
			if v > hi[a] {
				hi[a] = v
			}
		}
	}
	for a := 0; a < 3; a++ {
		if d := int(hi[a]) - int(lo[a]); d > span {
			axis, span = a, d
		}
	}
	return axis, span
}

func (b box) split() (box, box) {
	axis, _ := b.widestAxis()
	// slices.SortStableFunc (generic, no reflection) instead of
	// sort.SliceStable: this runs on every box at every split, and
	// sort.Slice's reflect.Swapper dominated MedianCut's cost.
	slices.SortStableFunc(b.pixels, func(x, y RGB) int {
		return int(chanAt(x, axis)) - int(chanAt(y, axis))
	})
	mid := len(b.pixels) / 2
	lo := append([]RGB(nil), b.pixels[:mid]...)
	hi := append([]RGB(nil), b.pixels[mid:]...)
	return box{lo}, box{hi}
}

func (b box) mean() RGB {
	var sr, sg, sb uint64
	for _, p := range b.pixels {
		sr += uint64(p.R)
		sg += uint64(p.G)
		sb += uint64(p.B)
	}
	n := uint64(len(b.pixels))
	return RGB{uint8(sr / n), uint8(sg / n), uint8(sb / n)}
}

// samplePixels collects RGB values from img, stride-sampling down to
// maxSamples for large images. Fully transparent pixels are skipped.
func samplePixels(img *image.RGBA) []RGB {
	return samplePixelsAlpha(img, 0)
}

// samplePixelsAlpha is samplePixels with an explicit alpha cutoff: a pixel
// contributes only when its alpha is non-zero and >= threshold. threshold 0
// reproduces samplePixels exactly (fully transparent pixels only skipped).
func samplePixelsAlpha(img *image.RGBA, threshold uint8) []RGB {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	total := w * h
	if total == 0 {
		return nil
	}
	step := 1
	if total > maxSamples {
		step = (total + maxSamples - 1) / maxSamples
	}

	out := make([]RGB, 0, min(total, maxSamples)+1)
	idx := 0
	for y := 0; y < h; y++ {
		row := img.PixOffset(b.Min.X, b.Min.Y+y)
		for x := 0; x < w; x++ {
			if idx%step == 0 {
				i := row + x*4
				if a := img.Pix[i+3]; a != 0 && a >= threshold {
					out = append(out, RGB{img.Pix[i], img.Pix[i+1], img.Pix[i+2]})
				}
			}
			idx++
		}
	}
	return out
}
