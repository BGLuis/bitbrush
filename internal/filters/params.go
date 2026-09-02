package filters

// Params is the decoded parameter bag handed across the WASM boundary
// (JSON object -> map). Effects pull typed values with the helpers below,
// always supplying a default.
type Params map[string]any

func (p Params) Float(key string, def float64) float64 {
	if v, ok := p[key].(float64); ok {
		return v
	}
	return def
}

func (p Params) Int(key string, def int) int {
	if v, ok := p[key].(float64); ok {
		return int(v)
	}
	return def
}

func (p Params) String(key, def string) string {
	if v, ok := p[key].(string); ok {
		return v
	}
	return def
}

func (p Params) Bool(key string, def bool) bool {
	if v, ok := p[key].(bool); ok {
		return v
	}
	return def
}
