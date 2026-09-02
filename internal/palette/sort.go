package palette

import (
	"image"
	"slices"

	"bitbrush/internal/colorspace"
)

// SortKey selects the ordering applied to a palette.
type SortKey int

const (
	// SortNone leaves palette order untouched.
	SortNone SortKey = iota
	// SortLuma orders by Rec.601 luma, darkest first.
	SortLuma
	// SortHue orders by HSL hue angle, ascending from 0 degrees.
	SortHue
	// SortPopulation orders by how many image pixels map to each entry,
	// most-used first. Requires the source image.
	SortPopulation
)

// SortPalette returns a new slice holding pal's colours in the order given
// by key; pal itself is not modified. SortPopulation needs img to weight
// entries by nearest-pixel count and returns an unsorted copy when img is
// nil; the other keys ignore img.
func SortPalette(pal []RGB, key SortKey, img *image.RGBA) []RGB {
	out := append([]RGB(nil), pal...)
	switch key {
	case SortLuma:
		slices.SortStableFunc(out, func(a, b RGB) int {
			return cmpFloat(luma(a), luma(b))
		})
	case SortHue:
		slices.SortStableFunc(out, func(a, b RGB) int {
			return cmpFloat(hueOf(a), hueOf(b))
		})
	case SortPopulation:
		if img == nil || len(out) < 2 {
			return out
		}
		counts := populationCounts(pal, img)
		slices.SortStableFunc(out, func(a, b RGB) int {
			return counts[b] - counts[a] // descending
		})
	}
	return out
}

// populationCounts tallies, for each palette colour, how many sampled
// pixels of img are nearest to it (sRGB metric).
func populationCounts(pal []RGB, img *image.RGBA) map[RGB]int {
	counts := make(map[RGB]int, len(pal))
	if len(pal) == 0 {
		return counts
	}
	m := NewMapper(pal, SpaceSRGB)
	for _, px := range samplePixels(img) {
		counts[pal[m.Index(px)]]++
	}
	return counts
}

// luma is the Rec.601 luminance of c, in [0,255].
func luma(c RGB) float64 {
	return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
}

// hueOf is the HSL hue of c in degrees [0,360).
func hueOf(c RGB) float64 {
	return colorspace.RGBToHSL(c.R, c.G, c.B).H
}

func cmpFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
