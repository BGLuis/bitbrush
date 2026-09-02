package filters

import (
	"image"
	"math"

	"bitbrush/internal/palette"
)

// Dither applies ordered or error-diffusion dithering to src.
//
// Params:
//
//	mode        string, "levels" (default), "palette", or "ordered"
//	serpentine  bool,   alternate scan direction per row for the diffusion
//	            modes, halves directional streaking (default true)
//
//	-- diffusion modes ("levels", "palette") --
//	kernel      string, error-diffusion stencil: "floyd-steinberg" (default),
//	            "atkinson", "stucki", "jarvis", "sierra", "burkes"
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
//
//	-- mode "ordered" --
//	matrix      string, Bayer matrix size: "2", "4" (default) or "8"
//	levels      int,  steps per channel, clamped to 2..16 (default 2)
//	grayscale   bool, threshold luma into one channel instead of each RGB
//	            channel independently (default true)
//
// The "floyd-steinberg" kernel with no "ordered" mode reproduces the classic
// error-diffusion result byte for byte.
func Dither(src *image.RGBA, p Params) (*image.RGBA, error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	serpentine := p.Bool("serpentine", true)
	switch p.String("mode", "levels") {
	case "palette":
		return ditherPalette(src, dst, w, h, serpentine, p)
	case "ordered":
		return ditherOrdered(src, dst, w, h, p)
	default:
		return ditherLevels(src, dst, w, h, serpentine, p)
	}
}

func init() { Register("dither", Dither) }

// clampLevels clamps a requested step count to the supported 2..16 range.
func clampLevels(levels int) int {
	switch {
	case levels < 2:
		return 2
	case levels > 16:
		return 16
	}
	return levels
}

func ditherLevels(src, dst *image.RGBA, w, h int, serpentine bool, p Params) (*image.RGBA, error) {
	levels := clampLevels(p.Int("levels", 2))
	quant := levelQuant(levels)
	k := diffKernelFor(p.String("kernel", "floyd-steinberg"))

	if p.Bool("grayscale", true) {
		plane, _, _ := lumaPlane(src)
		diffuse1(plane, w, h, serpentine, k, quant)
		writeGrayPlane(dst, src, plane, w, h)
		return dst, nil
	}

	// Each channel is thresholded and diffused independently, which is the
	// classic "colour Floyd–Steinberg" look (levels=2 gives an 8-colour
	// RGB cube: black, the 3 primaries, the 3 secondaries, white).
	plane := readVec3Plane(src, w, h)
	diffuse3(plane, w, h, serpentine, k, func(v vec3) vec3 {
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
	k := diffKernelFor(p.String("kernel", "floyd-steinberg"))

	pal := palette.MedianCut(src, n)
	if len(pal) == 0 {
		return cloneRGBA(src), nil // fully transparent source, nothing to dither
	}
	mapper := palette.NewMapper(pal, space)

	// The whole RGB error vector is quantised and diffused together — the
	// nearest palette entry depends on all three channels at once, unlike
	// the independent per-channel case above.
	plane := readVec3Plane(src, w, h)
	diffuse3(plane, w, h, serpentine, k, func(v vec3) vec3 {
		out := mapper.At(palette.RGB{R: clampU8(v[0]), G: clampU8(v[1]), B: clampU8(v[2])})
		return vec3{float64(out.R), float64(out.G), float64(out.B)}
	})
	writeVec3Plane(dst, src, plane, w, h)
	return dst, nil
}

// ditherOrdered thresholds each pixel against a normalised Bayer matrix. No
// error is carried between pixels, so the result is position-deterministic
// and free of the directional trails diffusion leaves.
func ditherOrdered(src, dst *image.RGBA, w, h int, p Params) (*image.RGBA, error) {
	levels := clampLevels(p.Int("levels", 2))
	quant := levelQuant(levels)
	n := bayerSize(p.String("matrix", "4"))
	m := bayerMatrix(n)
	// Spread one quantisation step across the matrix's [-0.5, 0.5) range.
	amp := 255.0 / float64(levels-1)

	if p.Bool("grayscale", true) {
		plane, _, _ := lumaPlane(src)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				i := y*w + x
				plane[i] = quant(plane[i] + (m[y%n][x%n]-0.5)*amp)
			}
		}
		writeGrayPlane(dst, src, plane, w, h)
		return dst, nil
	}

	plane := readVec3Plane(src, w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			t := (m[y%n][x%n] - 0.5) * amp
			plane[i] = vec3{quant(plane[i][0] + t), quant(plane[i][1] + t), quant(plane[i][2] + t)}
		}
	}
	writeVec3Plane(dst, src, plane, w, h)
	return dst, nil
}

// bayerSize maps the matrix param to an edge length, defaulting to 4.
func bayerSize(s string) int {
	switch s {
	case "2":
		return 2
	case "8":
		return 8
	default:
		return 4
	}
}

// bayerMatrix returns an n x n Bayer threshold matrix with entries in
// [0,1), built from the 2x2 recurrence rather than a hard-coded table. n
// must be 2, 4 or 8. Each cell is centred (+0.5) inside its bucket so the
// thresholds sit symmetrically around 0.5.
func bayerMatrix(n int) [][]float64 {
	cur := [][]int{{0, 2}, {3, 1}}
	for size := 2; size < n; size *= 2 {
		next := make([][]int, size*2)
		for i := range next {
			next[i] = make([]int, size*2)
		}
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				v := cur[y][x]
				next[y][x] = 4 * v
				next[y][x+size] = 4*v + 2
				next[y+size][x] = 4*v + 3
				next[y+size][x+size] = 4*v + 1
			}
		}
		cur = next
	}
	denom := float64(n * n)
	out := make([][]float64, n)
	for y := 0; y < n; y++ {
		out[y] = make([]float64, n)
		for x := 0; x < n; x++ {
			out[y][x] = (float64(cur[y][x]) + 0.5) / denom
		}
	}
	return out
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

// diffCoef places a fraction w of the residual error at pixel offset
// (dx, dy) — dx is mirrored with the scan direction, dy always runs forward.
// w is already divided by the kernel's normalising factor.
type diffCoef struct {
	dx, dy int
	w      float64
}

// diffKernelFor returns the named error-diffusion stencil, falling back to
// Floyd–Steinberg for an unknown name. Weights are pre-divided so the
// Floyd–Steinberg path stays byte-identical to the original 7/3/5/1 code.
func diffKernelFor(name string) []diffCoef {
	switch name {
	case "atkinson":
		return []diffCoef{
			{1, 0, 1.0 / 8}, {2, 0, 1.0 / 8},
			{-1, 1, 1.0 / 8}, {0, 1, 1.0 / 8}, {1, 1, 1.0 / 8},
			{0, 2, 1.0 / 8},
		}
	case "burkes":
		return []diffCoef{
			{1, 0, 8.0 / 32}, {2, 0, 4.0 / 32},
			{-2, 1, 2.0 / 32}, {-1, 1, 4.0 / 32}, {0, 1, 8.0 / 32}, {1, 1, 4.0 / 32}, {2, 1, 2.0 / 32},
		}
	case "stucki":
		return []diffCoef{
			{1, 0, 8.0 / 42}, {2, 0, 4.0 / 42},
			{-2, 1, 2.0 / 42}, {-1, 1, 4.0 / 42}, {0, 1, 8.0 / 42}, {1, 1, 4.0 / 42}, {2, 1, 2.0 / 42},
			{-2, 2, 1.0 / 42}, {-1, 2, 2.0 / 42}, {0, 2, 4.0 / 42}, {1, 2, 2.0 / 42}, {2, 2, 1.0 / 42},
		}
	case "jarvis":
		return []diffCoef{
			{1, 0, 7.0 / 48}, {2, 0, 5.0 / 48},
			{-2, 1, 3.0 / 48}, {-1, 1, 5.0 / 48}, {0, 1, 7.0 / 48}, {1, 1, 5.0 / 48}, {2, 1, 3.0 / 48},
			{-2, 2, 1.0 / 48}, {-1, 2, 3.0 / 48}, {0, 2, 5.0 / 48}, {1, 2, 3.0 / 48}, {2, 2, 1.0 / 48},
		}
	case "sierra":
		return []diffCoef{
			{1, 0, 5.0 / 32}, {2, 0, 3.0 / 32},
			{-2, 1, 2.0 / 32}, {-1, 1, 4.0 / 32}, {0, 1, 5.0 / 32}, {1, 1, 4.0 / 32}, {2, 1, 2.0 / 32},
			{-1, 2, 2.0 / 32}, {0, 2, 3.0 / 32}, {1, 2, 2.0 / 32},
		}
	default: // floyd-steinberg
		return []diffCoef{
			{1, 0, 7.0 / 16},
			{-1, 1, 3.0 / 16}, {0, 1, 5.0 / 16}, {1, 1, 1.0 / 16},
		}
	}
}

// diffuse1 runs error diffusion over a single-channel w*h plane in place:
// each value is replaced by quant(value) and the residual is spread to
// not-yet-visited neighbours per the kernel. serpentine alternates the scan
// direction every row so diffusion doesn't always trail the same way.
func diffuse1(plane []float64, w, h int, serpentine bool, k []diffCoef, quant func(float64) float64) {
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
			for _, c := range k {
				nx := x + c.dx*step
				ny := y + c.dy
				if nx < 0 || nx >= w || ny >= h {
					continue
				}
				plane[ny*w+nx] += e * c.w
			}
		}
	}
}

// diffuse3 is diffuse1 for a plane of RGB triples: quant sees and returns
// the full triple, so it can pick (e.g.) a nearest palette colour that
// depends on all three channels together.
func diffuse3(plane []vec3, w, h int, serpentine bool, k []diffCoef, quant func(vec3) vec3) {
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
			for _, c := range k {
				nx := x + c.dx*step
				ny := y + c.dy
				if nx < 0 || nx >= w || ny >= h {
					continue
				}
				addVec(&plane[ny*w+nx], e, c.w)
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
