package filters

import (
	"image"
	"image/color"
)

// blockAverage returns the mean RGBA of the rectangle
// [x0, x0+w) x [y0, y0+h) in src, clipped to src.Bounds().
// Averaging is done in straight (gamma-encoded) sRGB space, which is
// what the "8-bit" look wants and is cheap.
func blockAverage(src *image.RGBA, x0, y0, w, h int) color.RGBA {
	b := src.Bounds()
	x1, y1 := x0+w, y0+h
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if y0 < b.Min.Y {
		y0 = b.Min.Y
	}
	if x1 > b.Max.X {
		x1 = b.Max.X
	}
	if y1 > b.Max.Y {
		y1 = b.Max.Y
	}
	if x1 <= x0 || y1 <= y0 {
		return color.RGBA{}
	}

	var sr, sg, sb, sa uint64
	for y := y0; y < y1; y++ {
		i := src.PixOffset(x0, y)
		for x := x0; x < x1; x++ {
			sr += uint64(src.Pix[i])
			sg += uint64(src.Pix[i+1])
			sb += uint64(src.Pix[i+2])
			sa += uint64(src.Pix[i+3])
			i += 4
		}
	}
	n := uint64((x1 - x0) * (y1 - y0))
	return color.RGBA{
		R: uint8(sr / n),
		G: uint8(sg / n),
		B: uint8(sb / n),
		A: uint8(sa / n),
	}
}

// clampInt saturates v into [lo,hi].
func clampInt(v, lo, hi int) int {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	default:
		return v
	}
}

// luma is the Rec.709 luminance of a gamma-encoded sRGB pixel, in [0,255].
// It is the conventional grayscale for edge detection and dithering; it is
// not linear-light luminance.
func luma(r, g, b uint8) float64 {
	return 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)
}

// clampU8 rounds v to the nearest byte, saturating outside [0,255].
func clampU8(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 255:
		return 255
	default:
		return uint8(v + 0.5)
	}
}

// lumaPlane returns a row-major w*h plane of luma values for src, indexed
// [y*w+x] with the origin normalised to (0,0).
func lumaPlane(src *image.RGBA) (plane []float64, w, h int) {
	b := src.Bounds()
	w, h = b.Dx(), b.Dy()
	plane = make([]float64, w*h)
	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		for x := 0; x < w; x++ {
			plane[y*w+x] = luma(src.Pix[si], src.Pix[si+1], src.Pix[si+2])
			si += 4
		}
	}
	return plane, w, h
}

// convolve3x3 applies kernel k (row-major, k[4] is the centre tap) to a
// w*h plane, replicating the border, and returns a new plane.
func convolve3x3(plane []float64, w, h int, k [9]float64) []float64 {
	out := make([]float64, len(plane))
	at := func(x, y int) float64 {
		if x < 0 {
			x = 0
		} else if x >= w {
			x = w - 1
		}
		if y < 0 {
			y = 0
		} else if y >= h {
			y = h - 1
		}
		return plane[y*w+x]
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out[y*w+x] = k[0]*at(x-1, y-1) + k[1]*at(x, y-1) + k[2]*at(x+1, y-1) +
				k[3]*at(x-1, y) + k[4]*at(x, y) + k[5]*at(x+1, y) +
				k[6]*at(x-1, y+1) + k[7]*at(x, y+1) + k[8]*at(x+1, y+1)
		}
	}
	return out
}
