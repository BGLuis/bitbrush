package gradient

import (
	"testing"
)

func TestParseHex(t *testing.T) {
	cases := map[string]RGB{
		"#ff0000":     {255, 0, 0},
		"00ff00":      {0, 255, 0},
		"#abc":        {0xaa, 0xbb, 0xcc},
		"  #0A0B0C  ": {10, 11, 12},
	}
	for in, want := range cases {
		got, err := ParseHex(in)
		if err != nil || got != want {
			t.Fatalf("ParseHex(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "#12", "#12345", "#gggggg", "xyz"} {
		if _, err := ParseHex(in); err == nil {
			t.Fatalf("ParseHex(%q) should error", in)
		}
	}
}

func TestParseParamsAndGradient(t *testing.T) {
	data := []byte(`{
		"stops": [
			{"color": "#000000", "pos": 0},
			{"color": "#ffffff", "pos": 1}
		],
		"space": "oklch",
		"hue": "longer",
		"easing": "ease-in-out",
		"width": 40,
		"height": 20,
		"angle": 90
	}`)
	p, err := ParseParams(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.Width != 40 || p.Height != 20 || p.Angle != 90 {
		t.Fatalf("params render fields = %+v", p)
	}
	g, err := p.Gradient()
	if err != nil {
		t.Fatal(err)
	}
	if g.Space != SpaceOKLCh || g.Hue != HueLonger {
		t.Fatalf("gradient enums = %v %v", g.Space, g.Hue)
	}
	if _, ok := g.Easing.(CubicBezier); !ok {
		t.Fatalf("easing = %T, want CubicBezier", g.Easing)
	}
	if g.At(0) != (RGB{0, 0, 0}) || g.At(1) != (RGB{255, 255, 255}) {
		t.Fatal("gradient endpoints wrong")
	}
}

func TestParamsDefaultsAndErrors(t *testing.T) {
	// Empty space/hue/easing default to OKLab / shorter / Linear.
	p := Params{Stops: []ParamStop{{"#111111", 0}, {"#222222", 1}}}
	g, err := p.Gradient()
	if err != nil {
		t.Fatal(err)
	}
	if g.Space != SpaceOKLab || g.Hue != HueShorter {
		t.Fatalf("defaults = %v %v", g.Space, g.Hue)
	}
	if _, ok := g.Easing.(Linear); !ok {
		t.Fatalf("default easing = %T", g.Easing)
	}

	if _, err := (Params{}).Gradient(); err == nil {
		t.Fatal("no stops should error")
	}
	if _, err := (Params{Stops: []ParamStop{{"nope", 0}}}).Gradient(); err == nil {
		t.Fatal("bad colour should error")
	}
	if _, err := (Params{Stops: []ParamStop{{"#000", 0}}, Space: "banana"}).Gradient(); err == nil {
		t.Fatal("bad space should error")
	}
	if _, err := (Params{Stops: []ParamStop{{"#000", 0}}, Hue: "sideways"}).Gradient(); err == nil {
		t.Fatal("bad hue should error")
	}
	if _, err := (Params{Stops: []ParamStop{{"#000", 0}}, Easing: "boing()"}).Gradient(); err == nil {
		t.Fatal("bad easing should error")
	}
	if _, err := ParseParams([]byte("{not json")); err == nil {
		t.Fatal("bad JSON should error")
	}
}

func TestGenerateJSON(t *testing.T) {
	data := []byte(`{"stops":[{"color":"#ff0000","pos":0},{"color":"#0000ff","pos":1}],"space":"srgb","width":32,"height":8,"angle":90}`)
	img, err := GenerateJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 8 {
		t.Fatalf("size = %v", img.Bounds())
	}
	if got := rgbAt(img, 0, 0); chanDiff(got.R, 255) > 8 {
		t.Fatalf("left edge = %v, want ~red", got)
	}

	// Missing width/height fall back to defaults.
	img2, err := GenerateJSON([]byte(`{"stops":[{"color":"#000","pos":0},{"color":"#fff","pos":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if img2.Bounds().Dx() != 512 || img2.Bounds().Dy() != 512 {
		t.Fatalf("default size = %v", img2.Bounds())
	}
}
