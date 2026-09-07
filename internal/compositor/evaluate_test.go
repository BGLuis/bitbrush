package compositor

import (
	"bytes"
	"encoding/json"
	"image"
	"testing"

	"bitbrush/internal/filters"

	// Link every generator so SourceGenerator layers resolve, exactly as
	// cmd/wasm does.
	_ "bitbrush/internal/genall"
)

func gradientParams(t *testing.T) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"stops": []map[string]any{
			{"color": "#000000", "pos": 0.0},
			{"color": "#ffffff", "pos": 1.0},
		},
		"space": "srgb",
		"angle": 90.0,
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestEvaluateTwoLayerDeterministicAndDarkening(t *testing.T) {
	base := solid(24, 16, 180, 180, 180, 255)
	spec := Spec{
		Layers: []Layer{
			{Enabled: true, Source: SourceBase, Blend: BlendNormal, Opacity: 1},
			{Enabled: true, Source: SourceGradient, GenParams: gradientParams(t), Blend: BlendMultiply, Opacity: 1},
		},
	}

	a, err := Evaluate(spec, base, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Evaluate(spec, base, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("Evaluate is not deterministic: two runs differ")
	}
	if a.Bounds() != base.Bounds() {
		t.Fatalf("canvas %v, want base size %v", a.Bounds(), base.Bounds())
	}
	// multiply against a 0..1 ramp never brightens the base.
	for i := 0; i < len(a.Pix); i += 4 {
		if a.Pix[i] > 181 {
			t.Fatalf("pixel %d R=%d, multiply should not brighten past base 180", i/4, a.Pix[i])
		}
		if a.Pix[i+3] != 255 {
			t.Fatalf("pixel %d alpha=%d, want opaque", i/4, a.Pix[i+3])
		}
	}
}

func TestEvaluateThreeLayerGeneratorAndNoiseField(t *testing.T) {
	base := solid(32, 24, 90, 120, 150, 255)
	noiseParams, _ := json.Marshal(map[string]any{
		"field": "flow", "style": "duotone", "texture": "smooth",
		"stops": []string{"#101828", "#dc78a0"},
		"scale": 55.0, "distortion": 50.0, "seed": 7.0,
	})
	spec := Spec{
		Layers: []Layer{
			{Enabled: true, Source: SourceBase, Blend: BlendNormal, Opacity: 1},
			{Enabled: true, Source: SourceGenerator, Generator: "truchet", Blend: BlendScreen, Opacity: 0.7},
			{Enabled: true, Source: SourceNoiseField, GenParams: noiseParams, Blend: BlendSoftLight, Opacity: 0.4},
		},
	}
	a, err := Evaluate(spec, base, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Evaluate(spec, base, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("three-layer Evaluate is not deterministic")
	}
	for i := 3; i < len(a.Pix); i += 4 {
		if a.Pix[i] != 255 {
			t.Fatalf("pixel %d alpha=%d, want opaque (all layers opaque)", i/4, a.Pix[i])
		}
	}
}

func TestEvaluateChainMatchesDirectApply(t *testing.T) {
	base := solid(20, 20, 30, 160, 220, 255)
	spec := Spec{
		Layers: []Layer{{
			Enabled: true,
			Source:  SourceBase,
			Blend:   BlendNormal,
			Opacity: 1,
			Chain: []Stage{
				{Filter: "grayscale", Params: filters.Params{}},
				{Filter: "pixelate", Params: filters.Params{"blockSize": float64(4)}},
			},
		}},
	}
	got, err := Evaluate(spec, base, nil)
	if err != nil {
		t.Fatal(err)
	}

	gray, err := filters.Apply("grayscale", cloneRGBA(base), filters.Params{})
	if err != nil {
		t.Fatal(err)
	}
	want, err := filters.Apply("pixelate", gray, filters.Params{"blockSize": float64(4)})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("normal-over-transparent chain layer != direct filter application")
	}
}

func TestEvaluateSkipsDisabledLayers(t *testing.T) {
	base := solid(8, 8, 200, 60, 60, 255)
	spec := Spec{
		Layers: []Layer{
			{Enabled: true, Source: SourceBase, Blend: BlendNormal, Opacity: 1},
			{Enabled: false, Source: SourceImage, ImageIndex: 99, Blend: BlendNormal, Opacity: 1},
		},
	}
	out, err := Evaluate(spec, base, nil)
	if err != nil {
		t.Fatalf("disabled layer with a bad index should be skipped, got %v", err)
	}
	if !bytes.Equal(out.Pix, base.Pix) {
		t.Fatal("single enabled base layer should reproduce the base")
	}
}

func TestEvaluateImageLayer(t *testing.T) {
	base := solid(10, 10, 0, 0, 0, 255)
	extra := solid(10, 10, 255, 255, 255, 255)
	spec := Spec{
		Layers: []Layer{
			{Enabled: true, Source: SourceBase, Blend: BlendNormal, Opacity: 1},
			{Enabled: true, Source: SourceImage, ImageIndex: 0, Blend: BlendNormal, Opacity: 0.5},
		},
	}
	out, err := Evaluate(spec, base, []*image.RGBA{extra})
	if err != nil {
		t.Fatal(err)
	}
	if v := out.Pix[0]; v < 127 || v > 129 {
		t.Fatalf("black base + white image @0.5 -> %d, want ~128", v)
	}
}

func TestEvaluateErrors(t *testing.T) {
	base := solid(8, 8, 0, 0, 0, 255)
	tests := []struct {
		name string
		spec Spec
		base *image.RGBA
	}{
		{
			name: "unknown generator",
			spec: Spec{Layers: []Layer{{Enabled: true, Source: SourceGenerator, Generator: "nope", Blend: BlendNormal, Opacity: 1}}},
			base: base,
		},
		{
			name: "base layer without base image",
			spec: Spec{Width: 8, Height: 8, Layers: []Layer{{Enabled: true, Source: SourceBase, Blend: BlendNormal, Opacity: 1}}},
			base: nil,
		},
		{
			name: "image index out of range",
			spec: Spec{Width: 8, Height: 8, Layers: []Layer{{Enabled: true, Source: SourceImage, ImageIndex: 3, Blend: BlendNormal, Opacity: 1}}},
			base: base,
		},
		{
			name: "no size and no base",
			spec: Spec{Layers: []Layer{{Enabled: true, Source: SourceGradient, GenParams: gradientParams(t), Blend: BlendNormal, Opacity: 1}}},
			base: nil,
		},
	}
	for _, tc := range tests {
		if _, err := Evaluate(tc.spec, tc.base, nil); err == nil {
			t.Errorf("%s: expected an error", tc.name)
		}
	}
}
