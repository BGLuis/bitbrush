// Command wasm is the only part of BitBrush that talks to the browser.
// It registers functions on globalThis and delegates all real work to
// the pure-Go packages under internal/.
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"syscall/js"

	"bitbrush/internal/anim"
	"bitbrush/internal/filters"
)

func main() {
	js.Global().Set("bitbrushApplyFilter", js.FuncOf(applyFilter))
	js.Global().Set("bitbrushAsciiText", js.FuncOf(asciiText))
	js.Global().Set("bitbrushRenderGIF", js.FuncOf(renderGIF))
	registerGenerators()
	select {} // block forever so the registered funcs stay callable
}

// applyFilter(name string, rgba Uint8Array, w int, h int, paramsJSON string)
// returns { ok: bool, data: Uint8Array|null, error: string }.
func applyFilter(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "data": js.Null(), "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	src, err := decodeImage(args[1], args[2], args[3])
	if err != nil {
		return map[string]any{"ok": false, "data": js.Null(), "error": err.Error()}
	}
	params, err := decodeParams(args, 4)
	if err != nil {
		return map[string]any{"ok": false, "data": js.Null(), "error": err.Error()}
	}

	out, err := filters.Apply(args[0].String(), src, params)
	if err != nil {
		return map[string]any{"ok": false, "data": js.Null(), "error": err.Error()}
	}

	dst := js.Global().Get("Uint8Array").New(len(out.Pix))
	js.CopyBytesToJS(dst, out.Pix)
	return map[string]any{"ok": true, "data": dst, "error": ""}
}

// asciiText(rgba Uint8Array, w int, h int, paramsJSON string)
// returns { ok: bool, text: string, error: string }. Separate global
// because its result is a string, not pixels — filters.ASCIIText isn't a
// registered Filter for the same reason.
func asciiText(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "text": "", "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	src, err := decodeImage(args[0], args[1], args[2])
	if err != nil {
		return map[string]any{"ok": false, "text": "", "error": err.Error()}
	}
	params, err := decodeParams(args, 3)
	if err != nil {
		return map[string]any{"ok": false, "text": "", "error": err.Error()}
	}

	text, err := filters.ASCIIText(src, params)
	if err != nil {
		return map[string]any{"ok": false, "text": "", "error": err.Error()}
	}
	return map[string]any{"ok": true, "text": text, "error": ""}
}

// renderGIF(name string, rgba Uint8Array, w int, h int, keyframesJSON string,
// optionsJSON string) returns { ok, data: Uint8Array|null, error }, where
// data is an encoded animated GIF. keyframesJSON is [{t, params}, ...] and
// optionsJSON is {frames, fps, loop, pingPong, maxDimension}.
func renderGIF(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "data": js.Null(), "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	src, err := decodeImage(args[1], args[2], args[3])
	if err != nil {
		return map[string]any{"ok": false, "data": js.Null(), "error": err.Error()}
	}

	var kfsJSON []struct {
		T      float64        `json:"t"`
		Params filters.Params `json:"params"`
	}
	if err := json.Unmarshal([]byte(args[4].String()), &kfsJSON); err != nil {
		return map[string]any{"ok": false, "data": js.Null(), "error": fmt.Sprintf("bad keyframes: %v", err)}
	}
	var optJSON struct {
		Frames       int  `json:"frames"`
		FPS          int  `json:"fps"`
		Loop         bool `json:"loop"`
		PingPong     bool `json:"pingPong"`
		MaxDimension int  `json:"maxDimension"`
	}
	if err := json.Unmarshal([]byte(args[5].String()), &optJSON); err != nil {
		return map[string]any{"ok": false, "data": js.Null(), "error": fmt.Sprintf("bad options: %v", err)}
	}

	kfs := make([]anim.Keyframe, len(kfsJSON))
	for i, k := range kfsJSON {
		kfs[i] = anim.Keyframe{T: k.T, Params: k.Params}
	}

	data, err := anim.Render(src, args[0].String(), kfs, anim.Options{
		Frames:       optJSON.Frames,
		FPS:          optJSON.FPS,
		LoopForever:  optJSON.Loop,
		PingPong:     optJSON.PingPong,
		MaxDimension: optJSON.MaxDimension,
	})
	if err != nil {
		return map[string]any{"ok": false, "data": js.Null(), "error": err.Error()}
	}

	dst := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(dst, data)
	return map[string]any{"ok": true, "data": dst, "error": ""}
}

// decodeImage copies a Uint8Array of RGBA bytes into a Go-owned *image.RGBA.
func decodeImage(rgbaJS, wJS, hJS js.Value) (*image.RGBA, error) {
	w, h := wJS.Int(), hJS.Int()
	buf := make([]byte, rgbaJS.Get("length").Int())
	js.CopyBytesToGo(buf, rgbaJS)
	if len(buf) != w*h*4 {
		return nil, fmt.Errorf("expected %d bytes for a %dx%d RGBA image, got %d", w*h*4, w, h, len(buf))
	}
	return &image.RGBA{Pix: buf, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}, nil
}

// decodeParams reads the optional JSON-object argument at args[i], if any.
func decodeParams(args []js.Value, i int) (filters.Params, error) {
	var params filters.Params
	if len(args) > i && args[i].Type() == js.TypeString {
		if err := json.Unmarshal([]byte(args[i].String()), &params); err != nil {
			return nil, fmt.Errorf("bad params: %w", err)
		}
	}
	return params, nil
}
