package gradient

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Easing maps progress in [0,1] to [0,1]. It is applied to the local
// parameter BETWEEN two stops, never globally across the gradient.
type Easing interface {
	Ease(t float64) float64
}

// Linear is the identity easing.
type Linear struct{}

// Ease returns t unchanged.
func (Linear) Ease(t float64) float64 { return t }

func (Linear) String() string { return "linear" }

// CubicBezier is a CSS cubic-bezier() easing: control points (0,0),
// (X1,Y1), (X2,Y2), (1,1). Ease(p) solves x(t)=p for t (Newton's method
// with a bisection fallback, both with a fixed iteration count for
// determinism) and returns y(t).
type CubicBezier struct{ X1, Y1, X2, Y2 float64 }

// Ease returns the eased value of p. Endpoints are exact: Ease(0)==0 and
// Ease(1)==1.
func (c CubicBezier) Ease(p float64) float64 {
	if p <= 0 {
		return 0
	}
	if p >= 1 {
		return 1
	}
	t := c.solve(p)
	return bezierAxis(t, c.Y1, c.Y2)
}

// bezierAxis evaluates one coordinate of the bezier at parameter t with
// control values 0, p1, p2, 1.
func bezierAxis(t, p1, p2 float64) float64 {
	u := 1 - t
	return 3*u*u*t*p1 + 3*u*t*t*p2 + t*t*t
}

// bezierSlope is d/dt of bezierAxis.
func bezierSlope(t, p1, p2 float64) float64 {
	u := 1 - t
	return 3*u*u*p1 + 6*u*t*(p2-p1) + 3*t*t*(1-p2)
}

// solve returns the parameter t with x(t) ≈ x, for x in (0,1).
func (c CubicBezier) solve(x float64) float64 {
	t := x
	for i := 0; i < 8; i++ {
		fx := bezierAxis(t, c.X1, c.X2) - x
		if math.Abs(fx) < 1e-9 {
			return clamp01(t)
		}
		d := bezierSlope(t, c.X1, c.X2)
		if math.Abs(d) < 1e-12 {
			break
		}
		t -= fx / d
	}
	// Bisection fallback: always brackets a root in [0,1].
	lo, hi := 0.0, 1.0
	t = clamp01(x)
	for i := 0; i < 64; i++ {
		v := bezierAxis(t, c.X1, c.X2)
		if math.Abs(v-x) < 1e-12 {
			break
		}
		if v < x {
			lo = t
		} else {
			hi = t
		}
		t = 0.5 * (lo + hi)
	}
	return t
}

func (c CubicBezier) String() string {
	switch {
	case bezierApprox(c, 0.25, 0.1, 0.25, 1):
		return "ease"
	case bezierApprox(c, 0.42, 0, 1, 1):
		return "ease-in"
	case bezierApprox(c, 0, 0, 0.58, 1):
		return "ease-out"
	case bezierApprox(c, 0.42, 0, 0.58, 1):
		return "ease-in-out"
	}
	return fmt.Sprintf("cubic-bezier(%s, %s, %s, %s)",
		ftoa(c.X1), ftoa(c.Y1), ftoa(c.X2), ftoa(c.Y2))
}

func bezierApprox(c CubicBezier, x1, y1, x2, y2 float64) bool {
	const e = 1e-9
	return math.Abs(c.X1-x1) < e && math.Abs(c.Y1-y1) < e &&
		math.Abs(c.X2-x2) < e && math.Abs(c.Y2-y2) < e
}

// StepJump selects where the rises of a Steps easing fall, matching the CSS
// steps() jump-term.
type StepJump int

const (
	JumpEnd StepJump = iota
	JumpStart
	JumpBoth
	JumpNone
)

func (j StepJump) String() string {
	switch j {
	case JumpStart:
		return "start"
	case JumpBoth:
		return "both"
	case JumpNone:
		return "none"
	default:
		return "end"
	}
}

// Steps is a CSS steps(N, jump) easing: N equal plateaus over [0,1].
type Steps struct {
	N    int
	Jump StepJump
}

// Ease returns the step value for t (clamped to [0,1]).
func (s Steps) Ease(t float64) float64 {
	n := s.N
	if n < 1 {
		n = 1
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	cur := math.Floor(t * float64(n))
	var jumps float64
	switch s.Jump {
	case JumpStart:
		cur++
		jumps = float64(n)
	case JumpBoth:
		cur++
		jumps = float64(n) + 1
	case JumpNone:
		jumps = float64(n) - 1
	default: // JumpEnd
		jumps = float64(n)
	}
	if cur < 0 {
		cur = 0
	}
	if cur > jumps {
		cur = jumps
	}
	if jumps <= 0 {
		return 0
	}
	return cur / jumps
}

func (s Steps) String() string {
	return fmt.Sprintf("steps(%d, %s)", s.N, s.Jump)
}

// LinearPoint is one (input, output) knot of a CSS linear() easing, both in
// [0,1].
type LinearPoint struct{ Input, Output float64 }

// LinearPoints is a CSS linear() easing: piecewise-linear through Points,
// which must be sorted by non-decreasing Input.
type LinearPoints struct{ Points []LinearPoint }

// Ease interpolates the output for t (clamped to the knot range).
func (l LinearPoints) Ease(t float64) float64 {
	p := l.Points
	switch len(p) {
	case 0:
		return clamp01(t)
	case 1:
		return p[0].Output
	}
	if t <= p[0].Input {
		return p[0].Output
	}
	last := p[len(p)-1]
	if t >= last.Input {
		return last.Output
	}
	for i := 0; i+1 < len(p); i++ {
		a, b := p[i], p[i+1]
		if t >= a.Input && t <= b.Input {
			if b.Input == a.Input {
				return b.Output
			}
			f := (t - a.Input) / (b.Input - a.Input)
			return a.Output + (b.Output-a.Output)*f
		}
	}
	return last.Output
}

func (l LinearPoints) String() string {
	parts := make([]string, len(l.Points))
	for i, p := range l.Points {
		parts[i] = ftoa(p.Output) + " " + ftoa(p.Input*100) + "%"
	}
	return "linear(" + strings.Join(parts, ", ") + ")"
}

// ParseEasing parses a CSS easing string. It accepts: "linear", "ease",
// "ease-in", "ease-out", "ease-in-out", "cubic-bezier(x1,y1,x2,y2)",
// "steps(n[, start|end|both|none])" (also the "jump-*" spellings), and
// "linear(0, 0.25 25%, 1)". Anything else is an error.
func ParseEasing(s string) (Easing, error) {
	t := strings.ToLower(strings.TrimSpace(s))
	switch t {
	case "linear":
		return Linear{}, nil
	case "ease":
		return CubicBezier{0.25, 0.1, 0.25, 1}, nil
	case "ease-in":
		return CubicBezier{0.42, 0, 1, 1}, nil
	case "ease-out":
		return CubicBezier{0, 0, 0.58, 1}, nil
	case "ease-in-out":
		return CubicBezier{0.42, 0, 0.58, 1}, nil
	}

	if arg, ok := fnArg(t, "cubic-bezier"); ok {
		f, err := parseFloatList(arg, 4)
		if err != nil {
			return nil, fmt.Errorf("gradient: cubic-bezier: %w", err)
		}
		return CubicBezier{f[0], f[1], f[2], f[3]}, nil
	}
	if arg, ok := fnArg(t, "steps"); ok {
		return parseSteps(arg)
	}
	if arg, ok := fnArg(t, "linear"); ok {
		return parseLinearPoints(arg)
	}
	return nil, fmt.Errorf("gradient: unknown easing %q", s)
}

// fnArg returns the argument text of "name(...)" and whether s had that form.
func fnArg(s, name string) (string, bool) {
	if strings.HasPrefix(s, name+"(") && strings.HasSuffix(s, ")") {
		return s[len(name)+1 : len(s)-1], true
	}
	return "", false
}

func parseFloatList(s string, want int) ([]float64, error) {
	fields := strings.Split(s, ",")
	if len(fields) != want {
		return nil, fmt.Errorf("want %d numbers, got %d", want, len(fields))
	}
	out := make([]float64, want)
	for i, f := range fields {
		v, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		if err != nil {
			return nil, fmt.Errorf("bad number %q", strings.TrimSpace(f))
		}
		out[i] = v
	}
	return out, nil
}

func parseSteps(arg string) (Easing, error) {
	parts := strings.Split(arg, ",")
	if len(parts) == 0 || len(parts) > 2 {
		return nil, fmt.Errorf("gradient: steps(%s): want 1 or 2 args", arg)
	}
	n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || n < 1 {
		return nil, fmt.Errorf("gradient: steps: bad count %q", strings.TrimSpace(parts[0]))
	}
	jump := JumpEnd
	if len(parts) == 2 {
		switch strings.TrimSpace(parts[1]) {
		case "start", "jump-start":
			jump = JumpStart
		case "end", "jump-end":
			jump = JumpEnd
		case "both", "jump-both":
			jump = JumpBoth
		case "none", "jump-none":
			jump = JumpNone
		default:
			return nil, fmt.Errorf("gradient: steps: bad jump %q", strings.TrimSpace(parts[1]))
		}
	}
	if jump == JumpNone && n < 2 {
		return nil, fmt.Errorf("gradient: steps(n, jump-none) needs n >= 2")
	}
	return Steps{N: n, Jump: jump}, nil
}

func parseLinearPoints(arg string) (Easing, error) {
	raw := strings.Split(arg, ",")
	if len(raw) < 2 {
		return nil, fmt.Errorf("gradient: linear(): need at least 2 points")
	}

	type knot struct {
		out   float64
		in    float64
		hasIn bool
	}
	knots := make([]knot, 0, len(raw))
	for _, r := range raw {
		fields := strings.Fields(strings.TrimSpace(r))
		if len(fields) == 0 || len(fields) > 3 {
			return nil, fmt.Errorf("gradient: linear(): bad point %q", strings.TrimSpace(r))
		}
		out, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return nil, fmt.Errorf("gradient: linear(): bad output %q", fields[0])
		}
		if len(fields) == 1 {
			knots = append(knots, knot{out: out})
			continue
		}
		for _, in := range fields[1:] {
			v, err := parsePercent(in)
			if err != nil {
				return nil, err
			}
			knots = append(knots, knot{out: out, in: v, hasIn: true})
		}
	}

	n := len(knots)
	if !knots[0].hasIn {
		knots[0] = knot{out: knots[0].out, in: 0, hasIn: true}
	}
	if !knots[n-1].hasIn {
		knots[n-1] = knot{out: knots[n-1].out, in: 1, hasIn: true}
	}
	// Distribute runs of missing inputs evenly between their anchors.
	for i := 0; i < n; {
		if knots[i].hasIn {
			i++
			continue
		}
		j := i
		for j < n && !knots[j].hasIn {
			j++
		}
		lo, hi := knots[i-1].in, knots[j].in
		span := float64(j - (i - 1))
		for k := i; k < j; k++ {
			knots[k].in = lo + (hi-lo)*float64(k-(i-1))/span
			knots[k].hasIn = true
		}
		i = j
	}

	pts := make([]LinearPoint, n)
	maxIn := 0.0
	for i, k := range knots {
		in := k.in
		if in < maxIn { // CSS: input is non-decreasing
			in = maxIn
		}
		maxIn = in
		pts[i] = LinearPoint{Input: in, Output: k.out}
	}
	return LinearPoints{Points: pts}, nil
}

func parsePercent(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if !strings.HasSuffix(s, "%") {
		return 0, fmt.Errorf("gradient: linear(): input %q needs a %% suffix", s)
	}
	v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
	if err != nil {
		return 0, fmt.Errorf("gradient: linear(): bad input %q", s)
	}
	return v / 100, nil
}

// ftoa formats a float compactly for CSS output.
func ftoa(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}
