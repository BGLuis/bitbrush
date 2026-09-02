package colorspace

import "math"

// D65 reference white (2 degree observer), matching the CIE Lab definition
// used across BitBrush.
const (
	xn = 0.95047
	yn = 1.0
	zn = 1.08883

	// delta = 6/29, the knee of the Lab companding function.
	labDelta = 6.0 / 29.0
)

// Lab is a CIE L*a*b* colour (D65, 2 degree observer). L is in [0,100];
// a and b are the green-red and blue-yellow axes, unbounded in practice.
type Lab struct{ L, A, B float64 }

// LCh is the cylindrical form of Lab: L in [0,100], C >= 0 chroma, and
// H hue in DEGREES in [0,360).
type LCh struct{ L, C, H float64 }

// labF is the forward companding function f(t) with the usual delta=6/29.
func labF(t float64) float64 {
	if t > labDelta*labDelta*labDelta {
		return math.Cbrt(t)
	}
	return t/(3*labDelta*labDelta) + 4.0/29.0
}

// labFInv is the inverse of labF.
func labFInv(t float64) float64 {
	if t > labDelta {
		return t * t * t
	}
	return 3 * labDelta * labDelta * (t - 4.0/29.0)
}

// RGBToLab converts a gamma-encoded 8-bit sRGB triple to CIE Lab:
// sRGB8 -> linear -> XYZ (D65) -> Lab.
func RGBToLab(r, g, b uint8) Lab {
	lr := LinearizeChannel(float64(r) / 255)
	lg := LinearizeChannel(float64(g) / 255)
	lb := LinearizeChannel(float64(b) / 255)

	x := 0.4124564*lr + 0.3575761*lg + 0.1804375*lb
	y := 0.2126729*lr + 0.7151522*lg + 0.0721750*lb
	z := 0.0193339*lr + 0.1191920*lg + 0.9503041*lb

	fx := labF(x / xn)
	fy := labF(y / yn)
	fz := labF(z / zn)

	return Lab{
		L: 116*fy - 16,
		A: 500 * (fx - fy),
		B: 200 * (fy - fz),
	}
}

// LabToRGB is the inverse of RGBToLab: Lab -> XYZ (D65) -> linear -> sRGB8,
// with the final channels rounded and clamped to [0,255].
func LabToRGB(c Lab) (r, g, b uint8) {
	fy := (c.L + 16) / 116
	fx := fy + c.A/500
	fz := fy - c.B/200

	x := xn * labFInv(fx)
	y := yn * labFInv(fy)
	z := zn * labFInv(fz)

	lr := 3.2404542*x - 1.5371385*y - 0.4985314*z
	lg := -0.9692660*x + 1.8760108*y + 0.0415560*z
	lb := 0.0556434*x - 0.2040259*y + 1.0572252*z

	return to8(EncodeChannel(lr)), to8(EncodeChannel(lg)), to8(EncodeChannel(lb))
}

// LabToLCh converts Lab to cylindrical LCh. For achromatic colours (C ~ 0)
// the hue is undefined and reported as 0.
func LabToLCh(c Lab) LCh {
	C := math.Hypot(c.A, c.B)
	H := 0.0
	if C > 1e-8 {
		H = math.Atan2(c.B, c.A) * 180 / math.Pi
		if H < 0 {
			H += 360
		}
	}
	return LCh{L: c.L, C: C, H: H}
}

// LChToLab converts cylindrical LCh back to Lab.
func LChToLab(c LCh) Lab {
	h := c.H * math.Pi / 180
	return Lab{L: c.L, A: c.C * math.Cos(h), B: c.C * math.Sin(h)}
}
