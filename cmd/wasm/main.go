// Command wasm is the only part of BitBrush that talks to the browser.
// It registers functions on globalThis and delegates all real work to
// the pure-Go packages under internal/.
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"runtime/debug"
	"syscall/js"

	"bitbrush/internal/anim"
	"bitbrush/internal/filters"
)

func main() {
	// This runtime is single-threaded: GC mark/assist work runs on the same
	// thread doing pixel work, so a collection mid-drag is visible jank. Each
	// filter call churns tens of MB of scratch; at the default GOGC=100 that
	// triggers a cycle almost every frame. Trade resident memory for far
	// fewer cycles, with a soft ceiling so a huge export still stays bounded.
	debug.SetGCPercent(300)
	debug.SetMemoryLimit(768 << 20)

	uint8Array = js.Global().Get("Uint8Array")

	js.Global().Set("bitbrushApplyFilter", js.FuncOf(applyFilter))
	js.Global().Set("bitbrushApplyPipeline", js.FuncOf(applyPipeline))
	js.Global().Set("bitbrushAsciiText", js.FuncOf(asciiText))
	js.Global().Set("bitbrushRenderGIF", js.FuncOf(renderGIF))
	registerGenerators()
	registerCompositor()
	select {} // block forever so the registered funcs stay callable
}

// uint8Array is globalThis.Uint8Array, resolved once in main. Looking it up
// per call (js.Global().Get) is two boundary crossings on every op.
var uint8Array js.Value

// ioBuf is a single reusable staging buffer for RGBA bytes coming across the
// boundary, grown on demand and never shrunk. Safe because calls are
// serialised (one WASM instance, one thread) and decodeImage copies the JS
// bytes straight into a fresh *image.RGBA-shaped view of it.
var ioBuf []byte

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

	dst := uint8Array.New(len(out.Pix))
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

	dst := uint8Array.New(len(data))
	js.CopyBytesToJS(dst, data)
	return map[string]any{"ok": true, "data": dst, "error": ""}
}

// decodeImage copies a Uint8Array of RGBA bytes into a Go-owned *image.RGBA
// backed by the shared ioBuf. The caller must finish with the returned image
// (or copy what it needs out) before the next boundary call reuses ioBuf.
func decodeImage(rgbaJS, wJS, hJS js.Value) (*image.RGBA, error) {
	w, h := wJS.Int(), hJS.Int()
	n := rgbaJS.Get("length").Int()
	if n != w*h*4 {
		return nil, fmt.Errorf("expected %d bytes for a %dx%d RGBA image, got %d", w*h*4, w, h, n)
	}
	if cap(ioBuf) < n {
		ioBuf = make([]byte, n)
	}
	ioBuf = ioBuf[:n]
	js.CopyBytesToGo(ioBuf, rgbaJS)
	return &image.RGBA{Pix: ioBuf, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}, nil
}

// decodeImageFresh copies a Uint8Array of RGBA bytes into a newly allocated
// *image.RGBA that does NOT alias ioBuf. Use it whenever an operation needs
// more than one image live at once (ioBuf backs only one).
func decodeImageFresh(rgbaJS, wJS, hJS js.Value) (*image.RGBA, error) {
	w, h := wJS.Int(), hJS.Int()
	n := rgbaJS.Get("length").Int()
	if n != w*h*4 {
		return nil, fmt.Errorf("expected %d bytes for a %dx%d RGBA image, got %d", w*h*4, w, h, n)
	}
	buf := make([]byte, n)
	js.CopyBytesToGo(buf, rgbaJS)
	return &image.RGBA{Pix: buf, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}, nil
}

// decodeImageList slices one concatenated Uint8Array of RGBA bytes into a
// list of freshly allocated images using dims [[w,h],...]. Each image owns
// its backing slice — no ioBuf aliasing.
func decodeImageList(concatJS js.Value, dimsJSON string) ([]*image.RGBA, error) {
	var dims [][2]int
	if err := json.Unmarshal([]byte(dimsJSON), &dims); err != nil {
		return nil, fmt.Errorf("bad image dims: %w", err)
	}
	total := concatJS.Get("length").Int()
	whole := make([]byte, total)
	js.CopyBytesToGo(whole, concatJS)

	out := make([]*image.RGBA, len(dims))
	off := 0
	for i, d := range dims {
		w, h := d[0], d[1]
		n := w * h * 4
		if w <= 0 || h <= 0 || off+n > total {
			return nil, fmt.Errorf("image %d: bad dims %dx%d at offset %d of %d bytes", i, w, h, off, total)
		}
		buf := make([]byte, n)
		copy(buf, whole[off:off+n])
		out[i] = &image.RGBA{Pix: buf, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
		off += n
	}
	return out, nil
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
