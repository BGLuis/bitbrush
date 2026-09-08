package filters

import "testing"

func TestLegoRunsAndCovers(t *testing.T) {
	src := gradientImg(37, 29)
	assertBoundsFull(t, "lego", src, Params{"cell": 8.0, "colors": 12.0})
	assertBoundsFull(t, "lego", src, Params{"cell": 10.0, "outline": false, "studContrast": 0.6})
}

func TestLegoNoMutateAndDeterministic(t *testing.T) {
	src := gradientImg(30, 22)
	p := Params{"cell": 9.0, "colors": 16.0, "studContrast": 0.35, "outline": true}
	assertNoMutate(t, "lego", src, p)
	assertDeterministic(t, "lego", src, p)
}

func TestLegoDefault(t *testing.T) {
	assertDefaultMatches(t, "lego", gradientImg(24, 24),
		Params{"cell": 16.0, "colors": 16.0, "studContrast": 0.35, "outline": true})
}

func BenchmarkLego1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"cell": 16.0, "colors": 16.0}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("lego", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
