package palette

import "testing"

func TestSortPaletteLumaAscending(t *testing.T) {
	in := []RGB{{255, 255, 255}, {0, 0, 0}, {200, 20, 20}, {20, 200, 20}}
	got := SortPalette(in, SortLuma, nil)
	for i := 1; i < len(got); i++ {
		if luma(got[i-1]) > luma(got[i]) {
			t.Fatalf("not ascending by luma at %d: %+v", i, got)
		}
	}
	if got[0] != (RGB{0, 0, 0}) {
		t.Fatalf("darkest colour not first: %+v", got[0])
	}
}

func TestSortPaletteHueAscending(t *testing.T) {
	red, green, blue := RGB{255, 0, 0}, RGB{0, 255, 0}, RGB{0, 0, 255}
	got := SortPalette([]RGB{blue, red, green}, SortHue, nil)
	want := []RGB{red, green, blue} // HSL hues 0, 120, 240
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("hue sort = %+v, want %+v", got, want)
		}
	}
}

func TestSortPalettePopulationMostUsedFirst(t *testing.T) {
	red, green, blue := RGB{220, 20, 20}, RGB{20, 220, 20}, RGB{20, 20, 220}
	// red covers 3 of 5 bands, green and blue one each.
	img := stripes(100, 10, red, red, red, green, blue)
	got := SortPalette([]RGB{green, blue, red}, SortPopulation, img)
	if got[0] != red {
		t.Fatalf("most-used colour not first: %+v", got)
	}
}

func TestSortPaletteReturnsNewSlice(t *testing.T) {
	in := []RGB{{255, 255, 255}, {0, 0, 0}, {128, 128, 128}}
	snapshot := append([]RGB(nil), in...)
	got := SortPalette(in, SortLuma, nil)
	if !eqPal(in, snapshot) {
		t.Fatalf("input mutated: %+v, was %+v", in, snapshot)
	}
	if len(got) > 0 && &got[0] == &in[0] {
		t.Fatal("result aliases the input backing array")
	}
}

func TestSortPaletteNonePreservesOrder(t *testing.T) {
	in := []RGB{{9, 9, 9}, {1, 1, 1}, {5, 5, 5}}
	got := SortPalette(in, SortNone, nil)
	if !eqPal(got, in) {
		t.Fatalf("SortNone reordered: %+v", got)
	}
}
