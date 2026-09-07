package main

import (
	"encoding/json"
	"fmt"
	"math"
	"syscall/js"

	"bitbrush/internal/anim"
	"bitbrush/internal/generators"
	"bitbrush/internal/gradient"
	"bitbrush/internal/noisefield"
	"bitbrush/internal/palette"

	// Algorithmic generators self-register with internal/generators via
	// their init(); blank-import each so it is linked into the WASM binary.
	_ "bitbrush/internal/genall"
)

// This file registers the generator + palette globals. Same {ok,...,error}
// shape and panic-recovery discipline as main.go; no algorithm logic.

func registerGenerators() {
	js.Global().Set("bitbrushRenderGradient", js.FuncOf(renderGradient))
	js.Global().Set("bitbrushGradientCSS", js.FuncOf(gradientCSS))
	js.Global().Set("bitbrushRenderNoiseField", js.FuncOf(renderNoiseField))
	js.Global().Set("bitbrushRenderNoiseFieldGIF", js.FuncOf(renderNoiseFieldGIF))
	js.Global().Set("bitbrushRenderGenerator", js.FuncOf(renderGenerator))
	js.Global().Set("bitbrushExtractPalette", js.FuncOf(extractPalette))
	js.Global().Set("bitbrushGenPalette", js.FuncOf(genPalette))
}

func okData(pix []byte) map[string]any {
	dst := uint8Array.New(len(pix))
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

// bitbrushRenderNoiseField(paramsJSON string, w int, h int) ->
// {ok, data: Uint8Array|null, error}. Param parsing (enum names, hex
// colours) lives in internal/noisefield so the compositor can share it.
func renderNoiseField(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "data": js.Null(), "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	img, err := noisefield.RenderJSON([]byte(args[0].String()), args[1].Int(), args[2].Int())
	if err != nil {
		return errData(err)
	}
	return okData(img.Pix)
}

// bitbrushRenderNoiseFieldGIF(startJSON string, endJSON string, w int, h int, optionsJSON string) ->
// {ok, data: Uint8Array|null, error}.
func renderNoiseFieldGIF(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "data": js.Null(), "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	startP, err := noisefield.ParseParams([]byte(args[0].String()))
	if err != nil {
		return errData(fmt.Errorf("bad start params: %w", err))
	}
	endP, err := noisefield.ParseParams([]byte(args[1].String()))
	if err != nil {
		return errData(fmt.Errorf("bad end params: %w", err))
	}

	w, h := args[2].Int(), args[3].Int()

	var optJSON struct {
		Frames       int  `json:"frames"`
		FPS          int  `json:"fps"`
		Loop         bool `json:"loop"`
		PingPong     bool `json:"pingPong"`
		MaxDimension int  `json:"maxDimension"`
	}
	if err := json.Unmarshal([]byte(args[4].String()), &optJSON); err != nil {
		return errData(fmt.Errorf("bad options: %w", err))
	}

	data, err := anim.RenderNoiseField(startP, endP, anim.Options{
		Frames:       optJSON.Frames,
		FPS:          optJSON.FPS,
		LoopForever:  optJSON.Loop,
		PingPong:     optJSON.PingPong,
		MaxDimension: optJSON.MaxDimension,
	}, w, h)
	if err != nil {
		return errData(err)
	}

	dst := uint8Array.New(len(data))
	js.CopyBytesToJS(dst, data)
	return map[string]any{"ok": true, "data": dst, "error": ""}
}

// bitbrushRenderGenerator(name string, paramsJSON string, w int, h int) ->
// {ok, data: Uint8Array|null, error}. Dispatches to the internal/generators
// registry; paramsJSON is handed to the named generator untouched (each one
// defines and unmarshals its own params struct). Enums cross as lower-case
// name strings, same as the other generator globals.
func renderGenerator(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "data": js.Null(), "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	name := args[0].String()
	var raw json.RawMessage
	if args[1].Type() == js.TypeString {
		raw = json.RawMessage(args[1].String())
	}
	img, err := generators.Render(name, raw, args[2].Int(), args[3].Int())
	if err != nil {
		return errData(err)
	}
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
		Count          int     `json:"count"`
		Method         string  `json:"method"`
		Space          string  `json:"space"`
		Sort           string  `json:"sort"`
		AlphaThreshold int     `json:"alphaThreshold"`
		Seed           float64 `json:"seed"`
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
		Seed:           int64(math.Round(o.Seed)),
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
