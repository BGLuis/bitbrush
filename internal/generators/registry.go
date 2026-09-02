// Package generators holds BitBrush's algorithmic image generators — the
// ones that synthesise a picture from parameters alone, with no input image
// (strange attractors, harmonographs, Truchet tilings, contour maps, flow
// fields, L-systems, fractal flames, ...).
//
// It is the generator counterpart of internal/filters' registry: each
// generator package registers itself by string name from a file-level
// init(), and the WASM adapter dispatches to it through Render. Nothing here
// imports syscall/js; every generator is plain Go covered by go test.
//
// internal/gradient and internal/noisefield predate this registry and keep
// their own dedicated WASM globals; they may migrate here later.
package generators

import (
	"encoding/json"
	"fmt"
	"image"
	"sort"
)

// Generator synthesises a w x h opaque image from a JSON parameter blob.
// The blob is passed straight through from the caller — each generator
// defines and unmarshals its own params struct. A Generator must be a pure
// function of (params, w, h): anything stochastic takes an explicit seed.
type Generator func(params json.RawMessage, w, h int) (*image.RGBA, error)

var registry = map[string]Generator{}

// Register adds a generator under name. Call it from a file-level init().
// It panics on duplicate names so collisions surface immediately.
func Register(name string, fn Generator) {
	if _, dup := registry[name]; dup {
		panic("generators: duplicate registration " + name)
	}
	registry[name] = fn
}

// Names returns the registered generator names, sorted.
func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Render runs the named generator. params may be nil or empty; the
// generator is responsible for its own defaults.
func Render(name string, params json.RawMessage, w, h int) (*image.RGBA, error) {
	fn, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("generators: unknown generator %q", name)
	}
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("generators: bad size %dx%d", w, h)
	}
	img, err := fn(params, w, h)
	if err != nil {
		return nil, err
	}
	if img == nil {
		return nil, fmt.Errorf("generators: %q returned a nil image", name)
	}
	return img, nil
}
