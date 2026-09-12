package text

import (
	"testing"
)

func TestRenderText(t *testing.T) {
	img, err := Render(Params{
		Content:    "Hello World\nLine 2",
		FontFamily: "go",
		Size:       36,
		Color:      "#ffffff",
		X:          0.5,
		Y:          0.5,
		Align:      "center",
		Opacity:    1.0,
	}, 300, 150)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if img.Bounds().Dx() != 300 || img.Bounds().Dy() != 150 {
		t.Fatalf("unexpected bounds: %v", img.Bounds())
	}

	hasVisiblePixels := false
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] > 0 {
			hasVisiblePixels = true
			break
		}
	}
	if !hasVisiblePixels {
		t.Errorf("expected text to produce non-transparent pixels")
	}
}

func TestRenderEmpty(t *testing.T) {
	img, err := Render(Params{
		Content: "",
	}, 100, 100)
	if err != nil {
		t.Fatalf("Render empty failed: %v", err)
	}
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0 {
			t.Errorf("expected empty string to produce fully transparent canvas")
			break
		}
	}
}

func TestRenderMonoAndAlign(t *testing.T) {
	for _, align := range []string{"left", "center", "right"} {
		img, err := Render(Params{
			Content:    "Mono",
			FontFamily: "gomono",
			Size:       24,
			Color:      "#ff0000",
			X:          0.2,
			Y:          0.2,
			Align:      align,
			Opacity:    0.5,
		}, 200, 100)
		if err != nil {
			t.Fatalf("Render align %s failed: %v", align, err)
		}
		hasPixel := false
		for i := 3; i < len(img.Pix); i += 4 {
			if img.Pix[i] > 0 {
				hasPixel = true
				break
			}
		}
		if !hasPixel {
			t.Errorf("align %s expected pixels", align)
		}
	}
}
