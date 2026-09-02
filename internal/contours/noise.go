package contours

import "math"

// Classic Perlin gradient noise in 3-D with a seed-shuffled permutation
// table, plus fBm with a light domain warp — the heightfield behind the
// contour map. Ported from the reference "Living Landscape" sketch.

type perlin struct{ perm [512]int }

func newPerlin(seed int64) *perlin {
	var p [256]int
	for i := range p {
		p[i] = i
	}
	st := uint64(seed) % 2147483647
	if st == 0 {
		st = 1
	}
	for i := 255; i > 0; i-- {
		st = (st * 16807) % 2147483647
		j := int(st % uint64(i+1))
		p[i], p[j] = p[j], p[i]
	}
	pl := &perlin{}
	for i := 0; i < 512; i++ {
		pl.perm[i] = p[i&255]
	}
	return pl
}

func fade(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }

func grad(hash int, x, y, z float64) float64 {
	h := hash & 15
	u := x
	if h >= 8 {
		u = y
	}
	v := z
	if h < 4 {
		v = y
	} else if h == 12 || h == 14 {
		v = x
	}
	if h&1 != 0 {
		u = -u
	}
	if h&2 != 0 {
		v = -v
	}
	return u + v
}

func lerp(t, a, b float64) float64 { return a + t*(b-a) }

func (pl *perlin) at(x, y, z float64) float64 {
	X := int(math.Floor(x)) & 255
	Y := int(math.Floor(y)) & 255
	Z := int(math.Floor(z)) & 255
	x -= math.Floor(x)
	y -= math.Floor(y)
	z -= math.Floor(z)
	u, v, w := fade(x), fade(y), fade(z)
	pm := &pl.perm
	A := pm[X] + Y
	AA := pm[A] + Z
	AB := pm[A+1] + Z
	B := pm[X+1] + Y
	BA := pm[B] + Z
	BB := pm[B+1] + Z
	return lerp(w,
		lerp(v,
			lerp(u, grad(pm[AA], x, y, z), grad(pm[BA], x-1, y, z)),
			lerp(u, grad(pm[AB], x, y-1, z), grad(pm[BB], x-1, y-1, z))),
		lerp(v,
			lerp(u, grad(pm[AA+1], x, y, z-1), grad(pm[BA+1], x-1, y, z-1)),
			lerp(u, grad(pm[AB+1], x, y-1, z-1), grad(pm[BB+1], x-1, y-1, z-1))))
}

func (pl *perlin) fbm(x, y, z float64, octaves int) float64 {
	a, f, sum, norm := 0.55, 1.0, 0.0, 0.0
	for o := 0; o < octaves; o++ {
		sum += pl.at(x*f, y*f, z*f*0.6) * a
		norm += a
		a *= 0.5
		f *= 2.05
	}
	return clampf(sum/norm*1.15+0.5, 0.001, 0.999)
}
