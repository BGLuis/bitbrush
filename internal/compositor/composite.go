package compositor

import (
	"fmt"
	"image"
)

// Composite returns a new image the size of base with over blended on top
// using mode and a layer opacity in [0,1]. Both inputs are treated as
// straight-alpha 8-bit RGBA (the browser ImageData convention); neither is
// mutated. over must already match base's dimensions — call Fit first.
//
// An unrecognised mode is treated as Normal. The result is straight-alpha;
// a fully transparent output pixel is transparent black.
func Composite(base, over *image.RGBA, mode BlendMode, opacity float64) (*image.RGBA, error) {
	bb, ob := base.Bounds(), over.Bounds()
	if bb.Dx() != ob.Dx() || bb.Dy() != ob.Dy() {
		return nil, fmt.Errorf("compositor: size mismatch, base %dx%d vs over %dx%d",
			bb.Dx(), bb.Dy(), ob.Dx(), ob.Dy())
	}
	blend, ok := blendFn(mode)
	if !ok {
		blend, _ = blendFn(BlendNormal)
	}
	opacity = clamp01(opacity)

	w, h := bb.Dx(), bb.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		bi := base.PixOffset(bb.Min.X, bb.Min.Y+y)
		oi := over.PixOffset(ob.Min.X, ob.Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			br := float64(base.Pix[bi]) / 255
			bg := float64(base.Pix[bi+1]) / 255
			bl := float64(base.Pix[bi+2]) / 255
			ba := float64(base.Pix[bi+3]) / 255

			sr := float64(over.Pix[oi]) / 255
			sg := float64(over.Pix[oi+1]) / 255
			sl := float64(over.Pix[oi+2]) / 255
			sa := float64(over.Pix[oi+3]) / 255 * opacity

			ao := sa + ba*(1-sa)

			var cr, cg, cbl float64
			if ao > 0 {
				cr = compositeChannel(br, sr, ba, sa, ao, blend)
				cg = compositeChannel(bg, sg, ba, sa, ao, blend)
				cbl = compositeChannel(bl, sl, ba, sa, ao, blend)
			}

			dst.Pix[di] = to8(cr)
			dst.Pix[di+1] = to8(cg)
			dst.Pix[di+2] = to8(cbl)
			dst.Pix[di+3] = to8(ao)

			bi += 4
			oi += 4
			di += 4
		}
	}
	return dst, nil
}

// compositeChannel is the W3C source-over composite of one channel:
//
//	Co = ( αs·(1-αb)·Cs + αs·αb·B(Cb,Cs) + (1-αs)·αb·Cb ) / αo
//
// with ao > 0 guaranteed by the caller.
func compositeChannel(cb, cs, ab, as, ao float64, blend func(cb, cs float64) float64) float64 {
	co := as*(1-ab)*cs + as*ab*blend(cb, cs) + (1-as)*ab*cb
	return clamp01(co / ao)
}

// cloneRGBA returns a deep copy of src normalised to origin (0,0).
func cloneRGBA(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		copy(dst.Pix[di:di+b.Dx()*4], src.Pix[si:si+b.Dx()*4])
	}
	return dst
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func to8(v float64) uint8 {
	return uint8(clamp01(v)*255 + 0.5)
}
