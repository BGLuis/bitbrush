package filters

import "math"

// noise.go — a small deterministic value-noise generator shared by the paper
// and pencil filters. No global rand, no time.Now(): every sample is a pure
// function of (x, y, seed), so any textured result reproduces from its params.

// hashNoise is an integer hash of (x, y, seed) mapped to [0,1). Negative
// coordinates wrap through the unsigned conversion, which is fine — the point
// is a well-scrambled, repeatable value per lattice cell.
func hashNoise(x, y, seed int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(seed)*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h) / float64(1<<32)
}

// smootherstep is Perlin's 6t⁵−15t⁴+10t³ ease (C² continuous), for lattice
// interpolation without visible grid creasing.
func smootherstep(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

// valueNoise2D samples smoothly interpolated value noise at (x, y) for seed.
// Output is in [0,1] and the lattice period is 1 — scale the inputs to choose
// a feature size.
func valueNoise2D(x, y float64, seed int) float64 {
	xi, yi := int(math.Floor(x)), int(math.Floor(y))
	xf, yf := x-float64(xi), y-float64(yi)

	tl := hashNoise(xi, yi, seed)
	tr := hashNoise(xi+1, yi, seed)
	bl := hashNoise(xi, yi+1, seed)
	br := hashNoise(xi+1, yi+1, seed)

	u, v := smootherstep(xf), smootherstep(yf)
	top := tl + u*(tr-tl)
	bot := bl + u*(br-bl)
	return top + v*(bot-top)
}

// fractalNoise sums `octaves` of valueNoise2D at doubling frequency and
// halving amplitude (fBm), normalised back to [0,1]. Each octave uses a
// derived seed so the layers don't align.
func fractalNoise(x, y float64, octaves, seed int) float64 {
	if octaves < 1 {
		octaves = 1
	}
	var sum, norm float64
	amp, freq := 1.0, 1.0
	for o := 0; o < octaves; o++ {
		sum += amp * valueNoise2D(x*freq, y*freq, seed+o*101)
		norm += amp
		amp *= 0.5
		freq *= 2
	}
	return sum / norm
}
