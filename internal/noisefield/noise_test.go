package noisefield

import (
	"math"
	"testing"
)

func TestSnoiseRoughlyBounded(t *testing.T) {
	minv, maxv := math.Inf(1), math.Inf(-1)
	for i := -80; i <= 80; i++ {
		for j := -80; j <= 80; j++ {
			v := snoise(float64(i)*0.37, float64(j)*0.53)
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatalf("snoise(%d,%d) = %v", i, j, v)
			}
			if v < minv {
				minv = v
			}
			if v > maxv {
				maxv = v
			}
			if v < -1.05 || v > 1.05 {
				t.Fatalf("snoise(%d,%d) = %v outside [-1,1]", i, j, v)
			}
		}
	}
	// Sanity: the grid should actually exercise a decent swing.
	if maxv-minv < 0.5 {
		t.Fatalf("snoise output barely varies over the grid: [%v, %v]", minv, maxv)
	}
}

func TestFbmInUnitRange(t *testing.T) {
	for i := 0; i < 240; i++ {
		for j := 0; j < 240; j++ {
			v := fbm(float64(i)*0.11, float64(j)*0.19)
			if v < 0.0 || v > 1.0 {
				t.Fatalf("fbm(%d,%d) = %v outside [0,1]", i, j, v)
			}
		}
	}
}

func TestSnoiseDeterministic(t *testing.T) {
	if snoise(1.25, -3.75) != snoise(1.25, -3.75) {
		t.Fatal("snoise not deterministic")
	}
}

func TestModxNegatives(t *testing.T) {
	got := modx(-1.0, 6.0)
	if math.Abs(got-5.0) > 1e-12 {
		t.Fatalf("modx(-1,6) = %v, want 5", got)
	}
	if math.Abs(modx(7.5, 3.0)-1.5) > 1e-12 {
		t.Fatalf("modx(7.5,3) = %v, want 1.5", modx(7.5, 3.0))
	}
}
