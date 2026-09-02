package noisefield

import "math"

// This file ports the noise primitives the Gradient Studio shader relies
// on: Ashima Arts' 2D simplex noise, a 3-octave fBm on top of it, and the
// classic sin-dot hash. Everything is float64 (the GPU shader is float32);
// results therefore track the shader closely but are not bit-identical to
// it — that is expected and acceptable for the CPU core.

func mod289(x float64) float64 { return x - math.Floor(x*(1.0/289.0))*289.0 }

func permute(x float64) float64 { return mod289(((x * 34.0) + 1.0) * x) }

// snoise is Ashima Arts' 2D simplex noise ("webgl-noise", noise2D.glsl),
// ported scalar-for-scalar from the GLSL. Output is roughly in [-1, 1].
func snoise(vx, vy float64) float64 {
	const cx = 0.211324865405187 // (3-sqrt(3))/6
	const cy = 0.366025403784439 // 0.5*(sqrt(3)-1)
	const cz = -0.577350269189626
	const cw = 0.024390243902439 // 1/41

	// First corner.
	dv := (vx + vy) * cy
	ix := math.Floor(vx + dv)
	iy := math.Floor(vy + dv)

	di := (ix + iy) * cx
	x0x := vx - ix + di
	x0y := vy - iy + di

	// Other corners.
	var i1x, i1y float64
	if x0x > x0y {
		i1x, i1y = 1.0, 0.0
	} else {
		i1x, i1y = 0.0, 1.0
	}

	// x12 = x0.xyxy + C.xxzz, then x12.xy -= i1.
	x12x := x0x + cx - i1x
	x12y := x0y + cx - i1y
	x12z := x0x + cz
	x12w := x0y + cz

	// Permutations (avoid truncation in the permute polynomial).
	ix = mod289(ix)
	iy = mod289(iy)

	p0 := permute(iy + 0.0)
	p1 := permute(iy + i1y)
	p2 := permute(iy + 1.0)
	p0 = permute(p0 + ix + 0.0)
	p1 = permute(p1 + ix + i1x)
	p2 = permute(p2 + ix + 1.0)

	// Gradient contribution falloff.
	m0 := math.Max(0.5-(x0x*x0x+x0y*x0y), 0.0)
	m1 := math.Max(0.5-(x12x*x12x+x12y*x12y), 0.0)
	m2 := math.Max(0.5-(x12z*x12z+x12w*x12w), 0.0)
	m0 *= m0
	m0 *= m0
	m1 *= m1
	m1 *= m1
	m2 *= m2
	m2 *= m2

	// Gradients: 41 points uniformly over a line, mapped onto a diamond.
	t0 := 2.0*fract(p0*cw) - 1.0
	t1 := 2.0*fract(p1*cw) - 1.0
	t2 := 2.0*fract(p2*cw) - 1.0

	h0 := math.Abs(t0) - 0.5
	h1 := math.Abs(t1) - 0.5
	h2 := math.Abs(t2) - 0.5

	ox0 := math.Floor(t0 + 0.5)
	ox1 := math.Floor(t1 + 0.5)
	ox2 := math.Floor(t2 + 0.5)

	a0 := t0 - ox0
	a1 := t1 - ox1
	a2 := t2 - ox2

	// Normalise gradients implicitly by scaling m.
	m0 *= 1.79284291400159 - 0.85373472095314*(a0*a0+h0*h0)
	m1 *= 1.79284291400159 - 0.85373472095314*(a1*a1+h1*h1)
	m2 *= 1.79284291400159 - 0.85373472095314*(a2*a2+h2*h2)

	// Compute final noise value at P.
	gx := a0*x0x + h0*x0y
	gy := a1*x12x + h1*x12y
	gz := a2*x12z + h2*x12w

	return 130.0 * (m0*gx + m1*gy + m2*gz)
}

// fbm is the shader's fractal Brownian motion: 3 octaves of snoise, each
// octave at 2.03x frequency (plus a (1.7, 9.2) offset to decorrelate the
// lattice) and half amplitude, remapped from [-1,1]-ish to [0,1].
func fbm(px, py float64) float64 {
	v := 0.0
	amp := 0.5
	for i := 0; i < 3; i++ {
		v += amp * snoise(px, py)
		px = px*2.03 + 1.7
		py = py*2.03 + 9.2
		amp *= 0.5
	}
	return v*0.5 + 0.5
}

// hash is the shader's fract(sin(dot(p, (12.9898, 78.233))) * 43758.5453).
func hash(x, y float64) float64 {
	return fract(math.Sin(x*12.9898+y*78.233) * 43758.5453)
}
