package filters

import "image"

// Pixelate collapses each blockSize x blockSize cell of src to its mean
// colour, producing a coarse "8-bit" mosaic.
//
// Params:
//
//	blockSize  int, cell edge in pixels (default 8, clamped to >= 1)
func Pixelate(src *image.RGBA, p Params) (*image.RGBA, error) {
	block := p.Int("blockSize", 8)
	if block < 1 {
		block = 1
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	for by := 0; by < h; by += block {
		bh := block
		if by+bh > h {
			bh = h - by
		}
		for bx := 0; bx < w; bx += block {
			bw := block
			if bx+bw > w {
				bw = w - bx
			}

			avg := blockAverage(src, b.Min.X+bx, b.Min.Y+by, block, block)

			for y := 0; y < bh; y++ {
				i := dst.PixOffset(bx, by+y)
				for x := 0; x < bw; x++ {
					dst.Pix[i] = avg.R
					dst.Pix[i+1] = avg.G
					dst.Pix[i+2] = avg.B
					dst.Pix[i+3] = avg.A
					i += 4
				}
			}
		}
	}
	return dst, nil
}

func init() { Register("pixelate", Pixelate) }
