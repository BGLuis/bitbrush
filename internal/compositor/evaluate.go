package compositor

import (
	"encoding/json"
	"fmt"
	"image"

	"bitbrush/internal/filters"
	"bitbrush/internal/generators"
	"bitbrush/internal/gradient"
	"bitbrush/internal/mask"
	"bitbrush/internal/noisefield"
	"bitbrush/internal/text"
)

// maxDim caps a rendered canvas dimension, matching internal/gradient.
const maxDim = 1 << 14

// SourceKind names where a layer's pixels come from. Values cross the WASM
// boundary and sit in URL recipes verbatim.
type SourceKind string

const (
	SourceBase       SourceKind = "base"       // the primary image (passed to Evaluate as base)
	SourceImage      SourceKind = "image"      // an extra uploaded image, by ImageIndex into extras
	SourceGenerator  SourceKind = "generator"  // internal/generators registry, by Generator name
	SourceGradient   SourceKind = "gradient"   // internal/gradient, from GenParams
	SourceNoiseField SourceKind = "noisefield" // internal/noisefield, from GenParams
	SourceText       SourceKind = "text"       // internal/text, from GenParams
)

// Stage is one entry in a layer's own ordered filter chain.
type Stage struct {
	Filter string         `json:"filter"`
	Params filters.Params `json:"params"`
	Mask   *mask.Mask     `json:"mask,omitempty"`
}

// Layer is one entry in the stack, composited over everything below it.
type Layer struct {
	Enabled    bool            `json:"enabled"`
	Source     SourceKind      `json:"source"`
	ImageIndex int             `json:"imageIndex,omitempty"` // Source == "image"
	Generator  string          `json:"generator,omitempty"`  // Source == "generator"
	GenParams  json.RawMessage `json:"genParams,omitempty"`  // generator / gradient / noisefield params
	Fit        FitMode         `json:"fit,omitempty"`        // default cover
	Transform  *Transform      `json:"transform,omitempty"`  // move/scale/rotate after Fit
	Chain      []Stage         `json:"chain,omitempty"`
	Blend      BlendMode       `json:"blend"`
	Opacity    float64         `json:"opacity"` // 0..1
	// Mask restricts this layer's composite over what's below it. Unlike
	// Stage.Mask (which blends a filter stage's before/after within one
	// layer), this blends "nothing below the layer" against "the layer
	// composited over what's below" — so a KindLuma mask samples the
	// LUMINANCE OF WHAT'S BELOW the layer, not the layer's own pixels.
	Mask *mask.Mask `json:"mask,omitempty"`
}

// Spec is a full compose recipe. Width/Height override the base image size;
// when both are zero the base image's size is used.
type Spec struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Layers []Layer `json:"layers"`
}

// Evaluate composites spec's layers bottom-to-top over a transparent canvas
// and returns the final straight-alpha RGBA. base is the primary image (may
// be nil for a purely generative stack); extras[i] backs a layer with
// Source "image" and ImageIndex i. Deterministic: the same (spec, base,
// extras) always yields byte-identical output.
func Evaluate(spec Spec, base *image.RGBA, extras []*image.RGBA) (*image.RGBA, error) {
	w, h := spec.Width, spec.Height
	if w <= 0 || h <= 0 {
		if base == nil {
			return nil, fmt.Errorf("compositor: no canvas size and no base image")
		}
		b := base.Bounds()
		w, h = b.Dx(), b.Dy()
	}
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("compositor: bad canvas size %dx%d", w, h)
	}
	if w > maxDim || h > maxDim {
		return nil, fmt.Errorf("compositor: canvas %dx%d exceeds %d", w, h, maxDim)
	}

	acc := image.NewRGBA(image.Rect(0, 0, w, h))

	for i := range spec.Layers {
		layer := spec.Layers[i]
		if !layer.Enabled {
			continue
		}

		src, err := resolveSource(layer, base, extras, w, h)
		if err != nil {
			return nil, fmt.Errorf("compositor: layer %d: %w", i, err)
		}
		if b := src.Bounds(); b.Dx() != w || b.Dy() != h {
			src = Fit(src, w, h, layer.Fit)
		}
		if layer.Transform != nil && !layer.Transform.IsIdentity() {
			src = ApplyTransform(src, *layer.Transform, w, h)
		}

		cur := src
		for j, st := range layer.Chain {
			filtered, err := filters.Apply(st.Filter, cur, st.Params)
			if err != nil {
				return nil, fmt.Errorf("compositor: layer %d stage %d (%q): %w", i, j, st.Filter, err)
			}
			if st.Mask != nil {
				cur = mask.Compose(cur, filtered, st.Mask)
			} else {
				cur = filtered
			}
		}

		next, err := Composite(acc, cur, layer.Blend, layer.Opacity)
		if err != nil {
			return nil, fmt.Errorf("compositor: layer %d: %w", i, err)
		}
		if layer.Mask != nil {
			acc = mask.Compose(acc, next, layer.Mask)
		} else {
			acc = next
		}
	}
	return acc, nil
}

func resolveSource(layer Layer, base *image.RGBA, extras []*image.RGBA, w, h int) (*image.RGBA, error) {
	switch layer.Source {
	case SourceBase:
		if base == nil {
			return nil, fmt.Errorf("source %q but no base image", layer.Source)
		}
		return cloneRGBA(base), nil

	case SourceImage:
		if layer.ImageIndex < 0 || layer.ImageIndex >= len(extras) || extras[layer.ImageIndex] == nil {
			return nil, fmt.Errorf("image index %d out of range (%d extras)", layer.ImageIndex, len(extras))
		}
		return cloneRGBA(extras[layer.ImageIndex]), nil

	case SourceGenerator:
		return generators.Render(layer.Generator, layer.GenParams, w, h)

	case SourceGradient:
		p, err := gradient.ParseParams(nonNilJSON(layer.GenParams))
		if err != nil {
			return nil, err
		}
		p.Width, p.Height = w, h
		g, err := p.Gradient()
		if err != nil {
			return nil, err
		}
		return gradient.Generate(g, w, h, p.Angle)

	case SourceNoiseField:
		return noisefield.RenderJSON(nonNilJSON(layer.GenParams), w, h)

	case SourceText:
		var p text.Params
		if err := json.Unmarshal(nonNilJSON(layer.GenParams), &p); err != nil {
			return nil, fmt.Errorf("source text: bad params: %w", err)
		}
		return text.Render(p, w, h)
	}
	return nil, fmt.Errorf("unknown source %q", layer.Source)
}

func nonNilJSON(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	return raw
}
