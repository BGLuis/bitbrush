package gradient

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// CSSKind selects the CSS gradient function to emit.
type CSSKind int

const (
	CSSLinear CSSKind = iota
	CSSRadial
	CSSConic
)

// CSSOptions configures CSS emission.
type CSSOptions struct {
	Kind     CSSKind
	AngleDeg float64
	// Native emits a CSS Color 4 colour-interpolation-method ("in oklch
	// longer hue") plus the raw stops, letting the browser interpolate.
	// When false the gradient is baked into Samples fixed sRGB stops.
	Native bool
	// Samples is the baked stop count when !Native; clamped to [2,64],
	// defaulting to 16.
	Samples int
	// Selector, when non-empty, wraps the value in a rule
	// "<selector> { <property>: <value>; }".
	Selector string
	// Property is the declaration property for the wrapper rule; defaults
	// to "background".
	Property string
}

// CSS returns either a bare gradient value (Selector == "") or a full CSS
// rule.
func (g *Gradient) CSS(o CSSOptions) string {
	fn := "linear-gradient"
	switch o.Kind {
	case CSSRadial:
		fn = "radial-gradient"
	case CSSConic:
		fn = "conic-gradient"
	}

	var head []string
	switch o.Kind {
	case CSSLinear:
		head = append(head, cssNum(o.AngleDeg)+"deg")
	case CSSConic:
		head = append(head, "from "+cssNum(o.AngleDeg)+"deg")
	}
	if o.Native {
		method := "in " + g.Space.cssInterp()
		if g.Space.cylindrical() {
			method += " " + g.Hue.String() + " hue"
		}
		head = append(head, method)
	}

	var stops []Stop
	if o.Native {
		stops = g.Stops
	} else {
		n := o.Samples
		if n == 0 {
			n = 16
		}
		if n < 2 {
			n = 2
		}
		if n > 64 {
			n = 64
		}
		stops = g.SampleStops(n)
	}

	parts := make([]string, len(stops))
	for i, s := range stops {
		parts[i] = hex(s.Color) + " " + cssNum(clamp01(s.Pos)*100) + "%"
	}

	var b strings.Builder
	b.WriteString(fn)
	b.WriteByte('(')
	if len(head) > 0 {
		b.WriteString(strings.Join(head, " "))
		b.WriteString(", ")
	}
	b.WriteString(strings.Join(parts, ", "))
	b.WriteByte(')')
	value := b.String()

	if o.Selector == "" {
		return value
	}
	prop := o.Property
	if prop == "" {
		prop = "background"
	}
	return o.Selector + " { " + prop + ": " + value + "; }"
}

func hex(c RGB) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// cssNum formats a number for CSS, rounding to at most 3 decimal places and
// dropping trailing zeros.
func cssNum(f float64) string {
	r := math.Round(f*1000) / 1000
	if r == 0 {
		r = 0 // normalise -0
	}
	return strconv.FormatFloat(r, 'f', -1, 64)
}
