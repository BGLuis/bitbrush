package anim

import (
	"image"
	"math"

	"bitbrush/internal/noisefield"
)

// RenderNoiseFieldFrames renders N frames of a noisefield gradient by interpolating
// numeric parameters between start and end. w and h are scaled so the longest side
// is at most opt.MaxDimension (if opt.MaxDimension > 0).
func RenderNoiseFieldFrames(start, end noisefield.Params, opt Options, w, h int) ([]*image.RGBA, error) {
	nw, nh := scaleDimensions(w, h, opt.MaxDimension)
	if nw <= 0 || nh <= 0 {
		nw, nh = 360, 203 // sensible 16:9 fallback
	}

	n := opt.frameCount()
	frames := make([]*image.RGBA, 0, n)

	// Circular loop check: if angle spans 0..360 without ping-pong, avoid duplicate frame at end
	isAngle360 := !opt.PingPong && math.Abs(math.Abs(end.AngleDeg-start.AngleDeg)-360.0) < 0.001

	for i := 0; i < n; i++ {
		var t float64
		if opt.PingPong {
			t = phaseAt(i, n, true)
		} else if isAngle360 {
			t = float64(i) / float64(n)
		} else {
			t = phaseAt(i, n, false)
		}

		p := interpolateNoiseFieldParams(start, end, t)
		frame := noisefield.Render(p, nw, nh)
		frames = append(frames, frame)
	}

	return frames, nil
}

// RenderNoiseField is RenderNoiseFieldFrames followed by EncodeGIF.
func RenderNoiseField(start, end noisefield.Params, opt Options, w, h int) ([]byte, error) {
	frames, err := RenderNoiseFieldFrames(start, end, opt, w, h)
	if err != nil {
		return nil, err
	}
	return EncodeGIF(frames, opt)
}

// interpolateNoiseFieldParams blends numeric fields between start and end.
// Non-numeric fields (Field, Style, Texture, Stops, Spots) snap to start if t < 0.5, else end.
func interpolateNoiseFieldParams(start, end noisefield.Params, t float64) noisefield.Params {
	out := start
	if t >= 0.5 {
		out.Field = end.Field
		out.Style = end.Style
		out.Texture = end.Texture
		if len(end.Stops) > 0 {
			out.Stops = end.Stops
		}
		if len(end.Spots) > 0 {
			out.Spots = end.Spots
		}
	}

	out.AngleDeg = start.AngleDeg + (end.AngleDeg-start.AngleDeg)*t
	out.Scale = start.Scale + (end.Scale-start.Scale)*t
	out.Distortion = start.Distortion + (end.Distortion-start.Distortion)*t
	out.Time = start.Time + (end.Time-start.Time)*t

	if start.Seed == end.Seed {
		out.Seed = start.Seed
	} else {
		out.Seed = int64(math.Round(float64(start.Seed) + float64(end.Seed-start.Seed)*t))
	}

	return out
}

func scaleDimensions(w, h, maxDim int) (int, int) {
	if maxDim <= 0 || (w <= maxDim && h <= maxDim) {
		return w, h
	}
	longest := w
	if h > longest {
		longest = h
	}
	if longest <= 0 {
		return w, h
	}
	scale := float64(maxDim) / float64(longest)
	nw := int(math.Round(float64(w) * scale))
	nh := int(math.Round(float64(h) * scale))
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	return nw, nh
}
