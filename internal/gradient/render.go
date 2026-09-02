package gradient

import (
	"image"
	"math"
)

// lutSize is the resolution of the per-render colour lookup table. The
// gradient is sampled once into the table and every pixel indexes it, so
// the cost of the colour-space maths is O(lutSize) instead of O(pixels).
const lutSize = 1024

// Render rasterises the gradient as a linear gradient along angleDeg into a
// new opaque RGBA of size w x h. The angle follows the CSS convention:
// 0deg points to the top, 90deg to the right, increasing clockwise.
func (g *Gradient) Render(w, h int, angleDeg float64) *image.RGBA {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	lut := make([]RGB, lutSize)
	for i := range lut {
		lut[i] = g.At(float64(i) / float64(lutSize-1))
	}

	// CSS angle -> unit direction in screen space (y points down).
	rad := angleDeg * math.Pi / 180
	dx := math.Sin(rad)
	dy := -math.Cos(rad)

	cx := float64(w) / 2
	cy := float64(h) / 2

	// Length of the CSS gradient line for this box and angle.
	span := math.Abs(float64(w)*dx) + math.Abs(float64(h)*dy)
	if span < 1e-9 {
		span = 1
	}
	inv := 1 / span
	scale := float64(lutSize - 1)

	for y := 0; y < h; y++ {
		py := float64(y) + 0.5 - cy
		base := py * dy
		row := img.Pix[y*img.Stride:]
		for x := 0; x < w; x++ {
			px := float64(x) + 0.5 - cx
			t := (px*dx+base)*inv + 0.5
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			c := lut[int(t*scale+0.5)]
			o := x * 4
			row[o] = c.R
			row[o+1] = c.G
			row[o+2] = c.B
			row[o+3] = 255
		}
	}
	return img
}
