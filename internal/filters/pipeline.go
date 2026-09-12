package filters

import (
	"fmt"
	"image"
)

// PipelineStage is one step in a filter chain: a named effect and its params.
// It mirrors compositor.Stage so JSON produced by the shell is accepted here
// without re-marshalling (same field names, same shape).
type PipelineStage struct {
	Filter string `json:"filter"`
	Params Params `json:"params"`
}

// Pipeline applies stages in order to src, returning the final RGBA image.
// It is the same loop used by the compositor's layer chain, extracted so
// callers can apply a multi-step chain without constructing a full Spec.
//
// src is never mutated. Each stage's output becomes the next stage's input.
// An empty stages slice returns a clone of src.
func Pipeline(src *image.RGBA, stages []PipelineStage) (*image.RGBA, error) {
	cur := cloneRGBA(src)
	for i, st := range stages {
		out, err := Apply(st.Filter, cur, st.Params)
		if err != nil {
			return nil, fmt.Errorf("pipeline stage %d (%q): %w", i, st.Filter, err)
		}
		cur = out
	}
	return cur, nil
}
