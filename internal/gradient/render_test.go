package gradient

import (
	"bytes"
	"image"
	"testing"
)

func rgbAt(img *image.RGBA, x, y int) RGB {
	i := img.PixOffset(x, y)
	return RGB{img.Pix[i], img.Pix[i+1], img.Pix[i+2]}
}

func TestRenderHorizontal(t *testing.T) {
	g := New([]Stop{{RGB{255, 0, 0}, 0}, {RGB{0, 0, 255}, 1}}, SpaceSRGB, HueShorter, nil)
	img := g.Render(64, 32, 90) // 90deg = to the right

	for y := 0; y < 32; y++ {
		left := rgbAt(img, 0, y)
		right := rgbAt(img, 63, y)
		if chanDiff(left.R, 255) > 6 || left.B > 6 {
			t.Fatalf("left column @%d = %v, want ~first stop (red)", y, left)
		}
		if chanDiff(right.B, 255) > 6 || right.R > 6 {
			t.Fatalf("right column @%d = %v, want ~last stop (blue)", y, right)
		}
	}
	// Opaque.
	if img.Pix[3] != 255 {
		t.Fatalf("alpha = %d, want 255", img.Pix[3])
	}
	// Monotonic red decrease along X in the first row.
	prev := 255
	for x := 0; x < 64; x++ {
		r := int(rgbAt(img, x, 0).R)
		if r > prev+1 {
			t.Fatalf("red not decreasing along X at x=%d: %d then %d", x, prev, r)
		}
		prev = r
	}
}

func TestRenderAngleZeroVariesAlongY(t *testing.T) {
	g := New([]Stop{{RGB{255, 0, 0}, 0}, {RGB{0, 0, 255}, 1}}, SpaceSRGB, HueShorter, nil)
	img := g.Render(64, 32, 0)

	// Every column identical (no variation along X).
	for x := 1; x < 64; x++ {
		if rgbAt(img, x, 5) != rgbAt(img, 0, 5) {
			t.Fatalf("row varies along X at x=%d", x)
		}
	}
	// Rows do vary: top vs bottom differ strongly.
	top := rgbAt(img, 0, 0)
	bot := rgbAt(img, 0, 31)
	if chanDiff(top.R, bot.R) < 200 || chanDiff(top.B, bot.B) < 200 {
		t.Fatalf("angle 0 did not put the change along Y: top=%v bot=%v", top, bot)
	}
}

func TestRenderDeterministic(t *testing.T) {
	g := New([]Stop{
		{RGB{12, 200, 40}, 0}, {RGB{240, 10, 90}, 0.4}, {RGB{0, 0, 0}, 1},
	}, SpaceOKLab, HueShorter, CubicBezier{0.42, 0, 0.58, 1})
	a := g.Render(128, 96, 33.3)
	b := g.Render(128, 96, 33.3)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("Render is not deterministic")
	}
}

func TestGenerateValidates(t *testing.T) {
	g := New([]Stop{{RGB{1, 2, 3}, 0}, {RGB{4, 5, 6}, 1}}, SpaceOKLab, HueShorter, nil)
	if _, err := Generate(nil, 10, 10, 0); err == nil {
		t.Fatal("Generate(nil) should error")
	}
	if _, err := Generate(g, 0, 10, 0); err == nil {
		t.Fatal("Generate with zero width should error")
	}
	if _, err := Generate(g, 1<<20, 10, 0); err == nil {
		t.Fatal("Generate with oversized width should error")
	}
	img, err := Generate(g, 20, 10, 45)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 20 || img.Bounds().Dy() != 10 {
		t.Fatalf("image size = %v", img.Bounds())
	}
}

func BenchmarkRender1920x1080(b *testing.B) {
	g := New([]Stop{
		{RGB{255, 0, 0}, 0},
		{RGB{0, 255, 0}, 0.5},
		{RGB{0, 0, 255}, 1},
	}, SpaceOKLab, HueShorter, CubicBezier{0.42, 0, 0.58, 1})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Render(1920, 1080, 45)
	}
}
