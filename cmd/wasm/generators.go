package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"bitbrush/internal/gradient"
	"bitbrush/internal/noisefield"
	"bitbrush/internal/palette"
)

// This file registers the generator + palette globals. Same {ok,...,error}
// shape and panic-recovery discipline as main.go; no algorithm logic.

func registerGenerators() {
	js.Global().Set("bitbrushRenderGradient", js.FuncOf(renderGradient))
	js.Global().Set("bitbrushGradientCSS", js.FuncOf(gradientCSS))
	js.Global().Set("bitbrushRenderNoiseField", js.FuncOf(renderNoiseField))
	js.Global().Set("bitbrushExtractPalette", js.FuncOf(extractPalette))
	js.Global().Set("bitbrushGenPalette", js.FuncOf(genPalette))
}

func okData(pix []byte) map[string]any {
	dst := js.Global().Get("Uint8Array").New(len(pix))
	js.CopyBytesToJS(dst, pix)
	return map[string]any{"ok": true, "data": dst, "error": ""}
}

func errData(err error) map[string]any {
	return map[string]any{"ok": false, "data": js.Null(), "error": err.Error()}
}

func hexList(cols []palette.RGB) []any {
	out := make([]any, len(cols))
	for i, c := range cols {
		out[i] = fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
	}
	return out
}

// bitbrushRenderGradient(paramsJSON string) -> {ok, data: Uint8Array|null,
// error}. paramsJSON carries stops/space/hue/easing plus width/height/angle
// (see gradient.Params).
func renderGradient(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "data": js.Null(), "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	img, err := gradient.GenerateJSON([]byte(args[0].String()))
	if err != nil {
		return errData(err)
	}
	return okData(img.Pix)
}

// bitbrushGradientCSS(paramsJSON string, cssOptionsJSON string) ->
// {ok, css: string, error}. cssOptionsJSON is
// {kind, angle, native, samples, selector, property}.
func gradientCSS(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "css": "", "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	p, err := gradient.ParseParams([]byte(args[0].String()))
	if err != nil {
		return map[string]any{"ok": false, "css": "", "error": err.Error()}
	}
	g, err := p.Gradient()
	if err != nil {
		return map[string]any{"ok": false, "css": "", "error": err.Error()}
	}

	var o struct {
		Kind     string  `json:"kind"`
		Angle    float64 `json:"angle"`
		Native   bool    `json:"native"`
		Samples  int     `json:"samples"`
		Selector string  `json:"selector"`
		Property string  `json:"property"`
	}
	if err := json.Unmarshal([]byte(args[1].String()), &o); err != nil {
		return map[string]any{"ok": false, "css": "", "error": fmt.Sprintf("bad css options: %v", err)}
	}

	kind := gradient.CSSLinear
	switch o.Kind {
	case "radial":
		kind = gradient.CSSRadial
	case "conic":
		kind = gradient.CSSConic
	}

	css := g.CSS(gradient.CSSOptions{
		Kind:     kind,
		AngleDeg: o.Angle,
		Native:   o.Native,
		Samples:  o.Samples,
		Selector: o.Selector,
		Property: o.Property,
	})
	return map[string]any{"ok": true, "css": css, "error": ""}
}

var nfField = map[string]noisefield.Field{
	"linear": noisefield.FieldLinear, "radial": noisefield.FieldRadial,
	"conic": noisefield.FieldConic, "reflected": noisefield.FieldReflected,
	"diamond": noisefield.FieldDiamond, "mesh": noisefield.FieldMesh,
	"freeform": noisefield.FieldFreeform, "flow": noisefield.FieldFlow,
}

var nfStyle = map[string]noisefield.Style{
	"metallic": noisefield.StyleMetallic, "chrome": noisefield.StyleChrome,
	"iridescent": noisefield.StyleIridescent, "holographic": noisefield.StyleHolographic,
	"neon": noisefield.StyleNeon, "pastel": noisefield.StylePastel,
	"duotone": noisefield.StyleDuotone, "rainbow": noisefield.StyleRainbow,
}

var nfTexture = map[string]noisefield.Texture{
	"smooth": noisefield.TextureSmooth, "grain": noisefield.TextureGrain,
	"frosted": noisefield.TextureFrosted, "wave": noisefield.TextureWave,
	"wrinkle": noisefield.TextureWrinkle, "paper": noisefield.TexturePaper,
}

// bitbrushRenderNoiseField(paramsJSON string, w int, h int) ->
// {ok, data: Uint8Array|null, error}. paramsJSON:
// {field, style, texture, stops:[hex], spots:[{color,x,y}], angle, scale,
// distortion, seed, time} — enums as lower-case names.
func renderNoiseField(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "data": js.Null(), "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	var p struct {
		Field   string   `json:"field"`
		Style   string   `json:"style"`
		Texture string   `json:"texture"`
		Stops   []string `json:"stops"`
		Spots   []struct {
			Color string  `json:"color"`
			X     float64 `json:"x"`
			Y     float64 `json:"y"`
		} `json:"spots"`
		Angle      float64 `json:"angle"`
		Scale      float64 `json:"scale"`
		Distortion float64 `json:"distortion"`
		Seed       int64   `json:"seed"`
		Time       float64 `json:"time"`
	}
	if err := json.Unmarshal([]byte(args[0].String()), &p); err != nil {
		return errData(fmt.Errorf("bad params: %w", err))
	}

	stops := make([]noisefield.RGB, 0, len(p.Stops))
	for _, s := range p.Stops {
		c, err := gradient.ParseHex(s)
		if err != nil {
			return errData(fmt.Errorf("bad stop %q: %w", s, err))
		}
		stops = append(stops, noisefield.RGB{R: c.R, G: c.G, B: c.B})
	}
	spots := make([]noisefield.Spot, 0, len(p.Spots))
	for _, s := range p.Spots {
		c, err := gradient.ParseHex(s.Color)
		if err != nil {
			return errData(fmt.Errorf("bad spot colour %q: %w", s.Color, err))
		}
		spots = append(spots, noisefield.Spot{Color: noisefield.RGB{R: c.R, G: c.G, B: c.B}, X: s.X, Y: s.Y})
	}

	img := noisefield.Render(noisefield.Params{
		Field:      nfField[p.Field],
		Style:      nfStyle[p.Style],
		Texture:    nfTexture[p.Texture],
		Stops:      stops,
		Spots:      spots,
		AngleDeg:   p.Angle,
		Scale:      p.Scale,
		Distortion: p.Distortion,
		Seed:       p.Seed,
		Time:       p.Time,
	}, args[1].Int(), args[2].Int())
	return okData(img.Pix)
}

// bitbrushExtractPalette(rgba Uint8Array, w int, h int, optionsJSON string)
// -> {ok, colors: string[], error}. optionsJSON:
// {count, method:"median-cut"|"kmeans", space:"srgb"|"oklab",
//
//	sort:"none"|"luma"|"hue"|"population", alphaThreshold, seed}.
func extractPalette(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "colors": nil, "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	src, err := decodeImage(args[0], args[1], args[2])
	if err != nil {
		return map[string]any{"ok": false, "colors": nil, "error": err.Error()}
	}

	var o struct {
		Count          int    `json:"count"`
		Method         string `json:"method"`
		Space          string `json:"space"`
		Sort           string `json:"sort"`
		AlphaThreshold int    `json:"alphaThreshold"`
		Seed           int64  `json:"seed"`
	}
	if err := json.Unmarshal([]byte(args[3].String()), &o); err != nil {
		return map[string]any{"ok": false, "colors": nil, "error": fmt.Sprintf("bad options: %v", err)}
	}

	method := palette.MethodMedianCut
	if o.Method == "kmeans" {
		method = palette.MethodKMeans
	}
	space := palette.SpaceOKLab
	if o.Space == "srgb" {
		space = palette.SpaceSRGB
	}
	sortKey := palette.SortNone
	switch o.Sort {
	case "luma":
		sortKey = palette.SortLuma
	case "hue":
		sortKey = palette.SortHue
	case "population":
		sortKey = palette.SortPopulation
	}

	cols := palette.Extract(src, palette.ExtractOptions{
		Count:          o.Count,
		Method:         method,
		Space:          space,
		Sort:           sortKey,
		AlphaThreshold: uint8(o.AlphaThreshold),
		Seed:           o.Seed,
	})
	return map[string]any{"ok": true, "colors": hexList(cols), "error": ""}
}

var harmonyRule = map[string]palette.HarmonyRule{
	"complementary": palette.Complementary, "analogous": palette.Analogous,
	"triadic": palette.Triadic, "tetradic": palette.Tetradic,
	"split": palette.SplitComplementary, "monochromatic": palette.Monochromatic,
}

// bitbrushGenPalette(optionsJSON string) -> {ok, colors: string[], error}.
// optionsJSON: {base:"#rrggbb", rule, n, wheel:"hsl"|"oklch", spread}.
func genPalette(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "colors": nil, "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	var o struct {
		Base   string  `json:"base"`
		Rule   string  `json:"rule"`
		N      int     `json:"n"`
		Wheel  string  `json:"wheel"`
		Spread float64 `json:"spread"`
	}
	if err := json.Unmarshal([]byte(args[0].String()), &o); err != nil {
		return map[string]any{"ok": false, "colors": nil, "error": fmt.Sprintf("bad options: %v", err)}
	}
	base, err := gradient.ParseHex(o.Base)
	if err != nil {
		return map[string]any{"ok": false, "colors": nil, "error": err.Error()}
	}
	wheel := palette.WheelHSL
	if o.Wheel == "oklch" {
		wheel = palette.WheelOKLCh
	}
	rule, ok := harmonyRule[o.Rule]
	if !ok {
		return map[string]any{"ok": false, "colors": nil, "error": fmt.Sprintf("unknown rule %q", o.Rule)}
	}

	cols := palette.Harmony(palette.RGB{R: base.R, G: base.G, B: base.B}, rule, o.N, palette.HarmonyOptions{
		Wheel:     wheel,
		SpreadDeg: o.Spread,
	})
	return map[string]any{"ok": true, "colors": hexList(cols), "error": ""}
}
