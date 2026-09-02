package filters

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"bitbrush/internal/voronoi"
)

// Stipple renders src as a weighted Voronoi stipple drawing (Secord 2002):
// N points are seeded by rejection sampling against a density field derived
// from the image, then relaxed toward the weighted centroids of their
// Voronoi cells; each final point is drawn as a disc whose radius tracks the
// local density.
//
// Params:
//
//	points      int,   number of dots, clamped 100..20000 (default 4000)
//	iterations  int,   Lloyd relaxation passes, clamped 0..80 (default 30)
//	seed        int,   RNG seed for the initial point placement (default 0)
//	minRadius   float,  dot radius where density is 0 (default 0.6)
//	maxRadius   float,  dot radius where density is 1 (default 2.4)
//	gamma       float,  contrast on the density field, clamped 0.2..3
//	            (default 1)
//	invert      bool,   false (default): dark areas get more/larger dots;
//	            true: light areas do
//	ink         string, hex dot colour (default "#111111")
//	paper       string, hex background colour (default "#f5f2ea")
//
// Deterministic: a given seed + params always produces identical pixels.
// Source alpha is copied through per pixel.
func Stipple(src *image.RGBA, p Params) (*image.RGBA, error) {
	b := src.Bounds()
	W, H := b.Dx(), b.Dy()
	dst := newLike(src)
	if W == 0 || H == 0 {
		return dst, nil
	}

	points := clampInt(p.Int("points", 4000), 100, 20000)
	iterations := clampInt(p.Int("iterations", 30), 0, 80)
	seed := int64(p.Int("seed", 0))
	minR := p.Float("minRadius", 0.6)
	maxR := p.Float("maxRadius", 2.4)
	gamma := p.Float("gamma", 1.0)
	switch {
	case gamma < 0.2:
		gamma = 0.2
	case gamma > 3:
		gamma = 3
	}
	invert := p.Bool("invert", false)
	ink := hexOr(p.String("ink", "#111111"), color.RGBA{17, 17, 17, 255})
	paper := hexOr(p.String("paper", "#f5f2ea"), color.RGBA{245, 242, 234, 255})

	// Density field on a capped grid; results scale back to full resolution.
	const gridCap = 512
	gw, gh := W, H
	if m := max(W, H); m > gridCap {
		gw = int(math.Round(float64(W) * float64(gridCap) / float64(m)))
		gh = int(math.Round(float64(H) * float64(gridCap) / float64(m)))
	}
	gw = max(gw, 1)
	gh = max(gh, 1)

	dens := make([]float64, gw*gh)
	maxD := 0.0
	for gy := 0; gy < gh; gy++ {
		sy := clampInt(int((float64(gy)+0.5)/float64(gh)*float64(H)), 0, H-1)
		for gx := 0; gx < gw; gx++ {
			sx := clampInt(int((float64(gx)+0.5)/float64(gw)*float64(W)), 0, W-1)
			o := src.PixOffset(b.Min.X+sx, b.Min.Y+sy)
			tone := luma(src.Pix[o], src.Pix[o+1], src.Pix[o+2]) / 255
			d := 1 - tone
			if invert {
				d = tone
			}
			v := math.Pow(clamp01(d), gamma)
			dens[gy*gw+gx] = v
			if v > maxD {
				maxD = v
			}
		}
	}

	rng := rand.New(rand.NewSource(seed))
	sites := make([]voronoi.Site, 0, points)
	if maxD <= 0 {
		for len(sites) < points {
			sites = append(sites, voronoi.Site{X: rng.Float64() * float64(gw), Y: rng.Float64() * float64(gh)})
		}
	} else {
		maxAttempts := points * 400
		for att := 0; len(sites) < points && att < maxAttempts; att++ {
			x := rng.Float64() * float64(gw)
			y := rng.Float64() * float64(gh)
			if rng.Float64()*maxD <= dens[int(y)*gw+int(x)] {
				sites = append(sites, voronoi.Site{X: x, Y: y})
			}
		}
		for len(sites) < points { // top up if rejection under-filled
			sites = append(sites, voronoi.Site{X: rng.Float64() * float64(gw), Y: rng.Float64() * float64(gh)})
		}
	}

	sites = voronoi.Relax(sites, dens, gw, gh, iterations)

	// Paint paper + source alpha.
	for y := 0; y < H; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < W; x++ {
			dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2], dst.Pix[di+3] = paper.R, paper.G, paper.B, src.Pix[si+3]
			si += 4
			di += 4
		}
	}

	sxFull := float64(W) / float64(gw)
	syFull := float64(H) / float64(gh)
	for _, s := range sites {
		gx := clampInt(int(s.X), 0, gw-1)
		gy := clampInt(int(s.Y), 0, gh-1)
		rho := clamp01(dens[gy*gw+gx])
		rad := minR + (maxR-minR)*rho
		drawDisc(dst, W, H, s.X*sxFull, s.Y*syFull, rad, ink)
	}
	return dst, nil
}

func init() { Register("stipple", Stipple) }

// drawDisc alpha-composites a 1px-antialiased ink disc onto dst, leaving the
// alpha channel untouched.
func drawDisc(dst *image.RGBA, w, h int, cx, cy, r float64, ink color.RGBA) {
	if r <= 0 {
		return
	}
	x0 := clampInt(int(math.Floor(cx-r-1)), 0, w)
	x1 := clampInt(int(math.Ceil(cx+r+1)), 0, w)
	y0 := clampInt(int(math.Floor(cy-r-1)), 0, h)
	y1 := clampInt(int(math.Ceil(cy+r+1)), 0, h)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			cov := clamp01(r + 0.5 - d)
			if cov <= 0 {
				continue
			}
			o := dst.PixOffset(x, y)
			dst.Pix[o] = clampU8(float64(dst.Pix[o])*(1-cov) + float64(ink.R)*cov)
			dst.Pix[o+1] = clampU8(float64(dst.Pix[o+1])*(1-cov) + float64(ink.G)*cov)
			dst.Pix[o+2] = clampU8(float64(dst.Pix[o+2])*(1-cov) + float64(ink.B)*cov)
		}
	}
}

func hexOr(s string, def color.RGBA) color.RGBA {
	if c, ok := parseHexColor(s); ok {
		return c
	}
	return def
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	}
	return v
}
