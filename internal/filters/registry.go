// Package filters holds BitBrush's algorithmic image effects and a small
// name -> effect registry. Nothing here imports syscall/js; every effect
// is plain Go and covered by go test.
package filters

import (
	"fmt"
	"image"
)

// Filter transforms src into a new image. It must not mutate src.
type Filter func(src *image.RGBA, params Params) (*image.RGBA, error)

var registry = map[string]Filter{}

// Register adds an effect under name. Call it from a file-level init().
// It panics on duplicate names so collisions surface immediately.
func Register(name string, fn Filter) {
	if _, dup := registry[name]; dup {
		panic("filters: duplicate registration " + name)
	}
	registry[name] = fn
}

// Names returns the registered effect names (unordered).
func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	return out
}

// Apply runs the named effect against src.
func Apply(name string, src *image.RGBA, params Params) (*image.RGBA, error) {
	fn, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("filters: unknown effect %q", name)
	}
	return fn(src, params)
}
