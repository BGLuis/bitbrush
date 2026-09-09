package filters

import (
	"image"
	"image/color"
	"math"
)

// Voxel rebuilds src as a field of isometric cubes, one per cell, each
// shaded per face (bright top, darker left and right) with an optional
// outline and an optional luma-driven height.
//
// Params:
//
//	cell            int,    cell edge px, default 20 (clamped >= 2)
//	topShade        float,  0..2 multiplier for the top face (default 1.0)
//	leftShade       float,  0..2 multiplier for the left face (default 0.75)
//	rightShade      float,  0..2 multiplier for the right face (default 0.55)
//	outline         float,  edge stroke width px, 0 disables (default 1)
//	outlineColor    string, hex outline colour (default "#0a0a0a")
//	heightFromLuma  bool,   taller cube where the cell is brighter (default false)
//	lightFlip       bool,   swap the left/right face shades (default false)
//	background      string, hex fill behind the cubes (default "#000000")
func Voxel(src *image.RGBA, p Params) (*image.RGBA, error) {
	cell := clampInt(p.Int("cell", 20), 2, 1<<20)
	topShade := clampF(p.Float("topShade", 1.0), 0, 2)
	leftShade := clampF(p.Float("leftShade", 0.75), 0, 2)
	rightShade := clampF(p.Float("rightShade", 0.55), 0, 2)
	outline := clampF(p.Float("outline", 1), 0, 32)
	outlineCol := hexOr(p.String("outlineColor", "#0a0a0a"), color.RGBA{R: 10, G: 10, B: 10, A: 255})
	heightFromLuma := p.Bool("heightFromLuma", false)
	lightFlip := p.Bool("lightFlip", false)
	bg := hexOr(p.String("background", "#000000"), color.RGBA{A: 255})

	if lightFlip {
		leftShade, rightShade = rightShade, leftShade
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	// Fill the whole frame with the background; tall cubes then poke up into
	// it, and lower (nearer) rows overdraw upper ones for correct occlusion.
	for i := 0; i+3 < len(dst.Pix); i += 4 {
		dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = bg.R, bg.G, bg.B, 255
	}

	s := float64(cell)
	ax, ay, az := s/2, s/4, s/2

	// Grid indices, not pixel offsets: a cube's screen position is a single
	// continuous function of (col+X, row+Y) across the whole grid, not just
	// its own cell. Two neighbouring cubes then always share the exact same
	// projected edge (proven by construction — the shared corner is the same
	// (col,row) pair evaluated from either side), so the isometric field
	// tiles with no background bleeding through at the seams. Positioning
	// each cube from its own cell centre independently (as before) breaks
	// that: adjacent hexagonal cube silhouettes only touch at single points,
	// leaving a diamond lattice of background gaps between every cube.
	nCols := (w + cell - 1) / cell
	nRows := (h + cell - 1) / cell
	ox0 := float64(w) / 2
	oy0 := float64(h)/2 - float64(nCols+nRows)*ay/2

	shadeCol := func(c color.RGBA, k float64) color.RGBA {
		return color.RGBA{R: clampU8(float64(c.R) * k), G: clampU8(float64(c.G) * k), B: clampU8(float64(c.B) * k), A: 255}
	}

	for cy := 0; cy < h; cy += cell {
		for cx := 0; cx < w; cx += cell {
			cw := min(cell, w-cx)
			ch := min(cell, h-cy)
			avg := blockAverage(src, b.Min.X+cx, b.Min.Y+cy, cw, ch)
			base := color.RGBA{R: avg.R, G: avg.G, B: avg.B, A: 255}

			zTop := 1.0
			if heightFromLuma {
				zTop = 1 + luma(base.R, base.G, base.B)/255*1.5
			}

			col, row := float64(cx/cell), float64(cy/cell)
			// proj maps cube coords (X,Y in [0,1], Z in [0,zTop]) to screen,
			// via the grid-wide column/row this cube sits at.
			proj := func(X, Y, Z float64) [2]float64 {
				totalX, totalY := col+X, row+Y
				return [2]float64{
					ox0 + (totalX-totalY)*ax,
					oy0 + (totalX+totalY)*ay - Z*az,
				}
			}

			// Back faces first, then the top.
			left := [4][2]float64{proj(0, 0, 0), proj(0, 1, 0), proj(0, 1, zTop), proj(0, 0, zTop)}
			right := [4][2]float64{proj(1, 0, 0), proj(1, 1, 0), proj(1, 1, zTop), proj(1, 0, zTop)}
			top := [4][2]float64{proj(0, 0, zTop), proj(1, 0, zTop), proj(1, 1, zTop), proj(0, 1, zTop)}

			fillQuad(dst, src, left, shadeCol(base, leftShade))
			fillQuad(dst, src, right, shadeCol(base, rightShade))
			fillQuad(dst, src, top, shadeCol(base, topShade))

			if outline > 0 {
				for _, f := range [3][4][2]float64{left, right, top} {
					for e := 0; e < 4; e++ {
						a, c := f[e], f[(e+1)%4]
						strokeSeg(dst, a[0], a[1], c[0], c[1], outline/2+0.5, outlineCol)
					}
				}
			}
		}
	}
	return dst, nil
}

func init() { Register("voxel", Voxel) }

// fillQuad scanline-fills the convex quad q with col, clipped to dst, taking
// alpha from src pixel-for-pixel.
func fillQuad(dst, src *image.RGBA, q [4][2]float64, col color.RGBA) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	minY, maxY := q[0][1], q[0][1]
	for _, pt := range q {
		minY = math.Min(minY, pt[1])
		maxY = math.Max(maxY, pt[1])
	}
	y0 := clampInt(int(math.Floor(minY)), 0, h)
	y1 := clampInt(int(math.Ceil(maxY)), 0, h)
	for y := y0; y < y1; y++ {
		yc := float64(y) + 0.5
		xl, xr := math.Inf(1), math.Inf(-1)
		hit := false
		for e := 0; e < 4; e++ {
			ax, ay := q[e][0], q[e][1]
			bx, by := q[(e+1)%4][0], q[(e+1)%4][1]
			if (ay <= yc) == (by <= yc) {
				continue
			}
			t := (yc - ay) / (by - ay)
			x := ax + t*(bx-ax)
			xl = math.Min(xl, x)
			xr = math.Max(xr, x)
			hit = true
		}
		if !hit {
			continue
		}
		x0 := clampInt(int(math.Ceil(xl-0.5)), 0, w)
		x1 := clampInt(int(math.Floor(xr-0.5))+1, 0, w)
		for x := x0; x < x1; x++ {
			di := dst.PixOffset(x, y)
			si := src.PixOffset(b.Min.X+x, b.Min.Y+y)
			dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2] = col.R, col.G, col.B
			dst.Pix[di+3] = src.Pix[si+3]
		}
	}
}

// strokeSeg draws the segment (ax,ay)-(bx,by) into dst with half-width hw in
// col, clipped to the image.
func strokeSeg(dst *image.RGBA, ax, ay, bx, by, hw float64, col color.RGBA) {
	b := dst.Bounds()
	w, h := b.Dx(), b.Dy()
	minX := clampInt(int(math.Floor(math.Min(ax, bx)-hw)), 0, w)
	maxX := clampInt(int(math.Ceil(math.Max(ax, bx)+hw)), 0, w)
	minY := clampInt(int(math.Floor(math.Min(ay, by)-hw)), 0, h)
	maxY := clampInt(int(math.Ceil(math.Max(ay, by)+hw)), 0, h)
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			if distToSeg(float64(x)+0.5, float64(y)+0.5, ax, ay, bx, by) <= hw {
				o := dst.PixOffset(x, y)
				dst.Pix[o], dst.Pix[o+1], dst.Pix[o+2] = col.R, col.G, col.B
			}
		}
	}
}
