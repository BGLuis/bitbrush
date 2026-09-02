package filters

import "image"

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

// newLike returns a zeroed RGBA the same size as src, origin (0,0).
func newLike(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	return image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
}
