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

	// Clamped access to the luma plane (border replication).
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

	b := src.Bounds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			tl, tc, tr := at(x-1, y-1), at(x, y-1), at(x+1, y-1)
			ml, mr := at(x-1, y), at(x+1, y)
			bl, bc, br := at(x-1, y+1), at(x, y+1), at(x+1, y+1)

			gx := (tr - tl) + 2*(mr-ml) + (br - bl)
			gy := (bl - tl) + 2*(bc-tc) + (br - tr)
			mag := math.Sqrt(gx*gx + gy*gy)
			if normalize {
				mag = mag * 255 / sobelMaxMag
			}

			var v uint8
			if threshold > 0 {
				if mag >= threshold {
					v = 255
				}
			} else {
				v = clampU8(mag)
			}
			if invert {
				v = 255 - v
			}

			si := src.PixOffset(b.Min.X+x, b.Min.Y+y)
			di := dst.PixOffset(x, y)
			dst.Pix[di] = v
			dst.Pix[di+1] = v
			dst.Pix[di+2] = v
			dst.Pix[di+3] = src.Pix[si+3]
		}
	}
	return dst, nil
}

func init() { Register("sobel", Sobel) }
