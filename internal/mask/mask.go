package mask

import (
	"image"
	"math"
)

// Kind identifies the geometric or luminance shape of a mask.
type Kind string

const (
	KindRect    Kind = "rect"
	KindEllipse Kind = "ellipse"
	KindLuma    Kind = "luma"
)

// Mask describes a region of interest in normalized coordinates [0, 1]
// or luminance threshold, used to restrict filter application.
type Mask struct {
	Kind      Kind    `json:"kind"`
	X         float64 `json:"x,omitempty"`         // [0, 1] Left anchor
	Y         float64 `json:"y,omitempty"`         // [0, 1] Top anchor
	W         float64 `json:"w,omitempty"`         // [0, 1] Width
	H         float64 `json:"h,omitempty"`         // [0, 1] Height
	Feather   float64 `json:"feather,omitempty"`   // [0, 1] Edge transition softness
	Threshold float64 `json:"threshold,omitempty"` // [0, 1] Luminance threshold for luma
	Invert    bool    `json:"invert,omitempty"`    // Invert mask selection
}

// Alpha computes the blending weight [0.0..1.0] for pixel (px, py)
// with original RGB values (origR, origG, origB) on a canvas of size w x h.
// 0.0 means original pixel, 1.0 means fully filtered pixel.
func (m *Mask) Alpha(px, py, w, h int, origR, origG, origB uint8) float64 {
	if m == nil {
		return 1.0
	}

	var alpha float64
	fw := float64(w)
	fh := float64(h)

	switch m.Kind {
	case KindRect:
		x0 := m.X * fw
		y0 := m.Y * fh
		x1 := (m.X + m.W) * fw
		y1 := (m.Y + m.H) * fh

		dxLeft := float64(px) - x0
		dxRight := x1 - float64(px)
		dyTop := float64(py) - y0
		dyBottom := y1 - float64(py)

		d := math.Min(math.Min(dxLeft, dxRight), math.Min(dyTop, dyBottom))
		featherPx := m.Feather * math.Min(fw, fh) * 0.5

		if featherPx <= 0 {
			if d >= 0 {
				alpha = 1.0
			} else {
				alpha = 0.0
			}
		} else {
			if d <= 0 {
				alpha = 0.0
			} else if d < featherPx {
				alpha = d / featherPx
			} else {
				alpha = 1.0
			}
		}

	case KindEllipse:
		cx := (m.X + m.W*0.5) * fw
		cy := (m.Y + m.H*0.5) * fh
		rx := (m.W * 0.5) * fw
		ry := (m.H * 0.5) * fh

		if rx <= 0 || ry <= 0 {
			alpha = 0.0
			break
		}

		nx := (float64(px) - cx) / rx
		ny := (float64(py) - cy) / ry
		dist := math.Sqrt(nx*nx + ny*ny)

		featherNorm := m.Feather
		if featherNorm >= 1.0 {
			featherNorm = 0.999
		}

		if featherNorm <= 0 {
			if dist <= 1.0 {
				alpha = 1.0
			} else {
				alpha = 0.0
			}
		} else {
			if dist <= 1.0-featherNorm {
				alpha = 1.0
			} else if dist < 1.0 {
				alpha = (1.0 - dist) / featherNorm
			} else {
				alpha = 0.0
			}
		}

	case KindLuma:
		lum := (0.2126*float64(origR) + 0.7152*float64(origG) + 0.0722*float64(origB)) / 255.0
		threshold := m.Threshold
		feather := m.Feather

		if feather <= 0 {
			if lum >= threshold {
				alpha = 1.0
			} else {
				alpha = 0.0
			}
		} else {
			if lum <= threshold {
				alpha = 0.0
			} else if lum >= threshold+feather {
				alpha = 1.0
			} else {
				alpha = (lum - threshold) / feather
			}
		}

	default:
		alpha = 1.0
	}

	if m.Invert {
		alpha = 1.0 - alpha
	}

	if alpha < 0 {
		return 0
	}
	if alpha > 1 {
		return 1
	}
	return alpha
}

// Compose blends original and filtered images according to mask m.
// If m is nil, filtered is returned directly.
func Compose(original, filtered *image.RGBA, m *Mask) *image.RGBA {
	if m == nil {
		return filtered
	}

	b := original.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(b)

	for y := 0; y < h; y++ {
		so := original.PixOffset(b.Min.X, b.Min.Y+y)
		sf := filtered.PixOffset(b.Min.X, b.Min.Y+y)
		sd := dst.PixOffset(b.Min.X, b.Min.Y+y)
		for x := 0; x < w; x++ {
			or, og, ob, oa := original.Pix[so], original.Pix[so+1], original.Pix[so+2], original.Pix[so+3]
			fr, fg, fb, fa := filtered.Pix[sf], filtered.Pix[sf+1], filtered.Pix[sf+2], filtered.Pix[sf+3]

			t := m.Alpha(x, y, w, h, or, og, ob)

			dst.Pix[sd] = uint8(math.Round(float64(or)*(1.0-t) + float64(fr)*t))
			dst.Pix[sd+1] = uint8(math.Round(float64(og)*(1.0-t) + float64(fg)*t))
			dst.Pix[sd+2] = uint8(math.Round(float64(ob)*(1.0-t) + float64(fb)*t))
			dst.Pix[sd+3] = uint8(math.Round(float64(oa)*(1.0-t) + float64(fa)*t))

			so += 4
			sf += 4
			sd += 4
		}
	}

	return dst
}
