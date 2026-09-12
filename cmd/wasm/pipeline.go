package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"bitbrush/internal/filters"
)

// applyPipeline(rgba Uint8Array, w int, h int, chainJSON string)
// chainJSON: [{filter: string, params: object}, ...]
// Returns {ok: bool, data: Uint8Array|null, error: string}.
// The output image has the same dimensions as the input.
func applyPipeline(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]any{"ok": false, "data": js.Null(), "error": fmt.Sprintf("panic: %v", r)}
		}
	}()

	if len(args) < 4 {
		return map[string]any{"ok": false, "data": js.Null(), "error": "applyPipeline: need (rgba, w, h, chainJSON)"}
	}

	src, err := decodeImage(args[0], args[1], args[2])
	if err != nil {
		return map[string]any{"ok": false, "data": js.Null(), "error": err.Error()}
	}

	var stages []filters.PipelineStage
	if err := json.Unmarshal([]byte(args[3].String()), &stages); err != nil {
		return map[string]any{"ok": false, "data": js.Null(), "error": fmt.Sprintf("bad chain JSON: %v", err)}
	}

	out, err := filters.Pipeline(src, stages)
	if err != nil {
		return map[string]any{"ok": false, "data": js.Null(), "error": err.Error()}
	}

	dst := uint8Array.New(len(out.Pix))
	js.CopyBytesToJS(dst, out.Pix)
	return map[string]any{"ok": true, "data": dst, "error": ""}
}
