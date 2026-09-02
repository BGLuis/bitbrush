package noisefield

import "math"

// vec3 is a minimal RGB / xyz triple, mirroring GLSL's vec3 so the port
// reads close to the shader source.
type vec3 struct{ x, y, z float64 }

func (v vec3) addf(s float64) vec3 { return vec3{v.x + s, v.y + s, v.z + s} }
func (v vec3) mulf(s float64) vec3 { return vec3{v.x * s, v.y * s, v.z * s} }
func (v vec3) add(o vec3) vec3     { return vec3{v.x + o.x, v.y + o.y, v.z + o.z} }

// clampf is GLSL clamp for scalars.
func clampf(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// mix is GLSL mix / linear interpolation: a at t==0, b at t==1.
func mix(a, b, t float64) float64 { return a + (b-a)*t }

// mixv is component-wise mix on vec3.
func mixv(a, b vec3, t float64) vec3 {
	return vec3{mix(a.x, b.x, t), mix(a.y, b.y, t), mix(a.z, b.z, t)}
}

// grey builds vec3(s, s, s) — GLSL's vec3(scalar).
func grey(s float64) vec3 { return vec3{s, s, s} }

// fract is GLSL fract: x - floor(x). Always in [0,1).
func fract(x float64) float64 { return x - math.Floor(x) }

// modx is GLSL mod(x, y) = x - y*floor(x/y). Unlike Go's % / math.Mod it
// follows the sign of y, so negative x wraps into [0, y).
func modx(x, y float64) float64 { return x - y*math.Floor(x/y) }

// smoothstep is GLSL smoothstep: 0 below e0, 1 above e1, Hermite between.
func smoothstep(e0, e1, x float64) float64 {
	t := clampf((x-e0)/(e1-e0), 0.0, 1.0)
	return t * t * (3.0 - 2.0*t)
}

// lum is the shader's luminance: dot(c, vec3(.299, .587, .114)).
func lum(c vec3) float64 { return c.x*0.299 + c.y*0.587 + c.z*0.114 }

// hsv2rgb is the standard IQ hue/sat/value -> rgb used by the shader.
func hsv2rgb(h, s, v float64) vec3 {
	r := clampf(math.Abs(modx(h*6.0+0.0, 6.0)-3.0)-1.0, 0.0, 1.0)
	g := clampf(math.Abs(modx(h*6.0+4.0, 6.0)-3.0)-1.0, 0.0, 1.0)
	b := clampf(math.Abs(modx(h*6.0+2.0, 6.0)-3.0)-1.0, 0.0, 1.0)
	return vec3{v * mix(1.0, r, s), v * mix(1.0, g, s), v * mix(1.0, b, s)}
}

// hueShift rotates c about the achromatic axis vec3(0.57735) by angle a
// (radians) using Rodrigues' rotation formula, matching the shader.
func hueShift(c vec3, a float64) vec3 {
	const k = 0.57735
	cs := math.Cos(a)
	sn := math.Sin(a)
	// cross(vec3(k), c) with all components of the axis equal to k.
	crx := k*c.z - k*c.y
	cry := k*c.x - k*c.z
	crz := k*c.y - k*c.x
	kd := k*c.x + k*c.y + k*c.z
	return vec3{
		c.x*cs + crx*sn + k*kd*(1.0-cs),
		c.y*cs + cry*sn + k*kd*(1.0-cs),
		c.z*cs + crz*sn + k*kd*(1.0-cs),
	}
}
