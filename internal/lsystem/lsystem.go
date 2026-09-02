// Package lsystem expands a Lindenmayer system and draws it with turtle
// graphics. Built-in presets (Koch, plant, dragon, Sierpinski, tree) or a
// custom axiom + rules. Optional seeded jitter keeps it reproducible.
package lsystem

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"

	"bitbrush/internal/generators"
)

const maxExpanded = 3_000_000

type params struct {
	Preset       string  `json:"preset"` // plant | koch | dragon | sierpinski | tree | custom
	Axiom        string  `json:"axiom"`
	Rules        string  `json:"rules"` // "F=FF-[-F+F], X=..." (',' / ';' / newline separated)
	AngleDeg     float64 `json:"angleDeg"`
	Iterations   int     `json:"iterations"`
	Seed         int64   `json:"seed"`
	Jitter       float64 `json:"jitter"` // 0..1 random angle/length wobble
	LineWidth    float64 `json:"lineWidth"`
	ColorByDepth bool    `json:"colorByDepth"`
	Background   string  `json:"background"`
	Ink          string  `json:"ink"`
}

type preset struct {
	axiom string
	rules string
	angle float64
	iter  int
}

var presets = map[string]preset{
	"koch":       {"F", "F=F+F-F-F+F", 90, 4},
	"plant":      {"X", "X=F+[[X]-X]-F[-FX]+X;F=FF", 25, 5},
	"dragon":     {"FX", "X=X+YF+;Y=-FX-Y", 90, 12},
	"sierpinski": {"F-G-G", "F=F-G+F+G-F;G=GG", 120, 5},
	"tree":       {"F", "F=FF+[+F-F-F]-[-F+F+F]", 22, 4},
}

func render(raw json.RawMessage, w, h int) (*image.RGBA, error) {
	p := params{Preset: "plant", LineWidth: 1.4, Background: "#12130f", Ink: "#dfe7d0"}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("lsystem: bad params: %w", err)
		}
	}

	axiom, ruleStr, angle, iter := p.Axiom, p.Rules, p.AngleDeg, p.Iterations
	if pr, ok := presets[p.Preset]; ok && p.Preset != "custom" {
		axiom, ruleStr = pr.axiom, pr.rules
		if angle == 0 {
			angle = pr.angle
		}
		if iter == 0 {
			iter = pr.iter
		}
	}
	if axiom == "" {
		return nil, fmt.Errorf("lsystem: empty axiom")
	}
	iter = clampi(iter, 0, 18)
	p.Jitter = clampf(p.Jitter, 0, 1)
	turn := angle * math.Pi / 180

	rules := parseRules(ruleStr)
	s := expand(axiom, rules, iter)

	// Bounding box from a jitter-free dry run at unit step.
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	walk(s, turn, 0, nil, func(x0, y0, x1, y1 float64, _ int) {
		minX = math.Min(minX, math.Min(x0, x1))
		minY = math.Min(minY, math.Min(y0, y1))
		maxX = math.Max(maxX, math.Max(x0, x1))
		maxY = math.Max(maxY, math.Max(y0, y1))
	})
	if math.IsInf(minX, 0) {
		return nil, fmt.Errorf("lsystem: nothing to draw (axiom/rules produce no forward moves)")
	}
	spanX := math.Max(maxX-minX, 1e-6)
	spanY := math.Max(maxY-minY, 1e-6)
	scale := 0.9 * math.Min(float64(w)/spanX, float64(h)/spanY)
	offX := (float64(w) - spanX*scale) / 2
	offY := (float64(h) - spanY*scale) / 2

	bg := generators.ParseHex(p.Background, color.RGBA{18, 19, 15, 255})
	ink := generators.ParseHex(p.Ink, color.RGBA{223, 231, 208, 255})
	img := generators.NewCanvas(w, h, bg)

	rng := rand.New(rand.NewSource(p.Seed))
	hw := math.Max(0.5, p.LineWidth) * 0.5
	tf := func(x, y float64) (float64, float64) {
		return (x-minX)*scale + offX, (y-minY)*scale + offY
	}
	walk(s, turn, p.Jitter, rng, func(x0, y0, x1, y1 float64, depth int) {
		ax, ay := tf(x0, y0)
		bx, by := tf(x1, y1)
		col := ink
		if p.ColorByDepth {
			r, g, b := hsv(0.33+float64(depth)*0.07, 0.5, 1)
			col = color.RGBA{u8(r), u8(g), u8(b), 255}
		}
		dx, dy := bx-ax, by-ay
		l := math.Hypot(dx, dy)
		if l == 0 {
			generators.SplatAA(img, ax, ay, 1, col)
			return
		}
		nx, ny := -dy/l, dx/l
		for off := -hw; off <= hw; off += 0.75 {
			generators.Line(img, ax+nx*off, ay+ny*off, bx+nx*off, by+ny*off, 1, col)
		}
	})
	return img, nil
}

func parseRules(s string) map[rune]string {
	out := map[rune]string{}
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		part = strings.TrimSpace(part)
		eq := strings.IndexByte(part, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(part[:eq])
		val := strings.TrimSpace(part[eq+1:])
		if len(key) == 0 {
			continue
		}
		out[[]rune(key)[0]] = val
	}
	return out
}

func expand(axiom string, rules map[rune]string, iter int) string {
	cur := axiom
	for it := 0; it < iter; it++ {
		var b strings.Builder
		for _, r := range cur {
			if v, ok := rules[r]; ok {
				b.WriteString(v)
			} else {
				b.WriteRune(r)
			}
			if b.Len() > maxExpanded {
				break
			}
		}
		cur = b.String()
		if len(cur) > maxExpanded {
			break
		}
	}
	return cur
}

type tstate struct{ x, y, a float64 }

func walk(s string, turn, jitter float64, rng *rand.Rand, seg func(x0, y0, x1, y1 float64, depth int)) {
	x, y, a := 0.0, 0.0, -math.Pi/2
	var stack []tstate
	depth := 0
	for _, r := range s {
		switch r {
		case 'F', 'G':
			ln, aa := 1.0, a
			if jitter > 0 && rng != nil {
				ln *= 1 + (rng.Float64()-0.5)*jitter
				aa += (rng.Float64() - 0.5) * jitter * turn
			}
			nx := x + math.Cos(aa)*ln
			ny := y + math.Sin(aa)*ln
			seg(x, y, nx, ny, depth)
			x, y = nx, ny
		case 'f':
			x += math.Cos(a)
			y += math.Sin(a)
		case '+':
			a -= turn
		case '-':
			a += turn
		case '|':
			a += math.Pi
		case '[':
			stack = append(stack, tstate{x, y, a})
			depth++
		case ']':
			if n := len(stack); n > 0 {
				x, y, a = stack[n-1].x, stack[n-1].y, stack[n-1].a
				stack = stack[:n-1]
				depth--
			}
		}
	}
}

func hsv(h, s, v float64) (float64, float64, float64) {
	h = math.Mod(math.Mod(h, 1)+1, 1) * 6
	i := math.Floor(h)
	f := h - i
	pp := v * (1 - s)
	q := v * (1 - s*f)
	tt := v * (1 - s*(1-f))
	switch int(i) % 6 {
	case 0:
		return v, tt, pp
	case 1:
		return q, v, pp
	case 2:
		return pp, v, tt
	case 3:
		return pp, q, v
	case 4:
		return tt, pp, v
	default:
		return v, pp, q
	}
}

func u8(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 255
	}
	return uint8(v*255 + 0.5)
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func init() { generators.Register("lsystem", render) }
