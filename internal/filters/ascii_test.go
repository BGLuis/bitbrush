package filters

import (
	"bytes"
	"image"
	"strings"
	"testing"
)

func TestASCIISolidBlackOnBlackBackgroundStaysBlack(t *testing.T) {
	src := solid(32, 24, 0, 0, 0, 255)
	out, err := Apply("ascii", src, Params{"background": "#000000"})
	if err != nil {
		t.Fatal(err)
	}
	// luma 0 -> the default ramp's darkest glyph is a space: no ink, so
	// every pixel must equal the background exactly.
	for i := 0; i < len(out.Pix); i += 4 {
		if out.Pix[i] != 0 || out.Pix[i+1] != 0 || out.Pix[i+2] != 0 {
			t.Fatalf("pixel %d = %v, want black background", i/4, out.Pix[i:i+3])
		}
		if out.Pix[i+3] != 255 {
			t.Fatalf("pixel %d alpha = %d, want 255", i/4, out.Pix[i+3])
		}
	}
}

func TestASCIISolidWhiteRendersInk(t *testing.T) {
	src := solid(32, 24, 255, 255, 255, 255)
	out, err := Apply("ascii", src, Params{"background": "#000000"})
	if err != nil {
		t.Fatal(err)
	}
	// luma 255 -> the densest glyph ('@'): a meaningful fraction of pixels
	// must differ from the black background.
	var lit int
	for i := 0; i < len(out.Pix); i += 4 {
		if out.Pix[i] != 0 || out.Pix[i+1] != 0 || out.Pix[i+2] != 0 {
			lit++
		}
	}
	total := len(out.Pix) / 4
	if ratio := float64(lit) / float64(total); ratio < 0.05 {
		t.Fatalf("only %.1f%% of pixels have ink, want a dense glyph to light up more", ratio*100)
	}
}

func TestASCIIInvertMapsWhiteToBlankGlyph(t *testing.T) {
	src := solid(32, 24, 255, 255, 255, 255)
	out, err := Apply("ascii", src, Params{"background": "#000000", "invert": true})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(out.Pix); i += 4 {
		if out.Pix[i] != 0 || out.Pix[i+1] != 0 || out.Pix[i+2] != 0 {
			t.Fatalf("pixel %d = %v, want black (inverted ramp maps white -> blank glyph)", i/4, out.Pix[i:i+3])
		}
	}
}

func TestASCIIAlphaPreserved(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+1], src.Pix[i+2] = 80, 140, 200
		src.Pix[i+3] = byte((i / 4 * 3) % 256)
	}
	out, err := Apply("ascii", src, Params{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 3; i < len(out.Pix); i += 4 {
		if out.Pix[i] != src.Pix[i] {
			t.Fatalf("pixel %d alpha = %d, want %d", i/4, out.Pix[i], src.Pix[i])
		}
	}
}

func TestASCIICellSizeNotDividingDimensions(t *testing.T) {
	src := gradientImg(21, 17) // doesn't divide evenly by the default 8x12 cell
	out, err := Apply("ascii", src, Params{})
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != image.Rect(0, 0, 21, 17) {
		t.Fatalf("bounds changed: %v", out.Bounds())
	}
}

func TestASCIIDoesNotMutateSource(t *testing.T) {
	src := gradientImg(32, 24)
	before := append([]byte(nil), src.Pix...)
	if _, err := Apply("ascii", src, Params{}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src.Pix, before) {
		t.Fatal("ascii mutated its source image")
	}
}

func TestASCIIDeterministic(t *testing.T) {
	assertDeterministic(t, "ascii", gradientImg(40, 36), Params{"colored": true})
	assertDeterministic(t, "ascii", gradientImg(40, 36), Params{"colored": false, "invert": true})
}

func TestASCIIBackgroundInvalidFallsBackGracefully(t *testing.T) {
	src := solid(16, 12, 0, 0, 0, 255)
	out, err := Apply("ascii", src, Params{"background": "not-a-colour"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(out.Pix); i += 4 {
		if out.Pix[i] != 0 || out.Pix[i+1] != 0 || out.Pix[i+2] != 0 {
			t.Fatalf("pixel %d = %v, want the black fallback", i/4, out.Pix[i:i+3])
		}
	}
}

func TestASCIITextGridDimensions(t *testing.T) {
	src := gradientImg(40, 36)
	text, err := ASCIIText(src, Params{"cellWidth": 8.0, "cellHeight": 12.0})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) != 3 { // 36/12
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	for i, line := range lines {
		if len([]rune(line)) != 5 { // 40/8
			t.Fatalf("line %d has %d runes, want 5", i, len([]rune(line)))
		}
	}
}

func TestASCIITextUsesOnlyRampCharacters(t *testing.T) {
	const ramp = " .:-=+*#%@"
	text, err := ASCIIText(gradientImg(48, 36), Params{"ramp": ramp})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range text {
		if r == '\n' {
			continue
		}
		if !strings.ContainsRune(ramp, r) {
			t.Fatalf("character %q not in ramp %q", r, ramp)
		}
	}
}

func TestASCIITextDeterministic(t *testing.T) {
	src := gradientImg(40, 36)
	a, err := ASCIIText(src, Params{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := ASCIIText(src, Params{})
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("ASCIIText is not deterministic")
	}
}

func TestASCIITextEmptyImage(t *testing.T) {
	text, err := ASCIIText(image.NewRGBA(image.Rect(0, 0, 0, 0)), Params{})
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Fatalf("empty image -> %q, want empty string", text)
	}
}

func TestParseHexColor(t *testing.T) {
	cases := []struct {
		in      string
		wantOK  bool
		r, g, b uint8
	}{
		{"#ff0080", true, 0xff, 0x00, 0x80},
		{"ff0080", true, 0xff, 0x00, 0x80},
		{"#f08", true, 0xff, 0x00, 0x88},
		{"#zzzzzz", false, 0, 0, 0},
		{"#ffff", false, 0, 0, 0},
		{"", false, 0, 0, 0},
	}
	for _, c := range cases {
		got, ok := parseHexColor(c.in)
		if ok != c.wantOK {
			t.Fatalf("parseHexColor(%q) ok=%v, want %v", c.in, ok, c.wantOK)
		}
		if ok && (got.R != c.r || got.G != c.g || got.B != c.b || got.A != 255) {
			t.Fatalf("parseHexColor(%q) = %+v, want R=%d G=%d B=%d A=255", c.in, got, c.r, c.g, c.b)
		}
	}
}

func BenchmarkASCII1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("ascii", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
