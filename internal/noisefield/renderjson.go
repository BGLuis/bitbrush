package noisefield

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"
)

// fieldByName / styleByName / textureByName map the lower-case enum names
// used across the WASM boundary and in URL recipes to their typed values.
// An unknown name resolves to the zero value (Linear / Metallic / Smooth).
var fieldByName = map[string]Field{
	"linear": FieldLinear, "radial": FieldRadial, "conic": FieldConic,
	"reflected": FieldReflected, "diamond": FieldDiamond, "mesh": FieldMesh,
	"freeform": FieldFreeform, "flow": FieldFlow,
}

var styleByName = map[string]Style{
	"metallic": StyleMetallic, "chrome": StyleChrome, "iridescent": StyleIridescent,
	"holographic": StyleHolographic, "neon": StyleNeon, "pastel": StylePastel,
	"duotone": StyleDuotone, "rainbow": StyleRainbow,
}

var textureByName = map[string]Texture{
	"smooth": TextureSmooth, "grain": TextureGrain, "frosted": TextureFrosted,
	"wave": TextureWave, "wrinkle": TextureWrinkle, "paper": TexturePaper,
}

// ParamsJSON is the plain-data, JSON-decodable mirror of Params: enums are
// lower-case name strings and colours are hex strings, so a noise field can
// cross the WASM boundary or sit in a URL recipe.
type ParamsJSON struct {
	Field   string   `json:"field"`
	Style   string   `json:"style"`
	Texture string   `json:"texture"`
	Stops   []string `json:"stops"`
	Spots   []struct {
		Color string  `json:"color"`
		X     float64 `json:"x"`
		Y     float64 `json:"y"`
	} `json:"spots"`
	Angle      float64 `json:"angle"`
	Scale      float64 `json:"scale"`
	Distortion float64 `json:"distortion"`
	Seed       float64 `json:"seed"`
	Time       float64 `json:"time"`
}

// ParseParams decodes a ParamsJSON blob into a renderable Params.
func ParseParams(data []byte) (Params, error) {
	var p ParamsJSON
	if err := json.Unmarshal(data, &p); err != nil {
		return Params{}, fmt.Errorf("noisefield: parse params: %w", err)
	}
	return p.Params()
}

// Params resolves the string enums and hex colours of a ParamsJSON into a
// renderable Params.
func (p ParamsJSON) Params() (Params, error) {
	stops := make([]RGB, 0, len(p.Stops))
	for _, s := range p.Stops {
		c, err := parseHex(s)
		if err != nil {
			return Params{}, fmt.Errorf("noisefield: bad stop %q: %w", s, err)
		}
		stops = append(stops, c)
	}
	spots := make([]Spot, 0, len(p.Spots))
	for _, s := range p.Spots {
		c, err := parseHex(s.Color)
		if err != nil {
			return Params{}, fmt.Errorf("noisefield: bad spot colour %q: %w", s.Color, err)
		}
		spots = append(spots, Spot{Color: c, X: s.X, Y: s.Y})
	}
	return Params{
		Field:      fieldByName[strings.ToLower(strings.TrimSpace(p.Field))],
		Style:      styleByName[strings.ToLower(strings.TrimSpace(p.Style))],
		Texture:    textureByName[strings.ToLower(strings.TrimSpace(p.Texture))],
		Stops:      stops,
		Spots:      spots,
		AngleDeg:   p.Angle,
		Scale:      p.Scale,
		Distortion: p.Distortion,
		Seed:       int64(math.Round(p.Seed)),
		Time:       p.Time,
	}, nil
}

// RenderJSON decodes a ParamsJSON blob and renders it into a w x h image.
func RenderJSON(data []byte, w, h int) (*image.RGBA, error) {
	p, err := ParseParams(data)
	if err != nil {
		return nil, err
	}
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("noisefield: bad size %dx%d", w, h)
	}
	return Render(p, w, h), nil
}

// parseHex parses "#rgb" or "#rrggbb" (leading "#" optional) into an RGB.
// Kept local so the package stays dependency-free (colorspace aside).
func parseHex(s string) (RGB, error) {
	t := strings.TrimPrefix(strings.TrimSpace(s), "#")
	switch len(t) {
	case 3:
		r, e1 := strconv.ParseUint(t[0:1], 16, 8)
		g, e2 := strconv.ParseUint(t[1:2], 16, 8)
		b, e3 := strconv.ParseUint(t[2:3], 16, 8)
		if e1 != nil || e2 != nil || e3 != nil {
			return RGB{}, fmt.Errorf("bad hex colour %q", s)
		}
		return RGB{uint8(r * 17), uint8(g * 17), uint8(b * 17)}, nil
	case 6:
		v, err := strconv.ParseUint(t, 16, 32)
		if err != nil {
			return RGB{}, fmt.Errorf("bad hex colour %q", s)
		}
		return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}, nil
	}
	return RGB{}, fmt.Errorf("bad hex colour %q", s)
}
