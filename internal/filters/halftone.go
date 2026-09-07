package filters

import (
	"image"
	"math"
)

// Halftone reproduces an AM (amplitude-modulated) halftone screen — the
// "newspaper print" look. Tone is rendered as a grid of variable-sized dots
// sitting on a rotated screen; darker input grows the dot, lighter shrinks it.
// It is fully analytic and deterministic: identical params yield identical Pix.
//
// Params:
//
//	cellSize  int,    dot pitch in pixels, clamped to 2..64 (default 6)
//	angleDeg  float,  screen rotation in degrees (default 45) — "mono" only;
//	          "cmyk" / "rgb" use the classic per-channel screen angles
//	shape     string, "circle" (default) | "square" | "diamond" | "line"
//	channels  string, "mono" (default) | "cmyk" | "rgb"
//	gamma     float,  tone curve applied to dot coverage, clamped 0.2..3
//	          (default 1)
//	ink       string, hex ink colour, "mono" only (default "#000000")
//	paper     string, hex paper colour, "mono" only (default "#ffffff")
//	invert    bool,   swap which end of the tone range grows the dot
//	          (default false)
//
// Every output pixel is supersampled 3x3 for smooth dot edges. Source alpha
// is copied through untouched.
func Halftone(src *image.RGBA, p Params) (*image.RGBA, error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	cell := clampInt(p.Int("cellSize", 6), 2, 64)
	angle := p.Float("angleDeg", 45)
	shape := p.String("shape", "circle")
	channels := p.String("channels", "mono")
	gamma := p.Float("gamma", 1.0)
	switch {
	case gamma < 0.2:
		gamma = 0.2
	case gamma > 3:
		gamma = 3
	}
	invert := p.Bool("invert", false)

	switch channels {
	case "cmyk":
		halftoneCMYK(src, dst, w, h, cell, gamma, invert, shape)
	case "rgb":
		halftoneRGB(src, dst, w, h, cell, gamma, invert, shape)
	default:
		halftoneMono(src, dst, w, h, cell, angle, gamma, invert, shape, p)
	}
	return dst, nil
}

func init() { Register("halftone", Halftone) }

// halftoneMono screens the source luma with a single rotated dot grid and
// composites ink over paper.
func halftoneMono(src, dst *image.RGBA, w, h, cell int, angle, gamma float64, invert bool, shape string, p Params) {
	lp, _, _ := lumaPlane(src)
	// darkIsInk: low luma -> large dot. The screen normalises luma (0..255)
	// to [0,1] itself, so the intermediate divided plane is gone.
	frac := halftoneScreen(lp, w, h, angle, cell, gamma, 255, true, invert, shape)

	ink, ok := parseHexColor(p.String("ink", "#000000"))
	if !ok {
		ink, _ = parseHexColor("#000000")
	}
	paper, ok := parseHexColor(p.String("paper", "#ffffff"))
	if !ok {
		paper, _ = parseHexColor("#ffffff")
	}

	for y := 0; y < h; y++ {
		si := src.PixOffset(src.Bounds().Min.X, src.Bounds().Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			f := frac[y*w+x]
			dst.Pix[di] = clampU8(float64(paper.R)*(1-f) + float64(ink.R)*f)
			dst.Pix[di+1] = clampU8(float64(paper.G)*(1-f) + float64(ink.G)*f)
			dst.Pix[di+2] = clampU8(float64(paper.B)*(1-f) + float64(ink.B)*f)
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			di += 4
		}
	}
}

// halftoneCMYK runs four screens (C 15°, M 75°, Y 0°, K 45°) and
// multiply-composites the ink layers over white.
func halftoneCMYK(src, dst *image.RGBA, w, h, cell int, gamma float64, invert bool, shape string) {
	n := w * h
	cP := make([]float64, n)
	mP := make([]float64, n)
	yP := make([]float64, n)
	kP := make([]float64, n)
	for y := 0; y < h; y++ {
		si := src.PixOffset(src.Bounds().Min.X, src.Bounds().Min.Y+y)
		for x := 0; x < w; x++ {
			r := float64(src.Pix[si]) / 255
			g := float64(src.Pix[si+1]) / 255
			bl := float64(src.Pix[si+2]) / 255
			k := 1 - math.Max(r, math.Max(g, bl))
			var c, m, yy float64
			if k < 1 {
				c = (1 - r - k) / (1 - k)
				m = (1 - g - k) / (1 - k)
				yy = (1 - bl - k) / (1 - k)
			}
			idx := y*w + x
			cP[idx], mP[idx], yP[idx], kP[idx] = c, m, yy, k
			si += 4
		}
	}

	// darkIsInk false: a larger channel value grows the dot.
	fc := halftoneScreen(cP, w, h, 15, cell, gamma, 1, false, invert, shape)
	fm := halftoneScreen(mP, w, h, 75, cell, gamma, 1, false, invert, shape)
	fy := halftoneScreen(yP, w, h, 0, cell, gamma, 1, false, invert, shape)
	fk := halftoneScreen(kP, w, h, 45, cell, gamma, 1, false, invert, shape)

	for y := 0; y < h; y++ {
		si := src.PixOffset(src.Bounds().Min.X, src.Bounds().Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			idx := y*w + x
			cd, md, yd, kd := fc[idx], fm[idx], fy[idx], fk[idx]
			rr := (1 - cd) * (1 - kd)
			gg := (1 - md) * (1 - kd)
			bb := (1 - yd) * (1 - kd)
			dst.Pix[di] = clampU8(rr * 255)
			dst.Pix[di+1] = clampU8(gg * 255)
			dst.Pix[di+2] = clampU8(bb * 255)
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			di += 4
		}
	}
}

// halftoneRGB runs three screens (R 15°, G 75°, B 0°) and adds the ink
// layers over black.
func halftoneRGB(src, dst *image.RGBA, w, h, cell int, gamma float64, invert bool, shape string) {
	n := w * h
	rP := make([]float64, n)
	gP := make([]float64, n)
	bP := make([]float64, n)
	for y := 0; y < h; y++ {
		si := src.PixOffset(src.Bounds().Min.X, src.Bounds().Min.Y+y)
		for x := 0; x < w; x++ {
			idx := y*w + x
			rP[idx] = float64(src.Pix[si]) / 255
			gP[idx] = float64(src.Pix[si+1]) / 255
			bP[idx] = float64(src.Pix[si+2]) / 255
			si += 4
		}
	}

	fr := halftoneScreen(rP, w, h, 15, cell, gamma, 1, false, invert, shape)
	fg := halftoneScreen(gP, w, h, 75, cell, gamma, 1, false, invert, shape)
	fb := halftoneScreen(bP, w, h, 0, cell, gamma, 1, false, invert, shape)

	for y := 0; y < h; y++ {
		si := src.PixOffset(src.Bounds().Min.X, src.Bounds().Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			idx := y*w + x
			dst.Pix[di] = clampU8(fr[idx] * 255)
			dst.Pix[di+1] = clampU8(fg[idx] * 255)
			dst.Pix[di+2] = clampU8(fb[idx] * 255)
			dst.Pix[di+3] = src.Pix[si+3]
			si += 4
			di += 4
		}
	}
}

// halftone shape kinds, resolved from the string once per screen.
const (
	htCircle = iota
	htSquare
	htDiamond
	htLine
)

func htShapeKind(shape string) int {
	switch shape {
	case "square":
		return htSquare
	case "diamond":
		return htDiamond
	case "line":
		return htLine
	default:
		return htCircle
	}
}

// halftoneScreen renders one AM screen. plane holds tone in [0,toneNorm]
// (row-major w*h); toneNorm rescales it to [0,1] (255 for a raw luma plane, 1
// for an already-normalised channel plane). The grid is rotated by angleDeg;
// each output pixel is 3x3 supersampled and the returned plane holds the ink
// fraction in [0,1].
//
// darkIsInk inverts the tone before the coverage curve (low tone -> big dot);
// invert then flips it once more, exposing the public `invert` param.
//
// Per-cell dot geometry is precomputed once into a dense grid keyed by cell
// index (no map, no hashing) and stored as the exact scalar the shape test
// needs — for circles that is radius squared, so the supersample loop runs
// with zero math.Sqrt and zero math.Pow.
func halftoneScreen(plane []float64, w, h int, angleDeg float64, cellSize int, gamma, toneNorm float64, darkIsInk, invert bool, shape string) []float64 {
	out := make([]float64, w*h)
	cs := float64(cellSize)
	theta := angleDeg * math.Pi / 180
	cos, sin := math.Cos(theta), math.Sin(theta)
	kind := htShapeKind(shape)

	// Screen-space bbox of the image in cell units — the rotation is linear
	// so the extremes are at the four image corners. Pad by one cell for the
	// sub-pixel sample offsets.
	cxMin, cyMin := math.MaxInt, math.MaxInt
	cxMax, cyMax := math.MinInt, math.MinInt
	for _, c := range [4][2]float64{{0, 0}, {float64(w), 0}, {0, float64(h)}, {float64(w), float64(h)}} {
		cx := int(math.Floor((c[0]*cos + c[1]*sin) / cs))
		cy := int(math.Floor((-c[0]*sin + c[1]*cos) / cs))
		if cx < cxMin {
			cxMin = cx
		}
		if cx > cxMax {
			cxMax = cx
		}
		if cy < cyMin {
			cyMin = cy
		}
		if cy > cyMax {
			cyMax = cy
		}
	}
	cxMin--
	cyMin--
	cxMax++
	cyMax++
	gw := cxMax - cxMin + 1
	gh := cyMax - cyMin + 1

	// thr[cell] is the shape test threshold; a negative value means "empty
	// cell, never inked".
	thr := make([]float64, gw*gh)
	circleK := (cs * 0.72) * (cs * 0.72)
	for cy := cyMin; cy <= cyMax; cy++ {
		for cx := cxMin; cx <= cxMax; cx++ {
			scx := (float64(cx) + 0.5) * cs
			scy := (float64(cy) + 0.5) * cs
			icx := scx*cos - scy*sin
			icy := scx*sin + scy*cos
			tone := avgPlane(plane, w, h,
				int(math.Round(icx-cs/2)), int(math.Round(icy-cs/2)), cellSize) / toneNorm

			base := tone
			if darkIsInk {
				base = 1 - base
			}
			if invert {
				base = 1 - base
			}
			if base < 0 {
				base = 0
			} else if base > 1 {
				base = 1
			}
			cov := math.Pow(base, gamma)

			var t float64
			switch {
			case cov <= 0:
				t = -1
			case kind == htSquare:
				t = math.Sqrt(cov) * cs * 0.51
			case kind == htDiamond:
				t = math.Sqrt(cov) * cs * 0.9
			case kind == htLine:
				t = cov * cs * 0.5
			default: // circle — store radius squared
				t = cov * circleK
			}
			thr[(cy-cyMin)*gw+(cx-cxMin)] = t
		}
	}

	// invCs avoids a divide per sub-pixel; bigK lets int() do a floor without
	// math.Floor (coords stay well under bigK in magnitude).
	invCs := 1 / cs
	const bigK = 1 << 20

	counts := make([]int, w) // sub-pixel hit tally for the current row, reused
	for y := 0; y < h; y++ {
		for i := range counts {
			counts[i] = 0
		}
		for sy := 0; sy < 3; sy++ {
			py := float64(y) + (float64(sy)+0.5)/3
			for sx := 0; sx < 3; sx++ {
				px0 := (float64(sx) + 0.5) / 3
				// Screen-space coords at x=0 for this sub-pixel lane; step by
				// (cos, -sin) per x instead of recomputing the rotation.
				rx := px0*cos + py*sin
				ry := -px0*sin + py*cos
				for x := 0; x < w; x++ {
					cx := int(rx*invCs+bigK) - bigK
					cy := int(ry*invCs+bigK) - bigK
					if cx < cxMin {
						cx = cxMin
					} else if cx > cxMax {
						cx = cxMax
					}
					if cy < cyMin {
						cy = cyMin
					} else if cy > cyMax {
						cy = cyMax
					}
					t := thr[(cy-cyMin)*gw+(cx-cxMin)]
					if t >= 0 {
						dx := rx - (float64(cx)+0.5)*cs
						dy := ry - (float64(cy)+0.5)*cs
						if dx < 0 {
							dx = -dx
						}
						if dy < 0 {
							dy = -dy
						}
						var hit bool
						switch kind {
						case htSquare:
							hit = dx <= t && dy <= t
						case htDiamond:
							hit = dx+dy <= t
						case htLine:
							hit = dy <= t
						default: // circle
							hit = dx*dx+dy*dy <= t
						}
						if hit {
							counts[x]++
						}
					}
					rx += cos
					ry -= sin
				}
			}
		}
		row := y * w
		for x := 0; x < w; x++ {
			out[row+x] = float64(counts[x]) / 9
		}
	}
	return out
}

// avgPlane returns the mean of plane over [x0,x0+size) x [y0,y0+size),
// clipped to the w*h grid. An empty rectangle averages to 0.
func avgPlane(plane []float64, w, h, x0, y0, size int) float64 {
	x1, y1 := x0+size, y0+size
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > w {
		x1 = w
	}
	if y1 > h {
		y1 = h
	}
	if x1 <= x0 || y1 <= y0 {
		return 0
	}
	var sum float64
	for y := y0; y < y1; y++ {
		row := y * w
		for x := x0; x < x1; x++ {
			sum += plane[row+x]
		}
	}
	return sum / float64((x1-x0)*(y1-y0))
}
