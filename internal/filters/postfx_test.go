package filters

import (
	"bytes"
	"testing"
)

func TestVignette(t *testing.T) {
	src := solid(41, 41, 200, 200, 200, 255)
	out, err := Apply("vignette", src, Params{})
	if err != nil {
		t.Fatal(err)
	}
	cen := out.PixOffset(20, 20)
	if out.Pix[cen] < 198 {
		t.Fatalf("centre darkened too much: %d", out.Pix[cen])
	}
	cor := out.PixOffset(0, 0)
	if out.Pix[cor] >= 200 {
		t.Fatalf("corner not darkened: %d", out.Pix[cor])
	}
	assertNoMutate(t, "vignette", src, Params{})
	assertDeterministic(t, "vignette", gradientImg(30, 20), Params{"strength": 0.5})
}

func TestScanlines(t *testing.T) {
	src := solid(8, 6, 100, 100, 100, 255)
	out, err := Apply("scanlines", src, Params{"spacing": 3.0, "thickness": 1.0, "darkness": 0.5, "opacity": 1.0})
	if err != nil {
		t.Fatal(err)
	}
	if out.Pix[out.PixOffset(0, 0)] != 50 {
		t.Fatalf("row 0 should be halved, got %d", out.Pix[out.PixOffset(0, 0)])
	}
	if out.Pix[out.PixOffset(0, 1)] != 100 {
		t.Fatalf("row 1 should be untouched, got %d", out.Pix[out.PixOffset(0, 1)])
	}
	assertNoMutate(t, "scanlines", src, Params{})
}

func TestGrain(t *testing.T) {
	src := gradientImg(40, 30)
	if flat, _ := Apply("grain", src, Params{"amount": 0.0}); !bytes.Equal(flat.Pix, src.Pix) {
		t.Fatal("amount 0 must be identity")
	}
	a, _ := Apply("grain", src, Params{"amount": 0.3, "seed": 1.0})
	b, _ := Apply("grain", src, Params{"amount": 0.3, "seed": 2.0})
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("different seeds must differ")
	}
	assertNoMutate(t, "grain", src, Params{"amount": 0.3})
	assertDeterministic(t, "grain", src, Params{"amount": 0.3, "seed": 7.0})
}

func TestBlur(t *testing.T) {
	src := gradientImg(40, 30)
	if id, _ := Apply("blur", src, Params{"radius": 0.0}); !bytes.Equal(id.Pix, src.Pix) {
		t.Fatal("radius 0 must be identity")
	}
	sol := solid(20, 20, 70, 90, 110, 255)
	if out, _ := Apply("blur", sol, Params{"radius": 4.0}); !bytes.Equal(out.Pix, sol.Pix) {
		t.Fatal("solid image must survive a blur unchanged")
	}
	assertNoMutate(t, "blur", src, Params{"radius": 3.0})
	assertDeterministic(t, "blur", src, Params{"radius": 3.0, "type": "gaussian"})
}

func TestColorOverlay(t *testing.T) {
	src := gradientImg(30, 20)
	if id, _ := Apply("coloroverlay", src, Params{"opacity": 0.0}); !bytes.Equal(id.Pix, src.Pix) {
		t.Fatal("opacity 0 must be identity")
	}
	for _, m := range []string{"normal", "multiply", "screen", "overlay", "color"} {
		assertBoundsFull(t, "coloroverlay", src, Params{"mode": m, "opacity": 0.5})
	}
	assertNoMutate(t, "coloroverlay", src, Params{"opacity": 0.4})
	assertDeterministic(t, "coloroverlay", src, Params{"mode": "multiply", "opacity": 0.4})
}

func TestBloom(t *testing.T) {
	// A mid-grey solid with threshold 1 has nothing bright -> unchanged.
	sol := solid(24, 24, 120, 120, 120, 255)
	if out, _ := Apply("bloom", sol, Params{"threshold": 1.0}); !bytes.Equal(out.Pix, sol.Pix) {
		t.Fatal("no bright pixels -> bloom is a no-op")
	}
	src := gradientImg(40, 30)
	assertBoundsFull(t, "bloom", src, Params{"threshold": 0.5, "radius": 6.0})
	assertNoMutate(t, "bloom", src, Params{"threshold": 0.5})
	assertDeterministic(t, "bloom", src, Params{"threshold": 0.5, "radius": 6.0})
}

func TestChromatic(t *testing.T) {
	src := gradientImg(40, 30)
	if id, _ := Apply("chromatic", src, Params{"strength": 0.0}); !bytes.Equal(id.Pix, src.Pix) {
		t.Fatal("strength 0 must be identity")
	}
	assertBoundsFull(t, "chromatic", src, Params{"strength": 4.0})
	assertBoundsFull(t, "chromatic", src, Params{"strength": 4.0, "radial": false})
	assertNoMutate(t, "chromatic", src, Params{"strength": 4.0})
	assertDeterministic(t, "chromatic", src, Params{"strength": 4.0})
}

func TestDust(t *testing.T) {
	src := gradientImg(40, 30)
	if id, _ := Apply("dust", src, Params{"density": 0.0, "scratches": false}); !bytes.Equal(id.Pix, src.Pix) {
		t.Fatal("no specks, no scratches -> identity")
	}
	a, _ := Apply("dust", src, Params{"density": 0.01, "seed": 1.0})
	b, _ := Apply("dust", src, Params{"density": 0.01, "seed": 2.0})
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("different seeds must differ")
	}
	assertNoMutate(t, "dust", src, Params{"density": 0.01})
	assertDeterministic(t, "dust", src, Params{"density": 0.01, "seed": 3.0})
}

func TestCRT(t *testing.T) {
	src := gradientImg(48, 36)
	assertBoundsFull(t, "crt", src, Params{})
	assertBoundsFull(t, "crt", src, Params{"curvature": 0.3, "zoom": 0.9, "mask": 0.4, "vignette": 0.5})
	assertNoMutate(t, "crt", src, Params{})
	assertDeterministic(t, "crt", src, Params{"curvature": 0.2})
}

func BenchmarkBlur1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"radius": 8.0, "type": "gaussian"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("blur", src, p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBloom1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"threshold": 0.7, "radius": 12.0}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("bloom", src, p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCRT1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("crt", src, Params{}); err != nil {
			b.Fatal(err)
		}
	}
}
