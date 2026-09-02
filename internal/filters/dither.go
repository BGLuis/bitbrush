package filters

import (
	"image"
	"math"

	"bitbrush/internal/palette"
)

// Dither applies Floyd–Steinberg error diffusion to src.
//
// Params:
//
//	mode        string, "levels" (default) or "palette"
//	serpentine  bool,   alternate scan direction per row, halves directional
//	            streaking (default true)
//
//	-- mode "levels" --
//	levels      int,  steps per channel, clamped to 2..16 (default 2)
//	grayscale   bool, dither luma into one channel instead of each RGB
//	            channel independently (default true)
//
//	-- mode "palette" --
//	colors      int,    median-cut palette size, clamped to 2..256 (default 8)
//	space       string, "oklab" (default) or "srgb" nearest-colour metric
//	            (same options as the Colour Quantization filter)
func Dither(src *image.RGBA, p Params) (*image.RGBA, error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	serpentine := p.Bool("serpentine", true)
	if p.String("mode", "levels") == "palette" {
		return ditherPalette(src, dst, w, h, serpentine, p)
	}
	return ditherLevels(src, dst, w, h, serpentine, p)
}

func init() { Register("dither", Dither) }

func ditherLevels(src, dst *image.RGBA, w, h int, serpentine bool, p Params) (*image.RGBA, error) {
	levels := p.Int("levels", 2)
	switch {
	case levels < 2:
		levels = 2
	case levels > 16:
		levels = 16
	}
	quant := levelQuant(levels)

	if p.Bool("grayscale", true) {
		plane, _, _ := lumaPlane(src)
		diffuse1(plane, w, h, serpentine, quant)
		writeGrayPlane(dst, src, plane, w, h)
		return dst, nil
	}

	// Each channel is thresholded and diffused independently, which is the
	// classic "colour Floyd–Steinberg" look (levels=2 gives an 8-colour
	// RGB cube: black, the 3 primaries, the 3 secondaries, white).
	plane := readVec3Plane(src, w, h)
	diffuse3(plane, w, h, serpentine, func(v vec3) vec3 {
		return vec3{quant(v[0]), quant(v[1]), quant(v[2])}
	})
	writeVec3Plane(dst, src, plane, w, h)
	return dst, nil
}

func ditherPalette(src, dst *image.RGBA, w, h int, serpentine bool, p Params) (*image.RGBA, error) {
	n := p.Int("colors", 8)
	switch {
	case n < 2:
		n = 2
	case n > 256:
		n = 256
	}
	space := palette.SpaceOKLab
	if p.String("space", "oklab") == "srgb" {
		space = palette.SpaceSRGB
	}

	pal := palette.MedianCut(src, n)
	if len(pal) == 0 {
		return cloneRGBA(src), nil // fully transparent source, nothing to dither
	}
	mapper := palette.NewMapper(pal, space)

	// The whole RGB error vector is quantised and diffused together — the
	// nearest palette entry depends on all three channels at once, unlike
	// the independent per-channel case above.
	plane := readVec3Plane(src, w, h)
	diffuse3(plane, w, h, serpentine, func(v vec3) vec3 {
		out := mapper.At(palette.RGB{R: clampU8(v[0]), G: clampU8(v[1]), B: clampU8(v[2])})
		return vec3{float64(out.R), float64(out.G), float64(out.B)}
	})
	writeVec3Plane(dst, src, plane, w, h)
	return dst, nil
}

// levelQuant returns a function mapping a channel value to the nearest of
// `levels` evenly spaced steps across [0,255]. levels must be >= 2.
func levelQuant(levels int) func(float64) float64 {
	steps := float64(levels - 1)
	return func(v float64) float64 {
		switch {
		case v < 0:
			v = 0
		case v > 255:
			v = 255
		}
		return math.Round(v/255*steps) / steps * 255
	}
}

// vec3 is a working-precision RGB triple used while diffusing error.
type vec3 = [3]float64

// diffuse1 runs Floyd–Steinberg over a single-channel w*h plane in place:
// each value is replaced by quant(value) and the residual is diffused to
// not-yet-visited neighbours. serpentine alternates the scan direction
// every row so diffusion doesn't always trail the same way.
func diffuse1(plane []float64, w, h int, serpentine bool, quant func(float64) float64) {
	for y := 0; y < h; y++ {
		ltr := !serpentine || y%2 == 0
		x, end, step := 0, w, 1
		if !ltr {
			x, end, step = w-1, -1, -1
		}
		for ; x != end; x += step {
			i := y*w + x
			old := plane[i]
			q := quant(old)
			plane[i] = q
			e := old - q
			if e == 0 {
				continue
			}
			nx := x + step
			if nx >= 0 && nx < w {
				plane[y*w+nx] += e * (7.0 / 16)
			}
			if y+1 < h {
				row := (y + 1) * w
				if bx := x - step; bx >= 0 && bx < w {
					plane[row+bx] += e * (3.0 / 16)
				}
				plane[row+x] += e * (5.0 / 16)
				if nx >= 0 && nx < w {
					plane[row+nx] += e * (1.0 / 16)
				}
			}
		}
	}
}

// diffuse3 is diffuse1 for a plane of RGB triples: quant sees and returns
// the full triple, so it can pick (e.g.) a nearest palette colour that
// depends on all three channels together.
func diffuse3(plane []vec3, w, h int, serpentine bool, quant func(vec3) vec3) {
	for y := 0; y < h; y++ {
		ltr := !serpentine || y%2 == 0
		x, end, step := 0, w, 1
		if !ltr {
			x, end, step = w-1, -1, -1
		}
		for ; x != end; x += step {
			i := y*w + x
			old := plane[i]
			q := quant(old)
			plane[i] = q
			e := vec3{old[0] - q[0], old[1] - q[1], old[2] - q[2]}
			if e == (vec3{}) {
				continue
			}
			nx := x + step
			if nx >= 0 && nx < w {
				addVec(&plane[y*w+nx], e, 7.0/16)
			}
			if y+1 < h {
				row := (y + 1) * w
				if bx := x - step; bx >= 0 && bx < w {
					addVec(&plane[row+bx], e, 3.0/16)
				}
				addVec(&plane[row+x], e, 5.0/16)
				if nx >= 0 && nx < w {
					addVec(&plane[row+nx], e, 1.0/16)
				}
			}
		}
	}
}

func addVec(p *vec3, e vec3, frac float64) {
	p[0] += e[0] * frac
	p[1] += e[1] * frac
	p[2] += e[2] * frac
}

func readVec3Plane(src *image.RGBA, w, h int) []vec3 {
	b := src.Bounds()
	plane := make([]vec3, w*h)
	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		for x := 0; x < w; x++ {
			plane[y*w+x] = vec3{float64(src.Pix[si]), float64(src.Pix[si+1]), float64(src.Pix[si+2])}
			si += 4
		}
	}
	return plane
}

func writeVec3Plane(dst, src *image.RGBA, plane []vec3, w, h int) {
	b := src.Bounds()
	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			v := plane[y*w+x]
			dst.Pix[di] = clampU8(v[0])
			dst.Pix[di+1] = clampU8(v[1])
			dst.Pix[di+2] = clampU8(v[2])
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			di += 4
		}
	}
}

func writeGrayPlane(dst, src *image.RGBA, plane []float64, w, h int) {
	b := src.Bounds()
	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			v := clampU8(plane[y*w+x])
			dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2] = v, v, v
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			di += 4
		}
	}
}
