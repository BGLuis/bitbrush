package reactiondiffusion

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestReactionDiffusionDeterministic(t *testing.T) {
	p := params{
		Seed:     42,
		Preset:   "spots",
		Steps:    200,
		GridSize: 80,
	}
	blob, _ := json.Marshal(p)

	img1, err := render(blob, 80, 80)
	if err != nil {
		t.Fatalf("render 1 failed: %v", err)
	}
	img2, err := render(blob, 80, 80)
	if err != nil {
		t.Fatalf("render 2 failed: %v", err)
	}

	if !bytes.Equal(img1.Pix, img2.Pix) {
		t.Fatal("reactiondiffusion is not deterministic")
	}
}

func TestReactionDiffusionPresets(t *testing.T) {
	presets := []string{"coral", "mitosis", "spots", "labyrinth", "rings"}
	for _, pr := range presets {
		p := params{
			Seed:     101,
			Preset:   pr,
			Steps:    150,
			GridSize: 70,
		}
		blob, _ := json.Marshal(p)
		img, err := render(blob, 64, 64)
		if err != nil {
			t.Errorf("preset %q failed: %v", pr, err)
		}
		if img == nil || img.Bounds().Dx() != 64 || img.Bounds().Dy() != 64 {
			t.Errorf("bad output for preset %q", pr)
		}
	}
}
