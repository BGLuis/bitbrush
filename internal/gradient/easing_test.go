package gradient

import (
	"math"
	"testing"
)

func TestLinearIsIdentity(t *testing.T) {
	for i := 0; i <= 20; i++ {
		x := float64(i) / 20
		if got := (Linear{}).Ease(x); got != x {
			t.Fatalf("Linear.Ease(%v) = %v, want %v", x, got, x)
		}
	}
}

func TestCubicBezierIdentityCurveIsLinear(t *testing.T) {
	cb := CubicBezier{0, 0, 1, 1}
	for i := 0; i <= 100; i++ {
		x := float64(i) / 100
		if d := math.Abs(cb.Ease(x) - x); d > 1e-6 {
			t.Fatalf("cubic-bezier(0,0,1,1).Ease(%v) = %v, off by %v", x, cb.Ease(x), d)
		}
	}
}

func TestCubicBezierEndpointsExact(t *testing.T) {
	for _, cb := range []CubicBezier{
		{0.25, 0.1, 0.25, 1},
		{0.42, 0, 0.58, 1},
		{0.6, -0.2, 0.4, 1.3}, // overshooting control points
	} {
		if got := cb.Ease(0); got != 0 {
			t.Fatalf("%v.Ease(0) = %v, want 0", cb, got)
		}
		if got := cb.Ease(1); got != 1 {
			t.Fatalf("%v.Ease(1) = %v, want 1", cb, got)
		}
	}
}

func TestCubicBezierEaseInOutMonotonic(t *testing.T) {
	cb := CubicBezier{0.42, 0, 0.58, 1}
	prev := cb.Ease(0)
	for i := 1; i <= 200; i++ {
		x := float64(i) / 200
		cur := cb.Ease(x)
		if cur < prev-1e-9 {
			t.Fatalf("ease-in-out not monotonic at x=%v: %v then %v", x, prev, cur)
		}
		prev = cur
	}
}

func TestStepsJumpEndValues(t *testing.T) {
	s := Steps{N: 4, Jump: JumpEnd}
	allowed := map[float64]bool{0: true, 0.25: true, 0.5: true, 0.75: true}
	// Sample strictly inside [0,1): the last plateau only lifts to 1 at t==1.
	for i := 0; i < 100; i++ {
		x := float64(i) / 100
		got := s.Ease(x)
		if !allowed[math.Round(got*1e6)/1e6] {
			t.Fatalf("Steps{4,JumpEnd}.Ease(%v) = %v, not in {0,.25,.5,.75}", x, got)
		}
	}
	if got := s.Ease(1); got != 1 {
		t.Fatalf("Steps{4,JumpEnd}.Ease(1) = %v, want 1", got)
	}
}

func TestStepsJumpVariants(t *testing.T) {
	// jump-start lifts immediately; jump-none reaches 1 before t==1.
	if got := (Steps{N: 4, Jump: JumpStart}).Ease(0); got != 0.25 {
		t.Fatalf("Steps{4,JumpStart}.Ease(0) = %v, want 0.25", got)
	}
	if got := (Steps{N: 4, Jump: JumpNone}).Ease(0); got != 0 {
		t.Fatalf("Steps{4,JumpNone}.Ease(0) = %v, want 0", got)
	}
	if got := (Steps{N: 4, Jump: JumpNone}).Ease(0.999); got != 1 {
		t.Fatalf("Steps{4,JumpNone}.Ease(~1) = %v, want 1", got)
	}
}

func TestLinearPointsEasing(t *testing.T) {
	e, err := ParseEasing("linear(0, 0.25 25%, 1)")
	if err != nil {
		t.Fatal(err)
	}
	lp := e.(LinearPoints)
	want := []LinearPoint{{0, 0}, {0.25, 0.25}, {1, 1}}
	if len(lp.Points) != len(want) {
		t.Fatalf("points = %v, want %v", lp.Points, want)
	}
	for i, p := range lp.Points {
		if math.Abs(p.Input-want[i].Input) > 1e-12 || math.Abs(p.Output-want[i].Output) > 1e-12 {
			t.Fatalf("point %d = %v, want %v", i, p, want[i])
		}
	}
	for _, tc := range []struct{ in, out float64 }{
		{0, 0}, {0.125, 0.125}, {0.25, 0.25}, {0.625, 0.625}, {1, 1},
	} {
		if got := e.Ease(tc.in); math.Abs(got-tc.out) > 1e-9 {
			t.Fatalf("Ease(%v) = %v, want %v", tc.in, got, tc.out)
		}
	}
}

func TestParseEasingNamedForms(t *testing.T) {
	cases := map[string]Easing{
		"linear":      Linear{},
		"ease":        CubicBezier{0.25, 0.1, 0.25, 1},
		"ease-in":     CubicBezier{0.42, 0, 1, 1},
		"ease-out":    CubicBezier{0, 0, 0.58, 1},
		"ease-in-out": CubicBezier{0.42, 0, 0.58, 1},
	}
	for s, want := range cases {
		got, err := ParseEasing(s)
		if err != nil {
			t.Fatalf("ParseEasing(%q): %v", s, err)
		}
		if got != want {
			t.Fatalf("ParseEasing(%q) = %#v, want %#v", s, got, want)
		}
		// Round-trip through the canonical string.
		rt, err := ParseEasing(got.(interface{ String() string }).String())
		if err != nil {
			t.Fatalf("re-parse of %q: %v", s, err)
		}
		if rt != want {
			t.Fatalf("round-trip %q -> %#v", s, rt)
		}
	}
}

func TestParseEasingFunctionalForms(t *testing.T) {
	if got, err := ParseEasing("cubic-bezier(0.1, 0.2, 0.3, 0.4)"); err != nil || got != (CubicBezier{0.1, 0.2, 0.3, 0.4}) {
		t.Fatalf("cubic-bezier parse = %#v, %v", got, err)
	}
	for _, s := range []string{"steps(4, start)", "steps(4, jump-start)", "steps(3)", "steps(5, none)", "steps(2, both)"} {
		if _, err := ParseEasing(s); err != nil {
			t.Fatalf("ParseEasing(%q): %v", s, err)
		}
	}
	// steps round-trip
	got, _ := ParseEasing("steps(4, end)")
	if got.(Steps) != (Steps{N: 4, Jump: JumpEnd}) {
		t.Fatalf("steps parse = %#v", got)
	}
	if rt, _ := ParseEasing(got.(Steps).String()); rt != got {
		t.Fatalf("steps round-trip = %#v", rt)
	}
}

func TestParseEasingRejectsGarbage(t *testing.T) {
	for _, s := range []string{
		"", "   ", "bogus", "wiggle(1)",
		"cubic-bezier(1,2,3)", "cubic-bezier(a,b,c,d)", "cubic-bezier()",
		"steps(0, end)", "steps(3, sideways)", "steps(1, jump-none)", "steps(-2, end)",
		"linear()", "linear(0)", "linear(0, abc, 1)", "linear(0, 0.5 50, 1)",
	} {
		if e, err := ParseEasing(s); err == nil {
			t.Fatalf("ParseEasing(%q) = %#v, want error", s, e)
		}
	}
}
