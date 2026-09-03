package anim

import (
	"bytes"
	"image/gif"
	"testing"

	"bitbrush/internal/noisefield"
)

func sampleNoiseFieldParams() (noisefield.Params, noisefield.Params) {
	start := noisefield.Params{
		Field:      noisefield.FieldFlow,
		Style:      noisefield.StyleDuotone,
		Texture:    noisefield.TextureSmooth,
		Stops:      []noisefield.RGB{{R: 14, G: 165, B: 233}, {R: 139, G: 92, B: 246}},
		Spots:      []noisefield.Spot{{Color: noisefield.RGB{R: 255, G: 100, B: 50}, X: 0.3, Y: 0.4}},
		AngleDeg:   45,
		Scale:      50,
		Distortion: 50,
		Seed:       10,
		Time:       0.0,
	}
	end := start
	end.Time = 2.0
	end.AngleDeg = 405 // 45 + 360
	return start, end
}

func TestRenderNoiseFieldFramesPingPong(t *testing.T) {
	start, end := sampleNoiseFieldParams()
	opt := Options{
		Frames:       5,
		FPS:          15,
		LoopForever:  true,
		PingPong:     true,
		MaxDimension: 100,
	}

	frames, err := RenderNoiseFieldFrames(start, end, opt, 160, 90)
	if err != nil {
		t.Fatalf("RenderNoiseFieldFrames failed: %v", err)
	}
	if len(frames) != 5 {
		t.Fatalf("got %d frames, want 5", len(frames))
	}

	// In ping-pong with 5 frames, frame 0 and frame 4 should have identical dimensions
	b0 := frames[0].Bounds()
	b4 := frames[4].Bounds()
	if b0 != b4 {
		t.Errorf("frame 0 bounds %v != frame 4 bounds %v", b0, b4)
	}
	if b0.Dx() > 100 || b0.Dy() > 100 {
		t.Errorf("bounds %v exceeds MaxDimension 100", b0)
	}
}

func TestRenderNoiseFieldEncodesValidGIF(t *testing.T) {
	start, end := sampleNoiseFieldParams()
	opt := Options{
		Frames:       4,
		FPS:          10,
		LoopForever:  true,
		PingPong:     false,
		MaxDimension: 64,
	}

	data, err := RenderNoiseField(start, end, opt, 64, 64)
	if err != nil {
		t.Fatalf("RenderNoiseField failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("RenderNoiseField returned 0 bytes")
	}

	// Decode back with image/gif to confirm it's a valid GIF
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to decode generated GIF: %v", err)
	}
	if len(g.Image) != 4 {
		t.Errorf("GIF has %d frames, want 4", len(g.Image))
	}
}
