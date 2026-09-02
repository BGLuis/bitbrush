package colorspace

import "math"

// achromatic is the chroma below which hue is treated as undefined and
// reported as 0. Chosen well above the float noise a grey sRGB triple
// produces after conversion, but far below any visible chroma.
const achromatic = 1e-7

// to8 takes a gamma-encoded channel value (nominally [0,1]), scales it to
// 8-bit, rounds to the nearest level, and clamps to [0,255].
func to8(v float64) uint8 {
	n := math.Round(v * 255)
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return uint8(n)
}

// OKLabToLinearRGB is the inverse of the linear-RGB -> OKLab step used by
// RGBToOKLab (Ottosson's matrices). It returns linear-light RGB; values may
// fall outside [0,1] when the colour lies outside the sRGB gamut.
func OKLabToLinearRGB(c OKLab) (r, g, b float64) {
	l_ := c.L + 0.3963377774*c.A + 0.2158037573*c.B
	m_ := c.L - 0.1055613458*c.A - 0.0638541728*c.B
	s_ := c.L - 0.0894841775*c.A - 1.2914855480*c.B

	l := l_ * l_ * l_
	m := m_ * m_ * m_
	s := s_ * s_ * s_

	r = +4.0767416621*l - 3.3077115913*m + 0.2309699292*s
	g = -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
	b = -0.0041960863*l - 0.7034186147*m + 1.7076147010*s
	return
}

// OKLabToRGB converts OKLab to a gamma-encoded 8-bit sRGB triple:
// OKLabToLinearRGB, then EncodeChannel, round, and clamp to [0,255].
func OKLabToRGB(c OKLab) (r, g, b uint8) {
	lr, lg, lb := OKLabToLinearRGB(c)
	return to8(EncodeChannel(lr)), to8(EncodeChannel(lg)), to8(EncodeChannel(lb))
}

// OKLCh is the cylindrical form of OKLab: L is lightness, C >= 0 is chroma,
// and H is hue in DEGREES in [0,360).
type OKLCh struct{ L, C, H float64 }

// OKLabToOKLCh converts OKLab to cylindrical OKLCh. For achromatic colours
// (C ~ 0) the hue is undefined and reported as 0.
func OKLabToOKLCh(c OKLab) OKLCh {
	C := math.Hypot(c.A, c.B)
	H := 0.0
	if C > achromatic {
		H = math.Atan2(c.B, c.A) * 180 / math.Pi
		if H < 0 {
			H += 360
		}
	}
	return OKLCh{L: c.L, C: C, H: H}
}

// OKLChToOKLab converts cylindrical OKLCh back to OKLab.
func OKLChToOKLab(c OKLCh) OKLab {
	h := c.H * math.Pi / 180
	return OKLab{L: c.L, A: c.C * math.Cos(h), B: c.C * math.Sin(h)}
}
