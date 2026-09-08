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

// smoothstep is the Hermite S-curve: 0 for e<=lo, 1 for e>=hi, a smooth
// ramp in between.
func smoothstep(lo, hi, e float64) float64 {
	if hi == lo {
		if e < lo {
			return 0
		}
		return 1
	}
	t := clampF((e-lo)/(hi-lo), 0, 1)
	return t * t * (3 - 2*t)
}

// clampF saturates v into [lo,hi].
func clampF(v, lo, hi float64) float64 {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	default:
		return v
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

// bilinearSample reads src at the fractional position (fx, fy) with
// bilinear interpolation, clamping the four taps to the image edge so an
// out-of-range coordinate returns the nearest border pixel. Coordinates
// are in the normalised (0,0)-origin space, i.e. pixel centres sit at
// integer+0.5 is NOT assumed — (fx,fy)=(0,0) is the top-left pixel.
func bilinearSample(src *image.RGBA, fx, fy float64) color.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return color.RGBA{}
	}
	if fx < 0 {
		fx = 0
	} else if fx > float64(w-1) {
		fx = float64(w - 1)
	}
	if fy < 0 {
		fy = 0
	} else if fy > float64(h-1) {
		fy = float64(h - 1)
	}
	x0 := int(fx)
	y0 := int(fy)
	x1, y1 := x0+1, y0+1
	if x1 > w-1 {
		x1 = w - 1
	}
	if y1 > h-1 {
		y1 = h - 1
	}
	tx := fx - float64(x0)
	ty := fy - float64(y0)

	at := func(x, y int) (r, g, bl, a float64) {
		i := src.PixOffset(b.Min.X+x, b.Min.Y+y)
		return float64(src.Pix[i]), float64(src.Pix[i+1]), float64(src.Pix[i+2]), float64(src.Pix[i+3])
	}
	r00, g00, b00, a00 := at(x0, y0)
	r10, g10, b10, a10 := at(x1, y0)
	r01, g01, b01, a01 := at(x0, y1)
	r11, g11, b11, a11 := at(x1, y1)

	lerp := func(p, q, t float64) float64 { return p + (q-p)*t }
	top := func(p, q float64) float64 { return lerp(p, q, tx) }
	mix := func(a, b, c, d float64) float64 { return lerp(top(a, b), top(c, d), ty) }

	return color.RGBA{
		R: clampU8(mix(r00, r10, r01, r11)),
		G: clampU8(mix(g00, g10, g01, g11)),
		B: clampU8(mix(b00, b10, b01, b11)),
		A: clampU8(mix(a00, a10, a01, a11)),
	}
}

// boxBlur returns a new RGBA that is src blurred by a separable box filter
// of the given radius (window edge 2*radius+1). RGB channels are averaged;
// alpha is carried through from src unchanged, matching the package's
// "the effect never changes opacity" convention. radius <= 0 clones src.
func boxBlur(src *image.RGBA, radius int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if radius <= 0 || w == 0 || h == 0 {
		return cloneRGBA(src)
	}

	// Horizontal pass: src -> tmp.
	tmp := image.NewRGBA(image.Rect(0, 0, w, h))
	win := float64(2*radius + 1)
	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		ti := tmp.PixOffset(0, y)
		get := func(x int) (int, int, int) {
			if x < 0 {
				x = 0
			} else if x > w-1 {
				x = w - 1
			}
			o := si + x*4
			return int(src.Pix[o]), int(src.Pix[o+1]), int(src.Pix[o+2])
		}
		var sr, sg, sb int
		for k := -radius; k <= radius; k++ {
			r, g, bl := get(k)
			sr, sg, sb = sr+r, sg+g, sb+bl
		}
		for x := 0; x < w; x++ {
			o := ti + x*4
			tmp.Pix[o] = uint8(float64(sr)/win + 0.5)
			tmp.Pix[o+1] = uint8(float64(sg)/win + 0.5)
			tmp.Pix[o+2] = uint8(float64(sb)/win + 0.5)
			tmp.Pix[o+3] = 255
			ar, ag, ab := get(x + radius + 1)
			rr, rg, rb := get(x - radius)
			sr += ar - rr
			sg += ag - rg
			sb += ab - rb
		}
	}

	// Vertical pass: tmp -> dst, then restore src alpha.
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		get := func(y int) (int, int, int) {
			if y < 0 {
				y = 0
			} else if y > h-1 {
				y = h - 1
			}
			o := tmp.PixOffset(x, y)
			return int(tmp.Pix[o]), int(tmp.Pix[o+1]), int(tmp.Pix[o+2])
		}
		var sr, sg, sb int
		for k := -radius; k <= radius; k++ {
			r, g, bl := get(k)
			sr, sg, sb = sr+r, sg+g, sb+bl
		}
		for y := 0; y < h; y++ {
			o := dst.PixOffset(x, y)
			dst.Pix[o] = uint8(float64(sr)/win + 0.5)
			dst.Pix[o+1] = uint8(float64(sg)/win + 0.5)
			dst.Pix[o+2] = uint8(float64(sb)/win + 0.5)
			dst.Pix[o+3] = src.Pix[src.PixOffset(b.Min.X+x, b.Min.Y+y)+3]
			ar, ag, ab := get(y + radius + 1)
			rr, rg, rb := get(y - radius)
			sr += ar - rr
			sg += ag - rg
			sb += ab - rb
		}
	}
	return dst
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
