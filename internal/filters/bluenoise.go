package filters

import (
	"math"
	"sync"
)

// blueNoise64 is a 64x64 tile of thresholds in [0,1), with a blue-noise
// (high-frequency) distribution — no low-frequency clumping, so ordered
// dithering with it looks closer to error diffusion than Bayer does.
//
// The tile is built once, lazily, by the void-and-cluster method
// (Ulichney 1993): it is fully deterministic, so the dither output still
// reproduces from its parameters alone.

const blueNoiseDim = 64

var (
	blueNoiseOnce sync.Once
	blueNoiseTile [blueNoiseDim * blueNoiseDim]float64
)

func blueNoiseAt(x, y int) float64 {
	blueNoiseOnce.Do(buildBlueNoise)
	x = ((x % blueNoiseDim) + blueNoiseDim) % blueNoiseDim
	y = ((y % blueNoiseDim) + blueNoiseDim) % blueNoiseDim
	return blueNoiseTile[y*blueNoiseDim+x]
}

func buildBlueNoise() {
	const n = blueNoiseDim
	const sigma = 1.9
	// Gaussian energy kernel, toroidal, precomputed offsets within 3 sigma.
	rad := int(math.Ceil(3 * sigma))
	type off struct {
		dx, dy int
		wgt    float64
	}
	var kernel []off
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			kernel = append(kernel, off{dx, dy, math.Exp(-float64(dx*dx+dy*dy) / (2 * sigma * sigma))})
		}
	}

	binary := make([]bool, n*n)
	energy := make([]float64, n*n)
	splat := func(i int, sign float64) {
		x0, y0 := i%n, i/n
		for _, k := range kernel {
			xx := ((x0+k.dx)%n + n) % n
			yy := ((y0+k.dy)%n + n) % n
			energy[yy*n+xx] += sign * k.wgt
		}
	}
	tightestCluster := func() int {
		best, bestE := -1, math.Inf(-1)
		for i := 0; i < n*n; i++ {
			if binary[i] && energy[i] > bestE {
				best, bestE = i, energy[i]
			}
		}
		return best
	}
	largestVoid := func() int {
		best, bestE := -1, math.Inf(1)
		for i := 0; i < n*n; i++ {
			if !binary[i] && energy[i] < bestE {
				best, bestE = i, energy[i]
			}
		}
		return best
	}

	// Initial pattern: a scattered ~10% set from a deterministic hash, then
	// relaxed so no two points cluster.
	initialCount := n * n / 10
	placed := 0
	for i := 0; i < n*n && placed < initialCount; i++ {
		if hashNoise(i%n, i/n, 12345) < 0.1 {
			binary[i] = true
			splat(i, +1)
			placed++
		}
	}
	for iter := 0; iter < n*n; iter++ {
		c := tightestCluster()
		binary[c] = false
		splat(c, -1)
		v := largestVoid()
		binary[v] = true
		splat(v, +1)
		if c == v {
			break
		}
	}

	rank := make([]int, n*n)
	for i := range rank {
		rank[i] = -1
	}

	// Phase 1: remove points from the initial pattern, ranks placed..0.
	work := append([]bool(nil), binary...)
	we := append([]float64(nil), energy...)
	splatW := func(i int, sign float64) {
		x0, y0 := i%n, i/n
		for _, k := range kernel {
			xx := ((x0+k.dx)%n + n) % n
			yy := ((y0+k.dy)%n + n) % n
			we[yy*n+xx] += sign * k.wgt
		}
	}
	for r := placed - 1; r >= 0; r-- {
		best, bestE := -1, math.Inf(-1)
		for i := 0; i < n*n; i++ {
			if work[i] && we[i] > bestE {
				best, bestE = i, we[i]
			}
		}
		work[best] = false
		splatW(best, -1)
		rank[best] = r
	}

	// Phase 2: add points into the voids, ranks placed..n*n-1.
	work = append(work[:0], binary...)
	copy(we, energy)
	for r := placed; r < n*n; r++ {
		best, bestE := -1, math.Inf(1)
		for i := 0; i < n*n; i++ {
			if !work[i] && we[i] < bestE {
				best, bestE = i, we[i]
			}
		}
		work[best] = true
		splatW(best, +1)
		rank[best] = r
	}

	denom := float64(n * n)
	for i := 0; i < n*n; i++ {
		blueNoiseTile[i] = (float64(rank[i]) + 0.5) / denom
	}
}
