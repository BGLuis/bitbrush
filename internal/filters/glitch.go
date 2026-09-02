package filters

import (
	"image"
	"math/rand"
)

// Glitch simulates analogue signal corruption: horizontal "slice" bands
// wrap-displaced by a random offset, then a per-channel horizontal shift
// splits red and blue apart, with optional scanline darkening on top.
//
// All randomness is drawn from a single generator seeded from `seed`, so a
// given seed+params combination always reproduces the exact same image —
// that is what makes a glitch result shareable as a link.
//
// Params:
//
//	seed              int,   RNG seed (default 0)
//	rgbShift          int,   px offset between the R and B samples (default 4)
//	sliceCount        int,   number of horizontal bands, 0 disables slicing (default 8)
//	maxSliceShift     int,   max px wrap-offset per band (default 20)
//	sliceProbability  float, chance in [0,1] a given band is displaced at all (default 0.5)
//	scanlines         bool,  darken every other row (default false)
func Glitch(src *image.RGBA, p Params) (*image.RGBA, error) {
	seed := int64(p.Int("seed", 0))
	rgbShift := p.Int("rgbShift", 4)
	sliceCount := p.Int("sliceCount", 8)
	maxSliceShift := p.Int("maxSliceShift", 20)
	sliceProbability := p.Float("sliceProbability", 0.5)
	scanlines := p.Bool("scanlines", false)

	if sliceCount < 0 {
		sliceCount = 0
	}
	if maxSliceShift < 0 {
		maxSliceShift = 0
	}
	switch {
	case sliceProbability < 0:
		sliceProbability = 0
	case sliceProbability > 1:
		sliceProbability = 1
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	rng := rand.New(rand.NewSource(seed))
	rowShift := glitchBandShifts(h, sliceCount, maxSliceShift, sliceProbability, rng)

	// Pass 1: wrap-shift every row horizontally by its band's offset.
	mid := newLike(src)
	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := mid.PixOffset(0, y)
		dx := rowShift[y]
		for x := 0; x < w; x++ {
			sx := ((x-dx)%w + w) % w
			s, d := si+sx*4, di+x*4
			mid.Pix[d], mid.Pix[d+1], mid.Pix[d+2], mid.Pix[d+3] =
				src.Pix[s], src.Pix[s+1], src.Pix[s+2], src.Pix[s+3]
		}
	}

	// Pass 2: split R left / B right (border-clamped), then scanlines.
	for y := 0; y < h; y++ {
		mi := mid.PixOffset(0, y)
		di := dst.PixOffset(0, y)
		darken := scanlines && y%2 == 1
		for x := 0; x < w; x++ {
			rx := clampInt(x-rgbShift, 0, w-1)
			bx := clampInt(x+rgbShift, 0, w-1)
			r := mid.Pix[mi+rx*4]
			g := mid.Pix[mi+x*4+1]
			bl := mid.Pix[mi+bx*4+2]
			a := mid.Pix[mi+x*4+3]
			if darken {
				r = uint8(float64(r) * 0.82)
				g = uint8(float64(g) * 0.82)
				bl = uint8(float64(bl) * 0.82)
			}
			d := di + x*4
			dst.Pix[d], dst.Pix[d+1], dst.Pix[d+2], dst.Pix[d+3] = r, g, bl, a
		}
	}

	return dst, nil
}

func init() { Register("glitch", Glitch) }

// glitchBandShifts splits h rows into sliceCount bands and draws one
// horizontal wrap-shift per band: with probability sliceProbability the
// band's shift is uniform in [-maxShift, maxShift], otherwise 0. Every row
// in a band gets that band's shift. All rows are 0 when sliceCount <= 0 or
// maxShift <= 0.
func glitchBandShifts(h, sliceCount, maxShift int, sliceProbability float64, rng *rand.Rand) []int {
	shifts := make([]int, h)
	if h == 0 || sliceCount <= 0 || maxShift <= 0 {
		return shifts
	}
	bandHeight := (h + sliceCount - 1) / sliceCount
	for band := 0; band < sliceCount; band++ {
		dx := 0
		if rng.Float64() < sliceProbability {
			dx = rng.Intn(2*maxShift+1) - maxShift
		}
		start, end := band*bandHeight, (band+1)*bandHeight
		if end > h {
			end = h
		}
		for y := start; y < end; y++ {
			shifts[y] = dx
		}
	}
	return shifts
}
