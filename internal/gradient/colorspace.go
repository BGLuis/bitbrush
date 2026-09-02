package gradient

import "bitbrush/internal/colorspace"

// This file is the only bridge to package colorspace: small helpers that
// keep gradient.go readable and confine the conversion vocabulary.

// encLerp lerps one channel in linear light and re-encodes to 8-bit.
func encLerp(a, b uint8, u float64) uint8 {
	la := colorspace.LinearizeChannel(float64(a) / 255)
	lb := colorspace.LinearizeChannel(float64(b) / 255)
	return clampByte(colorspace.EncodeChannel(lerp(la, lb, u)) * 255)
}

func rgbToHSL(c RGB) colorspace.HSL { return colorspace.RGBToHSL(c.R, c.G, c.B) }

func hslToRGB(h, s, l float64) (r, g, b uint8) {
	return colorspace.HSLToRGB(colorspace.HSL{H: h, S: s, L: l})
}

func rgbToLab(c RGB) colorspace.Lab { return colorspace.RGBToLab(c.R, c.G, c.B) }

func labToRGB(l, a, b float64) (uint8, uint8, uint8) {
	return colorspace.LabToRGB(colorspace.Lab{L: l, A: a, B: b})
}

func rgbToLCh(c RGB) colorspace.LCh {
	return colorspace.LabToLCh(colorspace.RGBToLab(c.R, c.G, c.B))
}

func lchToRGB(l, ch, h float64) (uint8, uint8, uint8) {
	return colorspace.LabToRGB(colorspace.LChToLab(colorspace.LCh{L: l, C: ch, H: h}))
}

func rgbToOKLab(c RGB) colorspace.OKLab { return colorspace.RGBToOKLab(c.R, c.G, c.B) }

func okLabToRGB(l, a, b float64) (uint8, uint8, uint8) {
	return colorspace.OKLabToRGB(colorspace.OKLab{L: l, A: a, B: b})
}

func rgbToOKLCh(c RGB) colorspace.OKLCh {
	return colorspace.OKLabToOKLCh(colorspace.RGBToOKLab(c.R, c.G, c.B))
}

// okLChToRGB clamps the interpolated OKLCh back into the sRGB gamut (chroma
// reduction, holding L and H) before converting to 8-bit sRGB.
func okLChToRGB(l, c, h float64) (uint8, uint8, uint8) {
	in := colorspace.ClampToGamut(colorspace.OKLCh{L: l, C: c, H: h})
	return colorspace.OKLabToRGB(colorspace.OKLChToOKLab(in))
}
