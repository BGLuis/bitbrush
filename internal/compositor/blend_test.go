package compositor

import (
	"math"
	"testing"
)

func TestBlendFnValues(t *testing.T) {
	const eps = 1e-9
	cases := []struct {
		mode   BlendMode
		cb, cs float64
		want   float64
	}{
		{BlendNormal, 0.3, 0.7, 0.7},
		{BlendMultiply, 0.5, 0.5, 0.25},
		{BlendScreen, 0.5, 0.5, 0.75},
		{BlendOverlay, 0.5, 0.5, 0.5},
		{BlendOverlay, 0.25, 0.5, 0.25},
		{BlendDarken, 0.3, 0.7, 0.3},
		{BlendLighten, 0.3, 0.7, 0.7},
		{BlendColorDodge, 0.5, 0.5, 1.0},
		{BlendColorDodge, 0.5, 0.0, 0.5},
		{BlendColorDodge, 0.0, 0.9, 0.0},
		{BlendColorBurn, 0.5, 0.5, 0.0},
		{BlendColorBurn, 0.5, 1.0, 0.5},
		{BlendColorBurn, 1.0, 0.0, 1.0},
		{BlendHardLight, 0.5, 0.5, 0.5},
		{BlendHardLight, 0.5, 0.75, 0.75},
		{BlendDifference, 0.2, 0.7, 0.5},
		{BlendExclusion, 0.5, 0.5, 0.5},
		{BlendExclusion, 0.2, 0.7, 0.62},
		{BlendLinearDodge, 0.6, 0.6, 1.0},
		{BlendLinearDodge, 0.3, 0.4, 0.7},
		{BlendSubtract, 0.3, 0.5, 0.0},
		{BlendSubtract, 0.8, 0.3, 0.5},
		{BlendSoftLight, 0.37, 0.5, 0.37},            // cs == 0.5 is identity for any cb
		{BlendSoftLight, 0.4, 0.25, 0.28},            // cs < 0.5 darkens
		{BlendSoftLight, 0.4, 0.75, 0.5162277660168}, // cs > 0.5 lightens toward sqrt(cb)
	}
	for _, c := range cases {
		fn, ok := blendFn(c.mode)
		if !ok {
			t.Fatalf("blendFn(%q) not recognised", c.mode)
		}
		got := fn(c.cb, c.cs)
		if math.Abs(got-c.want) > eps {
			t.Errorf("%s(cb=%g, cs=%g) = %.12f, want %.12f", c.mode, c.cb, c.cs, got, c.want)
		}
	}
}

func TestBlendFnUnknownFallsBack(t *testing.T) {
	if _, ok := blendFn(BlendMode("no-such-mode")); ok {
		t.Fatal("blendFn reported an unknown mode as recognised")
	}
}
