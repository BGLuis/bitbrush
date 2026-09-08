package filters

import "image"

// Blur convolves src with a separable box filter, or a near-Gaussian built
// from three box passes.
//
// Params:
//
//	radius  int,    blur radius px, default 4 (0 = identity)
//	type    string, "gaussian" (default, 3 box passes) or "box"
func Blur(src *image.RGBA, p Params) (*image.RGBA, error) {
	radius := clampInt(p.Int("radius", 4), 0, 1<<12)
	kind := p.String("type", "gaussian")

	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return newLike(src), nil
	}
	if radius == 0 {
		return cloneRGBA(src), nil
	}
	if kind == "box" {
		return boxBlur(src, radius), nil
	}
	// Three box passes approximate a Gaussian (central limit). Split the
	// radius so the combined support stays close to the requested one.
	r := max(radius/3, 1)
	out := boxBlur(src, r)
	out = boxBlur(out, r)
	out = boxBlur(out, radius-2*r)
	return out, nil
}

func init() { Register("blur", Blur) }
