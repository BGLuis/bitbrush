package anim

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"

	"bitbrush/internal/filters"
)

// gradientImage is a small deterministic synthetic RGBA: a smooth colour
// gradient, fully opaque.
func gradientImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	dw := max(w-1, 1)
	dh := max(h-1, 1)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(x * 255 / dw),
				G: uint8(y * 255 / dh),
				B: uint8((x + y) * 255 / (dw + dh)),
				A: 0xff,
			})
		}
	}
	return img
}

func numericKeyframes() []Keyframe {
	return []Keyframe{
		{T: 0, Params: filters.Params{"a": 0.0, "b": 10.0}},
		{T: 1, Params: filters.Params{"a": 100.0, "b": 20.0}},
	}
}

func pixelateKeyframes() []Keyframe {
	return []Keyframe{
		{T: 0, Params: filters.Params{"blockSize": 2.0}},
		{T: 1, Params: filters.Params{"blockSize": 16.0}},
	}
}

func TestInterpolateNumericMidpointIsAverage(t *testing.T) {
	kfs := numericKeyframes()
	// Frame 1 of 3 with a linear phase lands exactly at t = 0.5.
	p := Interpolate(kfs, 1, 3, false)

	if got := p.Float("a", -1); got != 50 {
		t.Errorf("a: got %v, want 50", got)
	}
	if got := p.Float("b", -1); got != 15 {
		t.Errorf("b: got %v, want 15", got)
	}
}

func TestInterpolateIntStaysIntValued(t *testing.T) {
	kfs := []Keyframe{
		{T: 0, Params: filters.Params{"n": 0}},
		{T: 1, Params: filters.Params{"n": 10}},
	}
	p := Interpolate(kfs, 1, 3, false) // t = 0.5

	v, ok := p["n"].(int)
	if !ok {
		t.Fatalf("n: got %T (%v), want int", p["n"], p["n"])
	}
	if v != 5 {
		t.Errorf("n: got %d, want 5", v)
	}
}

func TestInterpolateStringAndBoolSnap(t *testing.T) {
	kfs := []Keyframe{
		{T: 0, Params: filters.Params{"mode": "a", "flag": false}},
		{T: 1, Params: filters.Params{"mode": "b", "flag": true}},
	}

	cases := []struct {
		i, n     int
		wantMode string
		wantFlag bool
	}{
		{0, 5, "a", false}, // t = 0.0
		{1, 5, "a", false}, // t = 0.25 -> nearer T=0
		{3, 5, "b", true},  // t = 0.75 -> nearer T=1
		{4, 5, "b", true},  // t = 1.0
	}
	for _, c := range cases {
		p := Interpolate(kfs, c.i, c.n, false)
		if got := p.String("mode", ""); got != c.wantMode {
			t.Errorf("i=%d/n=%d mode: got %q, want %q", c.i, c.n, got, c.wantMode)
		}
		if got := p.Bool("flag", !c.wantFlag); got != c.wantFlag {
			t.Errorf("i=%d/n=%d flag: got %v, want %v", c.i, c.n, got, c.wantFlag)
		}
	}
}

func TestInterpolatePingPongSequenceIsSymmetric(t *testing.T) {
	kfs := numericKeyframes()
	const n = 7

	got := make([]float64, n)
	for i := 0; i < n; i++ {
		got[i] = Interpolate(kfs, i, n, true).Float("a", -1)
	}

	// Endpoints return to the start value; the sequence mirrors itself.
	if got[0] != got[n-1] {
		t.Errorf("endpoints differ: %v vs %v", got[0], got[n-1])
	}
	for i := 0; i < n; i++ {
		if got[i] != got[n-1-i] {
			t.Errorf("not symmetric at i=%d: %v vs %v", i, got[i], got[n-1-i])
		}
	}
	// The middle frame must reach the far keyframe (phase 1 -> a = 100).
	if mid := got[n/2]; mid != 100 {
		t.Errorf("mid frame a: got %v, want 100", mid)
	}
}

func TestInterpolateSparseKeyIsConstantOutsideItsRange(t *testing.T) {
	kfs := []Keyframe{
		{T: 0, Params: filters.Params{"a": 0.0}},
		{T: 0.5, Params: filters.Params{"a": 50.0, "only": 8.0}},
		{T: 1, Params: filters.Params{"a": 100.0}},
	}
	// Frame 0: phase 0, before the only keyframe that carries "only".
	p := Interpolate(kfs, 0, 5, false)
	if got := p.Float("only", -1); got != 8 {
		t.Errorf("only: got %v, want 8 (held constant)", got)
	}
}

func TestInterpolateNoKeyframesReturnsEmpty(t *testing.T) {
	if p := Interpolate(nil, 0, 4, false); len(p) != 0 {
		t.Errorf("got %v, want empty params", p)
	}
}

func TestInterpolateSingleKeyframeIsConstant(t *testing.T) {
	kfs := []Keyframe{{T: 0.3, Params: filters.Params{"a": 42.0}}}
	for i := 0; i < 4; i++ {
		if got := Interpolate(kfs, i, 4, false).Float("a", -1); got != 42 {
			t.Errorf("i=%d: got %v, want 42", i, got)
		}
	}
}

func TestRenderFramesReturnsExactFrameCount(t *testing.T) {
	src := gradientImage(64, 48)
	frames, err := RenderFrames(src, "pixelate", pixelateKeyframes(), Options{Frames: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 10 {
		t.Fatalf("got %d frames, want 10", len(frames))
	}
}

func TestRenderFramesClampsFramesToTwo(t *testing.T) {
	src := gradientImage(32, 32)
	frames, err := RenderFrames(src, "pixelate", pixelateKeyframes(), Options{Frames: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2 (clamped)", len(frames))
	}
}

func TestRenderFramesMaxDimensionCapsLongestSideKeepingAspect(t *testing.T) {
	src := gradientImage(200, 100)
	frames, err := RenderFrames(src, "pixelate", pixelateKeyframes(), Options{Frames: 3, MaxDimension: 50})
	if err != nil {
		t.Fatal(err)
	}
	b := frames[0].Bounds()
	if b.Dx() != 50 || b.Dy() != 25 {
		t.Fatalf("got %dx%d, want 50x25", b.Dx(), b.Dy())
	}
}

func TestRenderFramesMaxDimensionZeroKeepsSize(t *testing.T) {
	src := gradientImage(120, 90)
	frames, err := RenderFrames(src, "pixelate", pixelateKeyframes(), Options{Frames: 3, MaxDimension: 0})
	if err != nil {
		t.Fatal(err)
	}
	b := frames[0].Bounds()
	if b.Dx() != 120 || b.Dy() != 90 {
		t.Fatalf("got %dx%d, want 120x90", b.Dx(), b.Dy())
	}
}

func TestRenderFramesNeverEnlarges(t *testing.T) {
	src := gradientImage(40, 30)
	frames, err := RenderFrames(src, "pixelate", pixelateKeyframes(), Options{Frames: 2, MaxDimension: 400})
	if err != nil {
		t.Fatal(err)
	}
	b := frames[0].Bounds()
	if b.Dx() != 40 || b.Dy() != 30 {
		t.Fatalf("got %dx%d, want 40x30 (unchanged)", b.Dx(), b.Dy())
	}
}

func TestRenderFramesErrorsWithoutKeyframes(t *testing.T) {
	src := gradientImage(16, 16)
	if _, err := RenderFrames(src, "pixelate", nil, Options{Frames: 4}); err == nil {
		t.Fatal("want error for zero keyframes, got nil")
	}
}

func TestRenderFramesActuallyAnimates(t *testing.T) {
	src := gradientImage(80, 60)
	kfs := []Keyframe{
		{T: 0, Params: filters.Params{"seed": 1.0, "rgbShift": 1.0, "maxSliceShift": 2.0}},
		{T: 1, Params: filters.Params{"seed": 7.0, "rgbShift": 25.0, "maxSliceShift": 40.0}},
	}
	frames, err := RenderFrames(src, "glitch", kfs, Options{Frames: 6})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(frames[0].Pix, frames[len(frames)-1].Pix) {
		t.Fatal("first and last glitch frames are identical; params were not interpolated into the effect")
	}
}

func TestEncodeGIFDecodesWithMatchingFramesAndBounds(t *testing.T) {
	src := gradientImage(64, 40)
	opt := Options{Frames: 5, FPS: 10, LoopForever: true}
	frames, err := RenderFrames(src, "pixelate", pixelateKeyframes(), opt)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeGIF(frames, opt)
	if err != nil {
		t.Fatal(err)
	}

	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gif.DecodeAll: %v", err)
	}
	if len(g.Image) != 5 {
		t.Errorf("got %d frames, want 5", len(g.Image))
	}
	if len(g.Delay) != 5 {
		t.Errorf("got %d delays, want 5", len(g.Delay))
	}
	if g.Delay[0] != 10 {
		t.Errorf("delay: got %d, want 10 centiseconds", g.Delay[0])
	}
	if g.LoopCount != 0 {
		t.Errorf("LoopCount: got %d, want 0 (loop forever)", g.LoopCount)
	}
	if b := g.Image[0].Bounds(); b.Dx() != 64 || b.Dy() != 40 {
		t.Errorf("frame bounds: got %dx%d, want 64x40", b.Dx(), b.Dy())
	}
	if g.Config.Width != 64 || g.Config.Height != 40 {
		t.Errorf("config: got %dx%d, want 64x40", g.Config.Width, g.Config.Height)
	}
}

func TestEncodeGIFLoopCountReflectsLoopForever(t *testing.T) {
	src := gradientImage(48, 48)

	for _, loop := range []bool{true, false} {
		opt := Options{Frames: 3, FPS: 25, LoopForever: loop}
		frames, err := RenderFrames(src, "pixelate", pixelateKeyframes(), opt)
		if err != nil {
			t.Fatal(err)
		}
		data, err := EncodeGIF(frames, opt)
		if err != nil {
			t.Fatal(err)
		}
		g, err := gif.DecodeAll(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		want := -1
		if loop {
			want = 0
		}
		if g.LoopCount != want {
			t.Errorf("LoopForever=%v: LoopCount got %d, want %d", loop, g.LoopCount, want)
		}
	}
}

func TestEncodeGIFPreservesTransparency(t *testing.T) {
	src := gradientImage(32, 32)
	// Punch a fully transparent hole.
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			src.SetRGBA(x, y, color.RGBA{})
		}
	}
	opt := Options{Frames: 2, FPS: 12}
	kfs := []Keyframe{{T: 0, Params: filters.Params{"blockSize": 4.0}}}
	frames, err := RenderFrames(src, "pixelate", kfs, opt)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeGIF(frames, opt)
	if err != nil {
		t.Fatal(err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	// A transparent palette entry must exist in the decoded first frame.
	pal, ok := g.Image[0].ColorModel().(color.Palette)
	if !ok {
		t.Fatal("frame color model is not a palette")
	}
	transparentSeen := false
	for _, c := range pal {
		if _, _, _, a := c.RGBA(); a == 0 {
			transparentSeen = true
			break
		}
	}
	if !transparentSeen {
		t.Error("no transparent palette entry survived encoding")
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	src := gradientImage(96, 72)
	kfs := []Keyframe{
		{T: 0, Params: filters.Params{"seed": 3.0, "rgbShift": 2.0}},
		{T: 1, Params: filters.Params{"seed": 41.0, "rgbShift": 24.0}},
	}
	opt := Options{Frames: 8, FPS: 20, LoopForever: true, PingPong: true}

	a, err := Render(src, "glitch", kfs, opt)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Render(src, "glitch", kfs, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("Render is not deterministic: %d vs %d bytes, differ", len(a), len(b))
	}
}

func BenchmarkRenderGIFGlitch(b *testing.B) {
	src := gradientImage(480, 480)
	kfs := []Keyframe{
		{T: 0, Params: filters.Params{"seed": 1.0, "rgbShift": 2.0}},
		{T: 1, Params: filters.Params{"seed": 99.0, "rgbShift": 30.0}},
	}
	opt := Options{Frames: 24, FPS: 24, LoopForever: true}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Render(src, "glitch", kfs, opt); err != nil {
			b.Fatal(err)
		}
	}
}
