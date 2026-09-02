// Package anim renders a BitBrush filter across many frames while
// interpolating its parameters between keyframes, then encodes the result
// as an animated GIF.
//
// It is the pure-Go algorithm core for the animated-GIF feature: no
// syscall/js, no build tags, no time.Now() and no unseeded randomness, so
// the same inputs always produce a byte-identical GIF and every function
// here is exercised by plain `go test ./internal/anim/...`.
package anim

import (
	"math"
	"sort"

	"bitbrush/internal/filters"
)

// Keyframe is a full params snapshot pinned at normalized time T in [0,1].
type Keyframe struct {
	T      float64
	Params filters.Params
}

// Options configures a render.
type Options struct {
	Frames       int  // frames to render, clamped to >= 2
	FPS          int  // clamped to 1..50; becomes a GIF delay of round(100/FPS) centiseconds (min 1)
	LoopForever  bool // true -> GIF LoopCount 0; false -> LoopCount -1 (play once)
	PingPong     bool // play A..B then B..A within Frames
	MaxDimension int  // downscale the longest side to this before rendering; 0 = keep size
}

const (
	minFrames = 2
	minFPS    = 1
	maxFPS    = 50
)

func (o Options) frameCount() int {
	if o.Frames < minFrames {
		return minFrames
	}
	return o.Frames
}

func (o Options) fps() int {
	f := o.FPS
	if f < minFPS {
		return minFPS
	}
	if f > maxFPS {
		return maxFPS
	}
	return f
}

// delayCentiseconds is the per-frame GIF delay: round(100/FPS), never below 1.
func (o Options) delayCentiseconds() int {
	d := int(math.Round(100.0 / float64(o.fps())))
	if d < 1 {
		d = 1
	}
	return d
}

// loopCount maps LoopForever to the image/gif convention: 0 loops forever,
// -1 plays every frame exactly once.
func (o Options) loopCount() int {
	if o.LoopForever {
		return 0
	}
	return -1
}

// phaseAt returns the effective interpolation phase in [0,1] for frame i of
// n. Without pingPong it ramps 0->1 linearly; with pingPong it is a
// triangle wave that rises 0->1 by the middle frame and falls back to 0,
// so the frame sequence is symmetric.
func phaseAt(i, n int, pingPong bool) float64 {
	if n <= 1 {
		return 0
	}
	if i < 0 {
		i = 0
	}
	if i > n-1 {
		i = n - 1
	}
	if pingPong {
		// Triangle wave folded around the middle frame: distance from the
		// centre is |i-mid|, identical for i and n-1-i, so the sequence is
		// exactly symmetric with no floating-point drift.
		mid := float64(n-1) / 2
		return 1 - math.Abs(float64(i)-mid)/mid
	}
	return float64(i) / float64(n-1)
}

// sortedKeyframes returns a copy of kfs sorted by T with each T clamped to
// [0,1]. The caller's slice is never mutated.
func sortedKeyframes(kfs []Keyframe) []Keyframe {
	out := make([]Keyframe, len(kfs))
	copy(out, kfs)
	for i := range out {
		out[i].T = clamp01(out[i].T)
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].T < out[b].T })
	return out
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// Interpolate returns the params for frame i of n (i in [0,n-1]).
//
// Numeric params (Go int- or float-kinded) are linearly interpolated
// between the two keyframes that bracket the frame's phase; when both
// bracketing values are int-kinded the result is a rounded int, otherwise
// it is a float64. Non-numeric params (string/bool/other) snap to the
// nearer bracketing keyframe. A key that only some keyframes carry is
// bracketed among just those keyframes, so it stays constant outside their
// range rather than being extrapolated. With pingPong the phase runs
// 0->1->0 across the n frames.
func Interpolate(kfs []Keyframe, i, n int, pingPong bool) filters.Params {
	if len(kfs) == 0 {
		return filters.Params{}
	}
	sorted := sortedKeyframes(kfs)
	if len(sorted) == 1 {
		return clineParams(sorted[0].Params)
	}

	t := phaseAt(i, n, pingPong)
	out := make(filters.Params)

	for _, key := range unionKeys(sorted) {
		ts, vs := valuesFor(sorted, key)
		switch len(vs) {
		case 0:
			continue
		case 1:
			out[key] = vs[0]
			continue
		}

		lo, hi := bracket(ts, t)
		if lo == hi {
			out[key] = vs[lo]
			continue
		}
		span := ts[hi] - ts[lo]
		local := 0.0
		if span > 0 {
			local = (t - ts[lo]) / span
		}
		out[key] = blend(vs[lo], vs[hi], local)
	}
	return out
}

// clineParams is a shallow copy of p (values are scalars, so shallow is
// enough) that callers may mutate freely.
func clineParams(p filters.Params) filters.Params {
	out := make(filters.Params, len(p))
	for k, v := range p {
		out[k] = v
	}
	return out
}

// unionKeys returns every param key that appears in any keyframe, sorted
// for deterministic iteration.
func unionKeys(kfs []Keyframe) []string {
	seen := make(map[string]struct{})
	for _, kf := range kfs {
		for k := range kf.Params {
			seen[k] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// valuesFor collects (T, value) pairs for key across the keyframes that
// carry it, preserving the keyframes' T order.
func valuesFor(kfs []Keyframe, key string) (ts []float64, vs []any) {
	for _, kf := range kfs {
		if v, ok := kf.Params[key]; ok {
			ts = append(ts, kf.T)
			vs = append(vs, v)
		}
	}
	return ts, vs
}

// bracket finds the indices into ts (ascending) that surround t. t at or
// before the first entry brackets to (0,0); at or after the last brackets
// to (last,last); otherwise lo is the last index with ts[lo] <= t.
func bracket(ts []float64, t float64) (lo, hi int) {
	if t <= ts[0] {
		return 0, 0
	}
	last := len(ts) - 1
	if t >= ts[last] {
		return last, last
	}
	for i := 0; i < last; i++ {
		if t >= ts[i] && t <= ts[i+1] {
			return i, i + 1
		}
	}
	return last, last
}

// blend interpolates a from->b by local in [0,1]. Numeric pairs lerp
// (rounded int when both sides are int-kinded, float64 otherwise);
// everything else snaps to the nearer end.
func blend(a, b any, local float64) any {
	af, aNum := toFloat(a)
	bf, bNum := toFloat(b)
	if aNum && bNum {
		v := af + (bf-af)*local
		if isIntKind(a) && isIntKind(b) {
			return int(math.Round(v))
		}
		return v
	}
	if local < 0.5 {
		return a
	}
	return b
}

// toFloat reports whether v is a Go numeric scalar and returns its float64
// value.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	default:
		return 0, false
	}
}

// isIntKind reports whether v is an integer-kinded Go scalar (so an
// interpolated result should stay whole).
func isIntKind(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}
