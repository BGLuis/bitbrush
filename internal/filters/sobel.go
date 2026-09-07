package filters

import (
	"image"
	"math"
)

// sobelMaxMag is the largest gradient magnitude the operator can produce
// for luma in [0,255]: |Gx|max = |Gy|max = 4*255, so mag <= sqrt(2)*4*255.
var sobelMaxMag = math.Sqrt2 * 4 * 255

// Sobel runs the 3x3 Sobel operator over src's luma and writes the gradient
// magnitude as a grayscale image. Alpha is carried through from src.
//
// Params:
//
//	normalize  bool,  scale so the theoretical max maps to 255 (default true)
//	threshold  float, >0 binarises the result at this value on the output
//	           scale (0..255 after normalize); 0 keeps the continuous ramp
//	invert     bool,  output 255-v (bright background, dark edges)
func Sobel(src *image.RGBA, p Params) (*image.RGBA, error) {
	normalize := p.Bool("normalize", true)
	invert := p.Bool("invert", false)
	threshold := p.Float("threshold", 0)

	plane, w, h := lumaPlane(src)
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}
	b := src.Bounds()

	// Squared cut in raw-magnitude space, so the thresholded path never calls
	// math.Sqrt. mag_out >= threshold  <=>  mag_raw^2 >= thr2.
	var thr2 float64
	if threshold > 0 {
		t := threshold
		if normalize {
			t = threshold * sobelMaxMag / 255
		}
		thr2 = t * t
	}

	resolve := func(gx, gy float64) uint8 {
		var v uint8
		if threshold > 0 {
			if gx*gx+gy*gy >= thr2 {
				v = 255
			}
		} else {
			mag := math.Sqrt(gx*gx + gy*gy)
			if normalize {
				mag = mag * 255 / sobelMaxMag
			}
			v = clampU8(mag)
		}
		if invert {
			v = 255 - v
		}
		return v
	}
	put := func(x, y int, v uint8) {
		si := src.PixOffset(b.Min.X+x, b.Min.Y+y)
		di := dst.PixOffset(x, y)
		dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2], dst.Pix[di+3] = v, v, v, src.Pix[si+3]
	}

	// Interior: direct plane indexing, no per-tap bounds checks.
	for y := 1; y < h-1; y++ {
		r0, r1, r2 := (y-1)*w, y*w, (y+1)*w
		for x := 1; x < w-1; x++ {
			tl, tc, tr := plane[r0+x-1], plane[r0+x], plane[r0+x+1]
			ml, mr := plane[r1+x-1], plane[r1+x+1]
			bl, bc, br := plane[r2+x-1], plane[r2+x], plane[r2+x+1]
			gx := (tr - tl) + 2*(mr-ml) + (br - bl)
			gy := (bl - tl) + 2*(bc-tc) + (br - tr)
			put(x, y, resolve(gx, gy))
		}
	}

	// Border ring: replicate edges.
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
	edge := func(x, y int) {
		tl, tc, tr := at(x-1, y-1), at(x, y-1), at(x+1, y-1)
		ml, mr := at(x-1, y), at(x+1, y)
		bl, bc, br := at(x-1, y+1), at(x, y+1), at(x+1, y+1)
		gx := (tr - tl) + 2*(mr-ml) + (br - bl)
		gy := (bl - tl) + 2*(bc-tc) + (br - tr)
		put(x, y, resolve(gx, gy))
	}
	for x := 0; x < w; x++ {
		edge(x, 0)
		if h > 1 {
			edge(x, h-1)
		}
	}
	for y := 1; y < h-1; y++ {
		edge(0, y)
		if w > 1 {
			edge(w-1, y)
		}
	}
	return dst, nil
}

func init() { Register("sobel", Sobel) }
