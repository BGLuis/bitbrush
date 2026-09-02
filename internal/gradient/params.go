package gradient

import (
	"encoding/json"
	"fmt"
	"image"
	"strconv"
	"strings"
)

// maxDim caps a rendered dimension, bounding memory at ~1 GiB of RGBA.
const maxDim = 1 << 14

// ParamStop is the plain-data form of a Stop: colour as "#rrggbb".
type ParamStop struct {
	Color string  `json:"color"`
	Pos   float64 `json:"pos"`
}

// Params is a plain-data mirror of a Gradient plus render settings, shaped
// so it JSON-decodes cleanly across the WASM boundary. Enums are lower-case
// names ("oklch", "longer", ...); colours are hex strings.
type Params struct {
	Stops  []ParamStop `json:"stops"`
	Space  string      `json:"space"`
	Hue    string      `json:"hue"`
	Easing string      `json:"easing"`
	Width  int         `json:"width"`
	Height int         `json:"height"`
	Angle  float64     `json:"angle"`
}

// ParseParams decodes JSON into Params.
func ParseParams(data []byte) (Params, error) {
	var p Params
	if err := json.Unmarshal(data, &p); err != nil {
		return Params{}, fmt.Errorf("gradient: parse params: %w", err)
	}
	return p, nil
}

// Gradient builds a *Gradient from the params, resolving the colour
// strings, the space and hue-arc names, and the easing expression. An empty
// Easing string means Linear{}.
func (p Params) Gradient() (*Gradient, error) {
	if len(p.Stops) == 0 {
		return nil, fmt.Errorf("gradient: params have no stops")
	}
	stops := make([]Stop, len(p.Stops))
	for i, s := range p.Stops {
		c, err := ParseHex(s.Color)
		if err != nil {
			return nil, err
		}
		stops[i] = Stop{Color: c, Pos: s.Pos}
	}
	space, err := parseSpace(p.Space)
	if err != nil {
		return nil, err
	}
	arc, err := parseHueArc(p.Hue)
	if err != nil {
		return nil, err
	}
	var easing Easing
	if strings.TrimSpace(p.Easing) != "" {
		easing, err = ParseEasing(p.Easing)
		if err != nil {
			return nil, err
		}
	}
	return New(stops, space, arc, easing), nil
}

// Generate renders g into a new opaque RGBA of size w x h with a linear
// gradient along angleDeg. It is the single entry point the WASM adapter
// calls.
func Generate(g *Gradient, w, h int, angleDeg float64) (*image.RGBA, error) {
	if g == nil {
		return nil, fmt.Errorf("gradient: nil gradient")
	}
	if len(g.Stops) == 0 {
		return nil, fmt.Errorf("gradient: no stops")
	}
	if w < 1 || h < 1 {
		return nil, fmt.Errorf("gradient: bad size %dx%d", w, h)
	}
	if w > maxDim || h > maxDim {
		return nil, fmt.Errorf("gradient: size %dx%d exceeds %d", w, h, maxDim)
	}
	return g.Render(w, h, angleDeg), nil
}

// GenerateJSON is a convenience wrapper: decode Params, build the gradient,
// and render at the params' Width/Height/Angle (each with a sane default).
func GenerateJSON(data []byte) (*image.RGBA, error) {
	p, err := ParseParams(data)
	if err != nil {
		return nil, err
	}
	g, err := p.Gradient()
	if err != nil {
		return nil, err
	}
	w, h := p.Width, p.Height
	if w < 1 {
		w = 512
	}
	if h < 1 {
		h = 512
	}
	return Generate(g, w, h, p.Angle)
}

func parseSpace(s string) (Space, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "oklab":
		return SpaceOKLab, nil
	case "srgb", "rgb":
		return SpaceSRGB, nil
	case "linear-rgb", "linearrgb", "srgb-linear", "linear":
		return SpaceLinearRGB, nil
	case "hsl":
		return SpaceHSL, nil
	case "lab":
		return SpaceLab, nil
	case "lch":
		return SpaceLCh, nil
	case "oklch":
		return SpaceOKLCh, nil
	}
	return 0, fmt.Errorf("gradient: unknown space %q", s)
}

func parseHueArc(s string) (HueArc, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "shorter":
		return HueShorter, nil
	case "longer":
		return HueLonger, nil
	case "increasing":
		return HueIncreasing, nil
	case "decreasing":
		return HueDecreasing, nil
	}
	return 0, fmt.Errorf("gradient: unknown hue arc %q", s)
}

// ParseHex parses "#rgb" or "#rrggbb" (the leading "#" is optional) into an
// RGB.
func ParseHex(s string) (RGB, error) {
	t := strings.TrimPrefix(strings.TrimSpace(s), "#")
	switch len(t) {
	case 3:
		r, err1 := strconv.ParseUint(t[0:1], 16, 8)
		g, err2 := strconv.ParseUint(t[1:2], 16, 8)
		b, err3 := strconv.ParseUint(t[2:3], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return RGB{}, fmt.Errorf("gradient: bad hex colour %q", s)
		}
		return RGB{uint8(r * 17), uint8(g * 17), uint8(b * 17)}, nil
	case 6:
		v, err := strconv.ParseUint(t, 16, 32)
		if err != nil {
			return RGB{}, fmt.Errorf("gradient: bad hex colour %q", s)
		}
		return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}, nil
	}
	return RGB{}, fmt.Errorf("gradient: bad hex colour %q", s)
}

// Hex renders c as "#rrggbb".
func (c RGB) Hex() string { return hex(c) }
