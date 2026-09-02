package noisefield

import (
	"math"

	"bitbrush/internal/colorspace"
)

// linRGB is a stop colour held in linear-light RGB, [0,1] per channel.
type linRGB struct{ r, g, b float64 }

// linearizeStops converts the 8-bit sRGB ramp to linear light once, up
// front. An empty ramp is treated as a single mid-grey stop.
func linearizeStops(stops []RGB) []linRGB {
	if len(stops) == 0 {
		stops = []RGB{{128, 128, 128}}
	}
	out := make([]linRGB, len(stops))
	for i, s := range stops {
		out[i] = linRGB{
			colorspace.LinearizeChannel(float64(s.R) / 255.0),
			colorspace.LinearizeChannel(float64(s.G) / 255.0),
			colorspace.LinearizeChannel(float64(s.B) / 255.0),
		}
	}
	return out
}

// paletteSRGB samples the ramp at t in [0,1] with piecewise-linear
// interpolation. Unlike the shader, which lerps raw sRGB, the interpolation
// runs in linear-light RGB and is re-encoded to sRGB on the way out — a
// small quality win that removes the muddy midpoint of an sRGB lerp.
//
// TODO: optional OKLab ramp — interpolate through OKLab for perceptually
// even stops once colorspace exposes the round trip.
func paletteSRGB(lin []linRGB, t float64) vec3 {
	var l linRGB
	if len(lin) == 1 {
		l = lin[0]
	} else {
		t = clampf(t, 0.0, 1.0)
		f := t * float64(len(lin)-1)
		i := int(math.Floor(f))
		if i < 0 {
			i = 0
		}
		if i > len(lin)-2 {
			i = len(lin) - 2
		}
		fr := f - float64(i)
		a, b := lin[i], lin[i+1]
		l = linRGB{
			mix(a.r, b.r, fr),
			mix(a.g, b.g, fr),
			mix(a.b, b.b, fr),
		}
	}
	return vec3{
		colorspace.EncodeChannel(l.r),
		colorspace.EncodeChannel(l.g),
		colorspace.EncodeChannel(l.b),
	}
}
