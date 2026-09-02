package palette

import (
	"image"
	"math"
	"math/rand"

	"bitbrush/internal/colorspace"
)

// KMeansOptions tunes KMeans. The zero value is usable: sRGB distance, 20
// Lloyd iterations, seed 0.
type KMeansOptions struct {
	// Space is the metric space the clustering runs in. SpaceOKLab groups
	// colours by perceived similarity; SpaceSRGB is the raw byte cube.
	Space Space
	// Iterations is the number of Lloyd refinement passes, clamped to
	// [1,100]. Zero means the default, 20.
	Iterations int
	// Seed seeds the k-means++ initialisation RNG. Identical inputs with an
	// identical Seed always yield an identical palette.
	Seed int64
}

// KMeans reduces img to k cluster colours via k-means: a deterministic,
// Seed-driven k-means++ initialisation followed by Lloyd iterations in the
// chosen space. Cluster centroids convert back to RGB through OKLab
// (SpaceOKLab) or a plain channel mean (SpaceSRGB). Empty clusters are
// re-seeded from the point farthest from its centroid.
//
// k <= 0 returns nil. k == 1 returns the single mean colour. When k is at
// least the number of distinct sampled colours, those distinct colours are
// returned directly. The result is sorted and de-duplicated, so it is
// stable for a given (img, k, opt).
func KMeans(img *image.RGBA, k int, opt KMeansOptions) []RGB {
	return kmeansPixels(samplePixels(img), k, opt)
}

// kmeansPixels is KMeans over an already-sampled pixel set (so Extract can
// apply its own alpha threshold first).
func kmeansPixels(pixels []RGB, k int, opt KMeansOptions) []RGB {
	if k <= 0 || len(pixels) == 0 {
		return nil
	}
	if k == 1 {
		return []RGB{channelMean(pixels)}
	}

	distinct := distinctColours(pixels)
	if k >= len(distinct) {
		return sortDedupeRGB(distinct)
	}

	iters := opt.Iterations
	switch {
	case iters == 0:
		iters = 20
	case iters < 1:
		iters = 1
	case iters > 100:
		iters = 100
	}

	space := opt.Space
	pts := make([][3]float64, len(pixels))
	for i, p := range pixels {
		pts[i] = coordOf(p, space)
	}

	centers := kmeansPlusPlusInit(pts, k, opt.Seed)

	assign := make([]int, len(pts))
	for i := range assign {
		assign[i] = -1
	}

	for it := 0; it < iters; it++ {
		changed := false
		for i, p := range pts {
			best, bestD := 0, math.Inf(1)
			for ci := range centers {
				if d := dist2(p, centers[ci]); d < bestD {
					best, bestD = ci, d
				}
			}
			if assign[i] != best {
				assign[i] = best
				changed = true
			}
		}
		if !changed {
			break
		}

		sums := make([][3]float64, k)
		counts := make([]int, k)
		for i, p := range pts {
			a := assign[i]
			sums[a][0] += p[0]
			sums[a][1] += p[1]
			sums[a][2] += p[2]
			counts[a]++
		}
		for ci := 0; ci < k; ci++ {
			if counts[ci] == 0 {
				continue
			}
			inv := 1 / float64(counts[ci])
			centers[ci] = [3]float64{sums[ci][0] * inv, sums[ci][1] * inv, sums[ci][2] * inv}
		}
		reseedEmpty(pts, assign, centers, counts)
	}

	pal := make([]RGB, len(centers))
	for i, c := range centers {
		pal[i] = coordToRGB(c, space)
	}
	return sortDedupeRGB(pal)
}

// kmeansPlusPlusInit picks k initial centres by the k-means++ rule, driven
// entirely by a math/rand stream seeded with seed, so the choice is
// reproducible.
func kmeansPlusPlusInit(pts [][3]float64, k int, seed int64) [][3]float64 {
	rng := rand.New(rand.NewSource(seed))
	centers := make([][3]float64, 0, k)
	centers = append(centers, pts[rng.Intn(len(pts))])

	d2 := make([]float64, len(pts))
	for len(centers) < k {
		var sum float64
		for i, p := range pts {
			best := dist2(p, centers[0])
			for ci := 1; ci < len(centers); ci++ {
				if d := dist2(p, centers[ci]); d < best {
					best = d
				}
			}
			d2[i] = best
			sum += best
		}
		if sum == 0 {
			// Every point already coincides with a centre; the remaining
			// slots cannot be distinguished. Repeat an arbitrary point.
			centers = append(centers, pts[rng.Intn(len(pts))])
			continue
		}
		target := rng.Float64() * sum
		acc, chosen := 0.0, len(pts)-1
		for i, v := range d2 {
			acc += v
			if acc >= target {
				chosen = i
				break
			}
		}
		centers = append(centers, pts[chosen])
	}
	return centers
}

// reseedEmpty moves each centre that captured no points onto the point
// currently farthest from its own centre, breaking ties by lowest index so
// the outcome is deterministic. A point is used for at most one reseed.
func reseedEmpty(pts [][3]float64, assign []int, centers [][3]float64, counts []int) {
	var used map[int]bool
	for ci := range centers {
		if counts[ci] > 0 {
			continue
		}
		if used == nil {
			used = make(map[int]bool)
		}
		far, farD := -1, math.Inf(-1)
		for i, p := range pts {
			if used[i] {
				continue
			}
			if d := dist2(p, centers[assign[i]]); d > farD {
				far, farD = i, d
			}
		}
		if far >= 0 {
			centers[ci] = pts[far]
			used[far] = true
		}
	}
}

func distinctColours(pixels []RGB) []RGB {
	seen := make(map[RGB]struct{}, len(pixels))
	out := make([]RGB, 0, len(pixels))
	for _, p := range pixels {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func channelMean(pixels []RGB) RGB {
	var sr, sg, sb uint64
	for _, p := range pixels {
		sr += uint64(p.R)
		sg += uint64(p.G)
		sb += uint64(p.B)
	}
	n := uint64(len(pixels))
	return RGB{uint8(sr / n), uint8(sg / n), uint8(sb / n)}
}

// coordOf places an RGB colour in the working coordinate system: OKLab
// (L,A,B) for SpaceOKLab, raw byte values for SpaceSRGB.
func coordOf(p RGB, space Space) [3]float64 {
	if space == SpaceOKLab {
		l := colorspace.RGBToOKLab(p.R, p.G, p.B)
		return [3]float64{l.L, l.A, l.B}
	}
	return [3]float64{float64(p.R), float64(p.G), float64(p.B)}
}

// coordToRGB is the inverse of coordOf for a centroid.
func coordToRGB(c [3]float64, space Space) RGB {
	if space == SpaceOKLab {
		r, g, b := colorspace.OKLabToRGB(colorspace.OKLab{L: c[0], A: c[1], B: c[2]})
		return RGB{r, g, b}
	}
	return RGB{clampByte(c[0]), clampByte(c[1]), clampByte(c[2])}
}

func clampByte(v float64) uint8 {
	n := math.Round(v)
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return uint8(n)
}

func dist2(a, b [3]float64) float64 {
	d0, d1, d2 := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return d0*d0 + d1*d1 + d2*d2
}
