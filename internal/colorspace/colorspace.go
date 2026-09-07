// Package colorspace converts between sRGB and the perceptual spaces
// BitBrush uses for dithering, quantization, gradient interpolation and
// palette extraction. Functions take float64 channels in [0,1] unless noted.
package colorspace

import "math"

// LinearizeChannel converts one gamma-encoded sRGB channel to linear light.
func LinearizeChannel(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// EncodeChannel is the inverse of LinearizeChannel.
func EncodeChannel(c float64) float64 {
	if c <= 0.0031308 {
		return c * 12.92
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}

// linLUT is LinearizeChannel evaluated at every 8-bit sRGB code. It lets the
// hot paths (OKLab conversion, palette mapping, grayscale) skip a per-pixel
// math.Pow. Because the input domain is exactly {0/255 .. 255/255}, a lookup
// here is bit-identical to calling LinearizeChannel(float64(c)/255).
var linLUT = func() [256]float64 {
	var t [256]float64
	for i := range t {
		t[i] = LinearizeChannel(float64(i) / 255)
	}
	return t
}()

// Linearize8 returns the linear-light value of one gamma-encoded 8-bit sRGB
// channel. Equivalent to LinearizeChannel(float64(c)/255), table-backed.
func Linearize8(c uint8) float64 { return linLUT[c] }

// encU8LUT maps a linear-light value in [0,1], quantised to encLUTSteps
// buckets, to its gamma-encoded 8-bit code. Used where the result is an 8-bit
// pixel anyway (grayscale tone), so bucket rounding is below output precision.
const encLUTSteps = 4096

var encU8LUT = func() [encLUTSteps + 1]uint8 {
	var t [encLUTSteps + 1]uint8
	for i := range t {
		v := EncodeChannel(float64(i)/encLUTSteps) * 255
		switch {
		case v <= 0:
			t[i] = 0
		case v >= 255:
			t[i] = 255
		default:
			t[i] = uint8(v + 0.5)
		}
	}
	return t
}()

// Encode8 gamma-encodes a linear-light value in [0,1] to an 8-bit sRGB code,
// clamping out-of-range input. Table-backed inverse of Linearize8.
func Encode8(y float64) uint8 {
	if y <= 0 {
		return encU8LUT[0]
	}
	if y >= 1 {
		return encU8LUT[encLUTSteps]
	}
	return encU8LUT[int(y*encLUTSteps+0.5)]
}

// OKLab is a colour in the OKLab perceptual space (Björn Ottosson, 2020):
// L is lightness in ~[0,1], A and B are the green–red and blue–yellow axes.
// Euclidean distance in OKLab is a good approximation of perceived difference.
type OKLab struct{ L, A, B float64 }

// RGBToOKLab converts a gamma-encoded 8-bit sRGB triple to OKLab.
func RGBToOKLab(r, g, b uint8) OKLab {
	lr := linLUT[r]
	lg := linLUT[g]
	lb := linLUT[b]

	l := math.Cbrt(0.4122214708*lr + 0.5363325363*lg + 0.0514459929*lb)
	m := math.Cbrt(0.2119034982*lr + 0.6806995451*lg + 0.1073969566*lb)
	s := math.Cbrt(0.0883024619*lr + 0.2817188376*lg + 0.6299787005*lb)

	return OKLab{
		L: 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		A: 1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		B: 0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}
}
