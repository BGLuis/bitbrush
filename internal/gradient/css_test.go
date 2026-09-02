package gradient

import (
	"strings"
	"testing"
)

func mkGradient(sp Space, hue HueArc) *Gradient {
	return New([]Stop{
		{RGB{255, 0, 0}, 0},
		{RGB{0, 255, 0}, 0.5},
		{RGB{0, 0, 255}, 1},
	}, sp, hue, nil)
}

func TestCSSNative(t *testing.T) {
	g := mkGradient(SpaceOKLCh, HueLonger)
	out := g.CSS(CSSOptions{AngleDeg: 90, Native: true})
	if !strings.Contains(out, "in oklch") {
		t.Fatalf("native output missing colour space: %q", out)
	}
	if !strings.Contains(out, "hue") {
		t.Fatalf("native output missing hue arc: %q", out)
	}
	if !strings.Contains(out, "longer") {
		t.Fatalf("native output missing arc name: %q", out)
	}
	if !strings.HasPrefix(out, "linear-gradient(90deg in oklch longer hue, ") {
		t.Fatalf("unexpected native prefix: %q", out)
	}
	if strings.Count(out, "#") != 3 {
		t.Fatalf("native output should carry the 3 raw stops: %q", out)
	}
}

func TestCSSBakedSampleCount(t *testing.T) {
	g := mkGradient(SpaceOKLab, HueShorter)
	out := g.CSS(CSSOptions{AngleDeg: 45, Native: false, Samples: 8})
	if n := strings.Count(out, "#"); n != 8 {
		t.Fatalf("Samples:8 produced %d colour stops: %q", n, out)
	}
	if strings.Contains(out, "in oklab") {
		t.Fatalf("baked output must not carry an interpolation method: %q", out)
	}
	if !strings.HasPrefix(out, "linear-gradient(45deg, #") {
		t.Fatalf("unexpected baked prefix: %q", out)
	}
	// Default sample count is 16.
	if n := strings.Count(g.CSS(CSSOptions{Native: false}), "#"); n != 16 {
		t.Fatalf("default sample count = %d, want 16", n)
	}
	// Clamp to [2,64].
	if n := strings.Count(g.CSS(CSSOptions{Native: false, Samples: 999}), "#"); n != 64 {
		t.Fatalf("Samples:999 clamped to %d, want 64", n)
	}
	if n := strings.Count(g.CSS(CSSOptions{Native: false, Samples: 1}), "#"); n != 2 {
		t.Fatalf("Samples:1 clamped to %d, want 2", n)
	}
}

func TestCSSSelectorWrapper(t *testing.T) {
	g := mkGradient(SpaceOKLab, HueShorter)
	out := g.CSS(CSSOptions{Selector: ".g", Native: true, AngleDeg: 0})
	if !strings.HasPrefix(out, ".g { background: linear-gradient(") {
		t.Fatalf("wrapper prefix wrong: %q", out)
	}
	if !strings.HasSuffix(out, "); }") {
		t.Fatalf("wrapper suffix wrong: %q", out)
	}

	out = g.CSS(CSSOptions{Selector: ".g", Property: "border-image", Native: true})
	if !strings.HasPrefix(out, ".g { border-image: linear-gradient(") {
		t.Fatalf("custom property wrong: %q", out)
	}

	// No selector => bare value.
	bare := g.CSS(CSSOptions{})
	if strings.Contains(bare, "{") {
		t.Fatalf("bare value should have no rule braces: %q", bare)
	}
}

func TestCSSKinds(t *testing.T) {
	g := mkGradient(SpaceOKLCh, HueShorter)
	if out := g.CSS(CSSOptions{Kind: CSSRadial, Native: true}); !strings.HasPrefix(out, "radial-gradient(in oklch shorter hue, ") {
		t.Fatalf("radial: %q", out)
	}
	if out := g.CSS(CSSOptions{Kind: CSSConic, AngleDeg: 30, Native: true}); !strings.HasPrefix(out, "conic-gradient(from 30deg in oklch shorter hue, ") {
		t.Fatalf("conic: %q", out)
	}
}

func TestCSSStopFormatting(t *testing.T) {
	g := mkGradient(SpaceOKLab, HueShorter)
	out := g.CSS(CSSOptions{Native: true, AngleDeg: 90})
	for _, want := range []string{"#ff0000 0%", "#00ff00 50%", "#0000ff 100%"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing stop %q in %q", want, out)
		}
	}
}
