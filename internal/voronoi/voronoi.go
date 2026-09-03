// Package voronoi provides weighted Lloyd relaxation over a raster density
// field. It is pure Go, stdlib only, and deterministic: the same sites,
// weights and iteration count always return the same result.
//
// It backs the "stipple" filter today (weighted Voronoi stippling, Secord
// 2002) and is written generally enough to also drive a cell-fill mosaic
// generator later.
package voronoi

import "math"

// Site is a generator point in pixel coordinates. X and Y may be
// fractional; callers place the raster origin at (0,0).
type Site struct{ X, Y float64 }

// Relax moves each site toward the weighted centroid of its Voronoi cell,
// repeated `iterations` times, and returns the new positions. The input
// slice is not modified.
//
//   - weight is a w*h row-major, non-negative density field; negative
//     entries are treated as 0.
//   - Nearest-site assignment uses jump flooding (JFA+1): O(pixels·log(max(w,h)))
//     per iteration, with a brute-force fallback only for pixels JFA leaves
//     unassigned (possible when sites are heavily clustered).
//   - A tiny uniform bias is added to every pixel weight so a cell that
//     covers only zero-weight pixels still has a well-defined centroid. A
//     site that ends up owning no pixel at all is left where it is — this
//     keeps the result deterministic without a random reseed.
func Relax(sites []Site, weight []float64, w, h, iterations int) []Site {
	out := append([]Site(nil), sites...)
	if iterations <= 0 || w <= 0 || h <= 0 || len(out) == 0 || len(weight) < w*h {
		return out
	}
	n := len(out)
	sumW := make([]float64, n)
	sumX := make([]float64, n)
	sumY := make([]float64, n)

	// Reusable buffers for JFA: zero heap allocations during the relaxation loop
	bufA := make([]int, w*h)
	bufB := make([]int, w*h)

	for it := 0; it < iterations; it++ {
		owner := nearestSitesBuf(out, w, h, bufA, bufB)
		for i := range sumW {
			sumW[i], sumX[i], sumY[i] = 0, 0, 0
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				o := owner[y*w+x]
				if o < 0 {
					continue
				}
				wt := weight[y*w+x]
				if wt < 0 {
					wt = 0
				}
				wt += 1e-6
				sumW[o] += wt
				sumX[o] += wt * (float64(x) + 0.5)
				sumY[o] += wt * (float64(y) + 0.5)
			}
		}
		for i := 0; i < n; i++ {
			if sumW[i] <= 0 {
				continue
			}
			out[i].X = sumX[i] / sumW[i]
			out[i].Y = sumY[i] / sumW[i]
		}
	}
	return out
}

// nearestSites returns, for every pixel, the index of its nearest site.
func nearestSites(sites []Site, w, h int) []int {
	bufA := make([]int, w*h)
	bufB := make([]int, w*h)
	return nearestSitesBuf(sites, w, h, bufA, bufB)
}

func nearestSitesBuf(sites []Site, w, h int, curr, next []int) []int {
	for i := range curr {
		curr[i] = -1
	}
	for i, s := range sites {
		x := clampi(int(s.X), 0, w-1)
		y := clampi(int(s.Y), 0, h-1)
		curr[y*w+x] = i // fixed iteration order => deterministic on collision
	}

	step := 1
	for step < w || step < h {
		step <<= 1
	}
	for step >>= 1; step >= 1; step >>= 1 {
		jfaPassBuf(curr, next, sites, w, h, step)
		curr, next = next, curr
	}
	jfaPassBuf(curr, next, sites, w, h, 1) // JFA+1 refinement
	curr, next = next, curr

	// Fallback for any pixel still unassigned.
	for idx := 0; idx < len(curr); idx++ {
		if curr[idx] >= 0 {
			continue
		}
		x, y := idx%w, idx/w
		best, bestD := -1, math.MaxFloat64
		for i, s := range sites {
			d := (s.X-float64(x))*(s.X-float64(x)) + (s.Y-float64(y))*(s.Y-float64(y))
			if d < bestD {
				bestD, best = d, i
			}
		}
		curr[idx] = best
	}
	return curr
}

func jfaPassBuf(src, dst []int, sites []Site, w, h, step int) {
	for y := 0; y < h; y++ {
		py := float64(y) + 0.5
		for x := 0; x < w; x++ {
			best := src[y*w+x]
			bestD := math.MaxFloat64
			px := float64(x) + 0.5
			if best >= 0 {
				s := sites[best]
				dx, dy := s.X-px, s.Y-py
				bestD = dx*dx + dy*dy
			}
			for dy := -1; dy <= 1; dy++ {
				ny := y + dy*step
				if ny < 0 || ny >= h {
					continue
				}
				row := ny * w
				for dx := -1; dx <= 1; dx++ {
					nx := x + dx*step
					if nx < 0 || nx >= w {
						continue
					}
					o := src[row+nx]
					if o < 0 {
						continue
					}
					s := sites[o]
					dxs, dys := s.X-px, s.Y-py
					if d := dxs*dxs + dys*dys; d < bestD {
						bestD, best = d, o
					}
				}
			}
			dst[y*w+x] = best
		}
	}
}

func dist2(s Site, x, y int) float64 {
	dx := s.X - (float64(x) + 0.5)
	dy := s.Y - (float64(y) + 0.5)
	return dx*dx + dy*dy
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
