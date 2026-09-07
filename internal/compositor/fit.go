package compositor

import (
	"image"
	"math"

	xdraw "golang.org/x/image/draw"
)

// FitMode controls how a layer source that isn't already the canvas size is
// placed into a w x h frame.
type FitMode string

const (
	FitCover   FitMode = "cover"   // scale to fill, crop overflow, centred (default)
	FitContain FitMode = "contain" // scale to fit inside, transparent letterbox
	FitStretch FitMode = "stretch" // scale both axes independently
	FitCenter  FitMode = "center"  // no scale, centred, clipped or padded
	FitTile    FitMode = "tile"    // repeat at native size to fill
)

// Fit returns a new w x h straight-alpha RGBA holding src placed per mode.
// src is not mutated. An unrecognised mode is treated as FitCover. Scaling
// uses Catmull-Rom (as internal/anim does).
func Fit(src *image.RGBA, w, h int, mode FitMode) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw == 0 || sh == 0 || w == 0 || h == 0 {
		return dst
	}

	switch mode {
	case FitStretch:
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, sb, xdraw.Src, nil)
	case FitCenter:
		ox := (w - sw) / 2
		oy := (h - sh) / 2
		xdraw.Draw(dst, image.Rect(ox, oy, ox+sw, oy+sh), src, sb.Min, xdraw.Src)
	case FitTile:
		for oy := 0; oy < h; oy += sh {
			for ox := 0; ox < w; ox += sw {
				xdraw.Draw(dst, image.Rect(ox, oy, ox+sw, oy+sh), src, sb.Min, xdraw.Src)
			}
		}
	case FitContain:
		s := math.Min(float64(w)/float64(sw), float64(h)/float64(sh))
		nw, nh := scaledDims(sw, sh, s)
		ox := (w - nw) / 2
		oy := (h - nh) / 2
		xdraw.CatmullRom.Scale(dst, image.Rect(ox, oy, ox+nw, oy+nh), src, sb, xdraw.Src, nil)
	default: // FitCover and any unknown mode
		s := math.Max(float64(w)/float64(sw), float64(h)/float64(sh))
		nw, nh := scaledDims(sw, sh, s)
		if nw < w {
			nw = w
		}
		if nh < h {
			nh = h
		}
		ox := (w - nw) / 2
		oy := (h - nh) / 2
		xdraw.CatmullRom.Scale(dst, image.Rect(ox, oy, ox+nw, oy+nh), src, sb, xdraw.Src, nil)
	}
	return dst
}

func scaledDims(sw, sh int, s float64) (int, int) {
	nw := int(math.Round(float64(sw) * s))
	nh := int(math.Round(float64(sh) * s))
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	return nw, nh
}
