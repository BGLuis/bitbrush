package chladni

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestChladniDeterministic(t *testing.T) {
	p := params{
		Seed:      1234,
		N:         3,
		M:         5,
		Particles: 5000,
		Palette:   "sand",
	}
	blob, _ := json.Marshal(p)

	img1, err := render(blob, 100, 100)
	if err != nil {
		t.Fatalf("render 1 failed: %v", err)
	}
	img2, err := render(blob, 100, 100)
	if err != nil {
		t.Fatalf("render 2 failed: %v", err)
	}

	if !bytes.Equal(img1.Pix, img2.Pix) {
		t.Fatal("chladni is not deterministic across identical calls")
	}
}

func TestChladniModes(t *testing.T) {
	modes := []struct{ n, m int }{
		{1, 2},
		{2, 3},
		{4, 4},
		{5, 7},
	}
	for _, tc := range modes {
		p := params{
			Seed:      42,
			N:         tc.n,
			M:         tc.m,
			Particles: 2000,
		}
		blob, _ := json.Marshal(p)
		img, err := render(blob, 80, 80)
		if err != nil {
			t.Errorf("render mode (%d,%d) failed: %v", tc.n, tc.m, err)
		}
		if img == nil || img.Bounds().Dx() != 80 || img.Bounds().Dy() != 80 {
			t.Errorf("bad bounds for (%d,%d)", tc.n, tc.m)
		}
	}
}
