package compositor

import "math"

// BlendMode names a separable blend function B(cb, cs), applied per channel
// on gamma-encoded sRGB values in [0,1] (W3C Compositing and Blending L1
// §9). Names cross the WASM boundary and sit in URL recipes verbatim.
type BlendMode string

const (
	BlendNormal      BlendMode = "normal"
	BlendMultiply    BlendMode = "multiply"
	BlendScreen      BlendMode = "screen"
	BlendOverlay     BlendMode = "overlay"
	BlendDarken      BlendMode = "darken"
	BlendLighten     BlendMode = "lighten"
	BlendColorDodge  BlendMode = "color-dodge"
	BlendColorBurn   BlendMode = "color-burn"
	BlendHardLight   BlendMode = "hard-light"
	BlendSoftLight   BlendMode = "soft-light"
	BlendDifference  BlendMode = "difference"
	BlendExclusion   BlendMode = "exclusion"
	BlendLinearDodge BlendMode = "linear-dodge" // Photoshop "add"
	BlendSubtract    BlendMode = "subtract"     // Photoshop "subtract"
)

// blendFn returns the channel blend function for m and whether m was
// recognised. Callers fall back to Normal when ok is false so a recipe
// written by a newer build still renders.
func blendFn(m BlendMode) (fn func(cb, cs float64) float64, ok bool) {
	switch m {
	case BlendNormal, "":
		return func(_, cs float64) float64 { return cs }, true
	case BlendMultiply:
		return func(cb, cs float64) float64 { return cb * cs }, true
	case BlendScreen:
		return blendScreen, true
	case BlendOverlay:
		return func(cb, cs float64) float64 { return blendHardLight(cs, cb) }, true
	case BlendDarken:
		return math.Min, true
	case BlendLighten:
		return math.Max, true
	case BlendColorDodge:
		return blendColorDodge, true
	case BlendColorBurn:
		return blendColorBurn, true
	case BlendHardLight:
		return blendHardLight, true
	case BlendSoftLight:
		return blendSoftLight, true
	case BlendDifference:
		return func(cb, cs float64) float64 { return math.Abs(cb - cs) }, true
	case BlendExclusion:
		return func(cb, cs float64) float64 { return cb + cs - 2*cb*cs }, true
	case BlendLinearDodge:
		return func(cb, cs float64) float64 { return clamp01(cb + cs) }, true
	case BlendSubtract:
		return func(cb, cs float64) float64 { return clamp01(cb - cs) }, true
	}
	return nil, false
}

func blendScreen(cb, cs float64) float64 { return cb + cs - cb*cs }

func blendHardLight(cb, cs float64) float64 {
	if cs <= 0.5 {
		return cb * (2 * cs)
	}
	return blendScreen(cb, 2*cs-1)
}

func blendColorDodge(cb, cs float64) float64 {
	if cb == 0 {
		return 0
	}
	if cs >= 1 {
		return 1
	}
	return math.Min(1, cb/(1-cs))
}

func blendColorBurn(cb, cs float64) float64 {
	if cb >= 1 {
		return 1
	}
	if cs == 0 {
		return 0
	}
	return 1 - math.Min(1, (1-cb)/cs)
}

// blendSoftLight is the W3C soft-light function.
func blendSoftLight(cb, cs float64) float64 {
	if cs <= 0.5 {
		return cb - (1-2*cs)*cb*(1-cb)
	}
	var d float64
	if cb <= 0.25 {
		d = ((16*cb-12)*cb + 4) * cb
	} else {
		d = math.Sqrt(cb)
	}
	return cb + (2*cs-1)*(d-cb)
}
