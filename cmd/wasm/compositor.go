package main

import (
	"encoding/json"
	"fmt"
	"image"
	"syscall/js"

	"bitbrush/internal/compositor"
)

// This file registers the layer-stack compositor global. Same
// {ok,...,error} shape and panic-recovery discipline as main.go; no
// algorithm logic — internal/compositor does the work.

func registerCompositor() {
	js.Global().Set("bitbrushRenderComposite", js.FuncOf(renderComposite))
}

// okImage is okData plus the result dimensions, matching the worker reply
// shape used by the other whole-image globals.
func okImage(img *image.RGBA) map[string]any {
	b := img.Bounds()
	dst := uint8Array.New(len(img.Pix))
	js.CopyBytesToJS(dst, img.Pix)
	return map[string]any{"ok": true, "data": dst, "width": b.Dx(), "height": b.Dy(), "error": ""}
}

func errComposite(err error) map[string]any {
	return map[string]any{"ok": false, "data": js.Null(), "width": 0, "height": 0, "error": err.Error()}
}

// bitbrushRenderComposite(
//
//	baseRGBA       Uint8Array | null,
//	baseW, baseH   int,
//	specJSON       string,              // compositor.Spec (carries width/height)
//	extrasRGBA     Uint8Array,          // image-layer buffers, concatenated (length 0 if none)
//	extrasDimsJSON string,              // "[[w,h],...]" in imageIndex order
//
// ) -> { ok, data: Uint8Array|null, width, height, error }.
func renderComposite(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{
				"ok": false, "data": js.Null(), "width": 0, "height": 0,
				"error": fmt.Sprintf("panic: %v", r),
			}
		}
	}()

	var base *image.RGBA
	if t := args[0].Type(); t != js.TypeNull && t != js.TypeUndefined {
		b, err := decodeImageFresh(args[0], args[1], args[2])
		if err != nil {
			return errComposite(err)
		}
		base = b
	}

	var spec compositor.Spec
	if err := json.Unmarshal([]byte(args[3].String()), &spec); err != nil {
		return errComposite(fmt.Errorf("bad spec: %w", err))
	}

	var extras []*image.RGBA
	if args[4].Truthy() && args[4].Get("length").Int() > 0 {
		xs, err := decodeImageList(args[4], args[5].String())
		if err != nil {
			return errComposite(err)
		}
		extras = xs
	}

	out, err := compositor.Evaluate(spec, base, extras)
	if err != nil {
		return errComposite(err)
	}
	return okImage(out)
}
