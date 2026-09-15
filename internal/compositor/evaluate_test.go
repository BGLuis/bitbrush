package compositor

import (
	"bytes"
	"encoding/json"
	"image"
	"testing"

	"bitbrush/internal/filters"
	"bitbrush/internal/mask"

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

func TestEvaluateLayerWithTransformMovesContent(t *testing.T) {
	base := solid(8, 8, 200, 60, 60, 255)
	extra := solid(8, 8, 255, 255, 255, 255)
	makeSpec := func(tr *Transform) Spec {
		return Spec{
			Layers: []Layer{
				{Enabled: true, Source: SourceBase, Blend: BlendNormal, Opacity: 1},
				{Enabled: true, Source: SourceImage, ImageIndex: 0, Blend: BlendNormal, Opacity: 1, Transform: tr},
			},
		}
	}

	plain, err := Evaluate(makeSpec(nil), base, []*image.RGBA{extra})
	if err != nil {
		t.Fatal(err)
	}
	moved, err := Evaluate(makeSpec(&Transform{OffsetX: 0.4}), base, []*image.RGBA{extra})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(plain.Pix, moved.Pix) {
		t.Fatal("a non-identity Transform on a layer had no effect on Evaluate's output")
	}

	moved2, err := Evaluate(makeSpec(&Transform{OffsetX: 0.4}), base, []*image.RGBA{extra})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(moved.Pix, moved2.Pix) {
		t.Fatal("Evaluate with a Transform is not deterministic: two runs differ")
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

func TestEvaluateTextLayer(t *testing.T) {
	textJSON, err := json.Marshal(map[string]any{
		"content":    "Hi",
		"fontFamily": "go",
		"size":       20.0,
		"color":      "#ffffff",
		"x":          0.5,
		"y":          0.5,
		"align":      "center",
		"opacity":    1.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	spec := Spec{
		Width:  64,
		Height: 64,
		Layers: []Layer{
			{
				Enabled:   true,
				Source:    SourceText,
				GenParams: textJSON,
				Blend:     BlendNormal,
				Opacity:   1.0,
			},
		},
	}

	out, err := Evaluate(spec, nil, nil)
	if err != nil {
		t.Fatalf("Evaluate with SourceText failed: %v", err)
	}

	hasPixel := false
	for i := 3; i < len(out.Pix); i += 4 {
		if out.Pix[i] > 0 {
			hasPixel = true
			break
		}
	}
	if !hasPixel {
		t.Errorf("expected text layer to render visible pixels")
	}
}

func TestEvaluateStageWithMask(t *testing.T) {
	base := solid(64, 64, 100, 100, 100, 255)
	// Apply invert filter, but masked to only a sub-rect
	maskJSON := []byte(`{"kind":"rect","x":0.25,"y":0.25,"w":0.5,"h":0.5,"feather":0}`)

	stage := Stage{
		Filter: "grayscale",
		Params: filters.Params{"brightness": 100.0},
	}
	if err := json.Unmarshal(maskJSON, &stage.Mask); err != nil {
		t.Fatal(err)
	}

	spec := Spec{
		Layers: []Layer{
			{
				Enabled: true,
				Source:  SourceBase,
				Blend:   BlendNormal,
				Opacity: 1.0,
				Chain:   []Stage{stage},
			},
		},
	}

	out, err := Evaluate(spec, base, nil)
	if err != nil {
		t.Fatalf("Evaluate with Stage.Mask failed: %v", err)
	}

	// Inside rect (32, 32) brightness is boosted
	idxInside := out.PixOffset(32, 32)
	// Outside rect (5, 5) remains original 100
	idxOutside := out.PixOffset(5, 5)

	if out.Pix[idxInside] <= 100 {
		t.Errorf("inside mask R=%d, want boosted > 100", out.Pix[idxInside])
	}
	if out.Pix[idxOutside] != 100 {
		t.Errorf("outside mask R=%d, want original 100", out.Pix[idxOutside])
	}
}

func TestEvaluateLayerMaskRevealsBelowOutsideRegion(t *testing.T) {
	base := solid(64, 64, 100, 100, 100, 255)
	overlay := solid(64, 64, 200, 50, 50, 255)
	spec := Spec{
		Layers: []Layer{
			{Enabled: true, Source: SourceBase, Blend: BlendNormal, Opacity: 1},
			{
				Enabled: true, Source: SourceImage, ImageIndex: 0, Blend: BlendNormal, Opacity: 1,
				Mask: &mask.Mask{Kind: mask.KindRect, X: 0.25, Y: 0.25, W: 0.5, H: 0.5},
			},
		},
	}
	out, err := Evaluate(spec, base, []*image.RGBA{overlay})
	if err != nil {
		t.Fatal(err)
	}

	// Outside the mask: the layer is fully hidden, base shows through unchanged.
	if r, g, b, a := at(out, 5, 5); r != 100 || g != 100 || b != 100 || a != 255 {
		t.Fatalf("outside mask = %d,%d,%d,%d, want base 100,100,100,255 untouched", r, g, b, a)
	}
	// Inside the mask: the layer composites normally, as if unmasked.
	if r, g, b, a := at(out, 32, 32); r != 200 || g != 50 || b != 50 || a != 255 {
		t.Fatalf("inside mask = %d,%d,%d,%d, want overlay 200,50,50,255", r, g, b, a)
	}
}

func TestEvaluateLayerMaskFeatherBlendsAtEdge(t *testing.T) {
	base := solid(64, 64, 100, 100, 100, 255)
	overlay := solid(64, 64, 200, 50, 50, 255)
	spec := Spec{
		Layers: []Layer{
			{Enabled: true, Source: SourceBase, Blend: BlendNormal, Opacity: 1},
			{
				Enabled: true, Source: SourceImage, ImageIndex: 0, Blend: BlendNormal, Opacity: 1,
				Mask: &mask.Mask{Kind: mask.KindRect, X: 0.25, Y: 0.25, W: 0.5, H: 0.5, Feather: 0.2},
			},
		},
	}
	out, err := Evaluate(spec, base, []*image.RGBA{overlay})
	if err != nil {
		t.Fatal(err)
	}
	// (17, 32) sits inside the feather band (1px past the 16px rect edge,
	// with a ~6.4px feather) — strictly between base and overlay, not equal
	// to either, proving the edge actually blends rather than hard-cutting.
	r, _, _, _ := at(out, 17, 32)
	if r <= 100 || r >= 200 {
		t.Fatalf("feathered edge R=%d, want strictly between base 100 and overlay 200", r)
	}
}

func TestEvaluateLayerMaskLumaSamplesBelowNotLayer(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			o := base.PixOffset(x, y)
			if x < 32 {
				base.Pix[o], base.Pix[o+1], base.Pix[o+2], base.Pix[o+3] = 10, 10, 10, 255 // dark half
			} else {
				base.Pix[o], base.Pix[o+1], base.Pix[o+2], base.Pix[o+3] = 250, 250, 250, 255 // bright half
			}
		}
	}
	overlay := solid(64, 64, 0, 255, 0, 255) // flat colour: no luminance variation of its own
	spec := Spec{
		Layers: []Layer{
			{Enabled: true, Source: SourceBase, Blend: BlendNormal, Opacity: 1},
			{
				Enabled: true, Source: SourceImage, ImageIndex: 0, Blend: BlendNormal, Opacity: 1,
				Mask: &mask.Mask{Kind: mask.KindLuma, Threshold: 0.5},
			},
		},
	}
	out, err := Evaluate(spec, base, []*image.RGBA{overlay})
	if err != nil {
		t.Fatal(err)
	}
	// If luma were sampled from the (flat) overlay instead of the base, both
	// sides would come out identical. They don't: the dark half of the BASE
	// hides the layer, the bright half reveals it.
	if r, g, b, a := at(out, 5, 32); r != 10 || g != 10 || b != 10 || a != 255 {
		t.Fatalf("dark-below side = %d,%d,%d,%d, want base 10,10,10,255 (layer hidden)", r, g, b, a)
	}
	if r, g, b, a := at(out, 50, 32); r != 0 || g != 255 || b != 0 || a != 255 {
		t.Fatalf("bright-below side = %d,%d,%d,%d, want overlay 0,255,0,255 (layer revealed)", r, g, b, a)
	}
}

func TestEvaluateLayerMaskIsDeterministic(t *testing.T) {
	base := solid(16, 16, 50, 50, 50, 255)
	overlay := solid(16, 16, 220, 220, 220, 255)
	makeSpec := func(m *mask.Mask) Spec {
		return Spec{
			Layers: []Layer{
				{Enabled: true, Source: SourceBase, Blend: BlendNormal, Opacity: 1},
				{Enabled: true, Source: SourceImage, ImageIndex: 0, Blend: BlendNormal, Opacity: 1, Mask: m},
			},
		}
	}

	plain, err := Evaluate(makeSpec(nil), base, []*image.RGBA{overlay})
	if err != nil {
		t.Fatal(err)
	}
	masked, err := Evaluate(makeSpec(&mask.Mask{Kind: mask.KindRect, X: 0.25, Y: 0.25, W: 0.5, H: 0.5}), base, []*image.RGBA{overlay})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(plain.Pix, masked.Pix) {
		t.Fatal("a layer Mask had no effect on Evaluate's output")
	}

	masked2, err := Evaluate(makeSpec(&mask.Mask{Kind: mask.KindRect, X: 0.25, Y: 0.25, W: 0.5, H: 0.5}), base, []*image.RGBA{overlay})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(masked.Pix, masked2.Pix) {
		t.Fatal("Evaluate with a layer Mask is not deterministic: two runs differ")
	}
}
