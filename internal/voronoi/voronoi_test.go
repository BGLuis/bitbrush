package voronoi

import (
	"math"
	"math/rand"
	"testing"
)

func uniform(w, h int) []float64 {
	f := make([]float64, w*h)
	for i := range f {
		f[i] = 1
	}
	return f
}

func minPairDist(sites []Site) float64 {
	m := math.MaxFloat64
	for i := 0; i < len(sites); i++ {
		for j := i + 1; j < len(sites); j++ {
			d := math.Hypot(sites[i].X-sites[j].X, sites[i].Y-sites[j].Y)
			if d < m {
				m = d
			}
		}
	}
	return m
}

func TestRelaxIterationsZeroIsIdentity(t *testing.T) {
	in := []Site{{1, 1}, {2, 2}, {3, 3}}
	out := Relax(in, uniform(8, 8), 8, 8, 0)
	if len(out) != len(in) {
		t.Fatalf("len %d != %d", len(out), len(in))
	}
	for i := range in {
		if out[i] != in[i] {
			t.Fatalf("site %d moved: %v != %v", i, out[i], in[i])
		}
	}
}

func TestRelaxDoesNotMutateInput(t *testing.T) {
	in := []Site{{2, 2}, {60, 60}, {5, 55}}
	cp := append([]Site(nil), in...)
	Relax(in, uniform(64, 64), 64, 64, 10)
	for i := range in {
		if in[i] != cp[i] {
			t.Fatalf("Relax mutated its input at %d", i)
		}
	}
}

func TestRelaxSpreadsClusteredSites(t *testing.T) {
	const w, h = 64, 64
	rng := rand.New(rand.NewSource(1))
	sites := make([]Site, 40)
	for i := range sites {
		// all bunched in a 6x6 corner
		sites[i] = Site{X: rng.Float64() * 6, Y: rng.Float64() * 6}
	}
	before := minPairDist(sites)
	out := Relax(sites, uniform(w, h), w, h, 40)
	after := minPairDist(out)
	if after <= before {
		t.Fatalf("uniform-weight relaxation should spread sites out: before=%.3f after=%.3f", before, after)
	}
}

func TestRelaxConverges(t *testing.T) {
	const w, h = 48, 48
	rng := rand.New(rand.NewSource(7))
	sites := make([]Site, 25)
	for i := range sites {
		sites[i] = Site{X: rng.Float64() * w, Y: rng.Float64() * h}
	}
	move := func(a, b []Site) float64 {
		var s float64
		for i := range a {
			s += math.Hypot(a[i].X-b[i].X, a[i].Y-b[i].Y)
		}
		return s
	}
	s1 := Relax(sites, uniform(w, h), w, h, 1)
	first := move(sites, s1)
	s10 := Relax(sites, uniform(w, h), w, h, 10)
	s11 := Relax(sites, uniform(w, h), w, h, 11)
	last := move(s10, s11)
	if last >= first {
		t.Fatalf("expected convergence: first-step move=%.3f, later-step move=%.3f", first, last)
	}
}

func TestRelaxDeterministic(t *testing.T) {
	const w, h = 40, 40
	rng := rand.New(rand.NewSource(3))
	sites := make([]Site, 30)
	wt := make([]float64, w*h)
	for i := range sites {
		sites[i] = Site{X: rng.Float64() * w, Y: rng.Float64() * h}
	}
	for i := range wt {
		wt[i] = rng.Float64()
	}
	a := Relax(sites, wt, w, h, 15)
	b := Relax(sites, wt, w, h, 15)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("non-deterministic at site %d: %v != %v", i, a[i], b[i])
		}
	}
}

func TestRelaxAllZeroWeightIsFinite(t *testing.T) {
	const w, h = 32, 32
	sites := []Site{{4, 4}, {20, 8}, {16, 24}, {28, 28}}
	out := Relax(sites, make([]float64, w*h), w, h, 12)
	for i, s := range out {
		if math.IsNaN(s.X) || math.IsNaN(s.Y) || math.IsInf(s.X, 0) || math.IsInf(s.Y, 0) {
			t.Fatalf("site %d not finite: %v", i, s)
		}
	}
}
