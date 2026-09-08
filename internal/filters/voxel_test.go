package filters

import "testing"

func TestVoxelRunsAndCovers(t *testing.T) {
	src := gradientImg(41, 33)
	assertBoundsFull(t, "voxel", src, Params{"cell": 12.0})
	assertBoundsFull(t, "voxel", src, Params{"cell": 16.0, "heightFromLuma": true, "lightFlip": true, "outline": 2.0})
}

func TestVoxelNoMutateAndDeterministic(t *testing.T) {
	src := gradientImg(32, 24)
	p := Params{"cell": 14.0, "heightFromLuma": true, "outline": 1.0}
	assertNoMutate(t, "voxel", src, p)
	assertDeterministic(t, "voxel", src, p)
}

func TestVoxelDefault(t *testing.T) {
	assertDefaultMatches(t, "voxel", gradientImg(30, 30),
		Params{"cell": 20.0, "topShade": 1.0, "leftShade": 0.75, "rightShade": 0.55,
			"outline": 1.0, "outlineColor": "#0a0a0a", "heightFromLuma": false, "lightFlip": false, "background": "#000000"})
}

func BenchmarkVoxel1080p(b *testing.B) {
	src := gradientImg(1920, 1080)
	p := Params{"cell": 20.0, "outline": 1.0}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Apply("voxel", src, p); err != nil {
			b.Fatal(err)
		}
	}
}
