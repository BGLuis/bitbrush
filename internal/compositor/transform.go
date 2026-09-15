package compositor

import (
	"image"
	"math"
)

// Transform places a layer's already-fitted (canvas-sized) content within
// the canvas: move (as a fraction of canvas width/height, 0 = centred),
// scale and rotate (degrees, positive = clockwise), all about the canvas
// centre. It runs after Fit and before the layer's filter Chain, so Fit
// decides the base framing and Transform nudges it.
type Transform struct {
	OffsetX  float64 `json:"offsetX,omitempty"`  // fraction of canvas width
	OffsetY  float64 `json:"offsetY,omitempty"`  // fraction of canvas height
	Scale    float64 `json:"scale,omitempty"`    // >0, default 1 (0 is treated as 1)
	Rotation float64 `json:"rotation,omitempty"` // degrees, default 0
}

// IsIdentity reports whether t has no visible effect. A zero Scale (e.g. an
// omitted field decoding to its Go zero value) is treated as the default 1.
func (t Transform) IsIdentity() bool {
	s := t.Scale
	if s == 0 {
		s = 1
	}
	return t.OffsetX == 0 && t.OffsetY == 0 && s == 1 && mod360(t.Rotation) == 0
}

func mod360(deg float64) float64 {
	m := math.Mod(deg, 360)
	if m < 0 {
		m += 360
	}
	return m
}

// ApplyTransform returns a new w x h straight-alpha RGBA: src (already w x h)
// moved/scaled/rotated about the canvas centre. src is not mutated. Content
// that maps outside src's bounds is fully transparent — never clamped or
// smeared from the border — so a moved/rotated layer correctly reveals
// whatever is beneath it in the stack.
func ApplyTransform(src *image.RGBA, t Transform, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	if w <= 0 || h <= 0 {
		return dst
	}

	cx, cy := float64(w)/2, float64(h)/2
	theta := t.Rotation * math.Pi / 180
	cosT, sinT := math.Cos(theta), math.Sin(theta)

	scale := t.Scale
	if scale == 0 {
		scale = 1
	}
	if math.Abs(scale) < 1e-6 {
		scale = 1e-6
	}

	offX := t.OffsetX * float64(w)
	offY := t.OffsetY * float64(h)

	for dy := 0; dy < h; dy++ {
		py0 := float64(dy) + 0.5 - cy - offY
		di := dst.PixOffset(0, dy)
		for dx := 0; dx < w; dx++ {
			px0 := float64(dx) + 0.5 - cx - offX

			// Undo rotation (rotate by -theta) then undo scale, landing back
			// in src's pixel-centre coordinate space.
			sx := px0*cosT + py0*sinT
			sy := -px0*sinT + py0*cosT
			fx := sx/scale + cx - 0.5
			fy := sy/scale + cy - 0.5

			r, g, b, a := bilinearSampleTransparent(src, fx, fy)
			dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2], dst.Pix[di+3] = r, g, b, a
			di += 4
		}
	}
	return dst
}

// bilinearSampleTransparent samples src at (fx, fy) (pixel-centre
// coordinates) with 4-tap bilinear interpolation, blended in premultiplied
// alpha to avoid dark fringing at a transparent edge. Any tap outside src's
// bounds contributes fully transparent rather than clamping to the border,
// so a sample near or past src's edge fades to transparent instead of
// smearing the border colour.
func bilinearSampleTransparent(src *image.RGBA, fx, fy float64) (r, g, b, a uint8) {
	sb := src.Bounds()
	x0 := math.Floor(fx)
	y0 := math.Floor(fy)
	tx := fx - x0
	ty := fy - y0
	ix0, iy0 := int(x0), int(y0)

	r00, g00, b00, a00 := tapPremultiplied(src, sb, ix0, iy0)
	r10, g10, b10, a10 := tapPremultiplied(src, sb, ix0+1, iy0)
	r01, g01, b01, a01 := tapPremultiplied(src, sb, ix0, iy0+1)
	r11, g11, b11, a11 := tapPremultiplied(src, sb, ix0+1, iy0+1)

	rp := bilerp(r00, r10, r01, r11, tx, ty)
	gp := bilerp(g00, g10, g01, g11, tx, ty)
	bp := bilerp(b00, b10, b01, b11, tx, ty)
	ap := bilerp(a00, a10, a01, a11, tx, ty)

	if ap <= 1e-6 {
		return 0, 0, 0, 0
	}
	return to8(clamp01(rp / ap)), to8(clamp01(gp / ap)), to8(clamp01(bp / ap)), to8(clamp01(ap))
}

// tapPremultiplied returns one sample's premultiplied RGB + alpha in [0,1],
// or all zero (transparent) when (x,y) falls outside b.
func tapPremultiplied(src *image.RGBA, b image.Rectangle, x, y int) (r, g, b_, a float64) {
	if x < 0 || x >= b.Dx() || y < 0 || y >= b.Dy() {
		return 0, 0, 0, 0
	}
	o := src.PixOffset(b.Min.X+x, b.Min.Y+y)
	a = float64(src.Pix[o+3]) / 255
	r = float64(src.Pix[o]) / 255 * a
	g = float64(src.Pix[o+1]) / 255 * a
	b_ = float64(src.Pix[o+2]) / 255 * a
	return
}

func bilerp(v00, v10, v01, v11, tx, ty float64) float64 {
	top := v00 + (v10-v00)*tx
	bottom := v01 + (v11-v01)*tx
	return top + (bottom-top)*ty
}
