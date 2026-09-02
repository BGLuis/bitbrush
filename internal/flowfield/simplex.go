package flowfield

import (
	"math"
	"math/rand"
)

// 2-D simplex noise with a seed-shuffled permutation, ported from the
// reference "Sumi Field" sketch. Output is roughly in [-1, 1].

var grad2 = [8][2]float64{
	{1, 1}, {-1, 1}, {1, -1}, {-1, -1}, {1, 0}, {-1, 0}, {0, 1}, {0, -1},
}

type simplex struct {
	perm     [512]int
	permMod8 [512]int
}

func newSimplex(rng *rand.Rand) *simplex {
	var p [256]int
	for i := range p {
		p[i] = i
	}
	for i := 255; i > 0; i-- {
		j := rng.Intn(i + 1)
		p[i], p[j] = p[j], p[i]
	}
	s := &simplex{}
	for i := 0; i < 512; i++ {
		s.perm[i] = p[i&255]
		s.permMod8[i] = s.perm[i] % 8
	}
	return s
}

func (s *simplex) noise2(xin, yin float64) float64 {
	const (
		f2 = 0.3660254037844386  // 0.5*(sqrt(3)-1)
		g2 = 0.21132486540518713 // (3-sqrt(3))/6
	)
	sk := (xin + yin) * f2
	i := math.Floor(xin + sk)
	j := math.Floor(yin + sk)
	t := (i + j) * g2
	x0 := xin - (i - t)
	y0 := yin - (j - t)
	var i1, j1 int
	if x0 > y0 {
		i1 = 1
	} else {
		j1 = 1
	}
	x1 := x0 - float64(i1) + g2
	y1 := y0 - float64(j1) + g2
	x2 := x0 - 1 + 2*g2
	y2 := y0 - 1 + 2*g2
	ii := int(i) & 255
	jj := int(j) & 255
	n := 0.0
	if tt := 0.5 - x0*x0 - y0*y0; tt > 0 {
		g := grad2[s.permMod8[ii+s.perm[jj]]]
		tt *= tt
		n += tt * tt * (g[0]*x0 + g[1]*y0)
	}
	if tt := 0.5 - x1*x1 - y1*y1; tt > 0 {
		g := grad2[s.permMod8[ii+i1+s.perm[jj+j1]]]
		tt *= tt
		n += tt * tt * (g[0]*x1 + g[1]*y1)
	}
	if tt := 0.5 - x2*x2 - y2*y2; tt > 0 {
		g := grad2[s.permMod8[ii+1+s.perm[jj+1]]]
		tt *= tt
		n += tt * tt * (g[0]*x2 + g[1]*y2)
	}
	return 70 * n
}
