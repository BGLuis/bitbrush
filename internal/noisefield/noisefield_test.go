package noisefield

import (
	"bytes"
	"image"
	"testing"
)

func organicParams() Params {
	return Params{
		Field:   FieldFlow,
		Style:   StyleHolographic,
		Texture: TextureGrain,
		Stops:   []RGB{{10, 20, 40}, {220, 80, 140}},
		Spots: []Spot{
			{Color: RGB{255, 40, 40}, X: 0.25, Y: 0.30},
			{Color: RGB{40, 255, 40}, X: 0.75, Y: 0.28},
			{Color: RGB{40, 40, 255}, X: 0.50, Y: 0.80},
		},
		AngleDeg:   35,
		Scale:      55,
		Distortion: 50,
		Seed:       12345,
		Time:       0.0,
	}
}

func checkOpaque(t *testing.T, img *image.RGBA) {
	t.Helper()
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 255 {
			t.Fatalf("alpha byte %d = %d, want 255", i, img.Pix[i])
		}
	}
}

func TestRenderDeterministic(t *testing.T) {
	p := organicParams()
	a := Render(p, 96, 72)
	b := Render(p, 96, 72)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("Render is not deterministic for identical Params")
	}

	// A geometric field too.
	g := Params{Field: FieldConic, Stops: []RGB{{0, 0, 0}, {255, 0, 128}, {255, 255, 255}}, AngleDeg: 20}
	if !bytes.Equal(Render(g, 80, 80).Pix, Render(g, 80, 80).Pix) {
		t.Fatal("Render is not deterministic for a geometric field")
	}
}

func TestSizeAndOpacity(t *testing.T) {
	for _, sz := range [][2]int{{1, 1}, {64, 48}, {33, 100}, {200, 7}} {
		p := organicParams()
		img := Render(p, sz[0], sz[1])
		if img.Bounds() != image.Rect(0, 0, sz[0], sz[1]) {
			t.Fatalf("bounds = %v, want %v", img.Bounds(), image.Rect(0, 0, sz[0], sz[1]))
		}
		checkOpaque(t, img)
	}
}

func TestNonPositiveSize(t *testing.T) {
	if got := Render(organicParams(), 0, 10).Bounds(); got != image.Rect(0, 0, 0, 0) {
		t.Fatalf("w=0 bounds = %v, want empty", got)
	}
	if got := Render(organicParams(), 10, -3).Bounds(); got != image.Rect(0, 0, 0, 0) {
		t.Fatalf("h<0 bounds = %v, want empty", got)
	}
}

func TestTimeAffectsOrganicOnly(t *testing.T) {
	p := organicParams()
	p.Field = FieldFlow
	p.Time = 0
	a := Render(p, 48, 36)
	p.Time = 1
	b := Render(p, 48, 36)
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("Time did not change FieldFlow output")
	}

	g := Params{Field: FieldLinear, Stops: []RGB{{0, 0, 0}, {255, 255, 255}}, AngleDeg: 90}
	g.Time = 0
	c := Render(g, 48, 36)
	g.Time = 7.5
	d := Render(g, 48, 36)
	if !bytes.Equal(c.Pix, d.Pix) {
		t.Fatal("Time changed a geometric (FieldLinear) field")
	}
}

func TestSeedAffectsOrganicOnly(t *testing.T) {
	p := organicParams()
	p.Field = FieldMesh
	p.Seed = 1
	a := Render(p, 48, 36)
	p.Seed = 2
	b := Render(p, 48, 36)
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("Seed did not change FieldMesh output")
	}

	g := Params{Field: FieldRadial, Stops: []RGB{{0, 0, 0}, {255, 255, 255}}, Seed: 1}
	c := Render(g, 48, 36)
	g.Seed = 999999
	d := Render(g, 48, 36)
	if !bytes.Equal(c.Pix, d.Pix) {
		t.Fatal("Seed changed a geometric (FieldRadial) field")
	}
}

func TestSeedReduceStable(t *testing.T) {
	// Negative and huge seeds must still be finite and in a sane range.
	for _, s := range []int64{0, 1, -1, 99999, 100000, -100000, 1 << 60, -(1 << 60)} {
		v := seedReduce(s)
		if v < 7.3 || v >= 100000+7.3 {
			t.Fatalf("seedReduce(%d) = %v out of expected range", s, v)
		}
	}
}

func TestLinearGradientEdges(t *testing.T) {
	const w, h = 64, 48
	black := []RGB{{0, 0, 0}, {255, 255, 255}}

	// AngleDeg 90 => left-to-right: left edge dark, right edge light.
	p := Params{Field: FieldLinear, Stops: black, AngleDeg: 90}
	img := Render(p, w, h)
	left := img.RGBAAt(0, h/2)
	right := img.RGBAAt(w-1, h/2)
	if left.R > 48 {
		t.Fatalf("AngleDeg=90 left edge R=%d, want near black", left.R)
	}
	if right.R < 210 {
		t.Fatalf("AngleDeg=90 right edge R=%d, want near white", right.R)
	}

	// AngleDeg 270 => reversed.
	p.AngleDeg = 270
	img = Render(p, w, h)
	left = img.RGBAAt(0, h/2)
	right = img.RGBAAt(w-1, h/2)
	if left.R < 210 {
		t.Fatalf("AngleDeg=270 left edge R=%d, want near white", left.R)
	}
	if right.R > 48 {
		t.Fatalf("AngleDeg=270 right edge R=%d, want near black", right.R)
	}
}

func TestSingleStopIsFlat(t *testing.T) {
	p := Params{Field: FieldDiamond, Stops: []RGB{{200, 100, 50}}}
	img := Render(p, 40, 40)
	first := img.RGBAAt(0, 0)
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			if img.RGBAAt(x, y) != first {
				t.Fatalf("single-stop field not flat at (%d,%d)", x, y)
			}
		}
	}
}

func TestMissingStopsIsGrey(t *testing.T) {
	// No stops on a geometric field -> mid grey everywhere.
	p := Params{Field: FieldRadial}
	img := Render(p, 32, 32)
	c := img.RGBAAt(16, 16)
	if c.R != c.G || c.G != c.B {
		t.Fatalf("missing stops not neutral grey: %+v", c)
	}
	if c.R < 100 || c.R > 160 {
		t.Fatalf("missing stops grey R=%d, want mid grey", c.R)
	}
}

func TestMissingSpotsRenders(t *testing.T) {
	p := Params{Field: FieldFreeform, Scale: 40, Distortion: 40}
	img := Render(p, 48, 36)
	checkOpaque(t, img)
}

func TestAllEnumValuesRender(t *testing.T) {
	const w, h = 64, 48
	base := organicParams()

	for f := FieldLinear; f <= FieldFlow; f++ {
		for tx := TextureSmooth; tx <= TexturePaper; tx++ {
			p := base
			p.Field = f
			p.Texture = tx
			img := Render(p, w, h)
			if img.Bounds() != image.Rect(0, 0, w, h) {
				t.Fatalf("field=%d texture=%d bounds %v", f, tx, img.Bounds())
			}
			checkOpaque(t, img)
		}
	}

	for s := StyleMetallic; s <= StyleRainbow; s++ {
		p := base
		p.Field = FieldFlow
		p.Style = s
		img := Render(p, w, h)
		checkOpaque(t, img)
	}
}

func BenchmarkRenderFlow512(b *testing.B) {
	p := Params{
		Field:   FieldFlow,
		Style:   StyleHolographic,
		Texture: TextureGrain,
		Stops:   []RGB{{20, 20, 40}, {200, 50, 120}},
		Spots: []Spot{
			{Color: RGB{255, 0, 0}, X: 0.2, Y: 0.3},
			{Color: RGB{0, 255, 0}, X: 0.8, Y: 0.25},
			{Color: RGB{0, 0, 255}, X: 0.5, Y: 0.8},
			{Color: RGB{255, 255, 0}, X: 0.7, Y: 0.6},
		},
		AngleDeg:   45,
		Scale:      60,
		Distortion: 50,
		Seed:       12345,
		Time:       1.5,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Render(p, 512, 512)
	}
}
