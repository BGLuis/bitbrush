package colorspace

// gamutEps is the tolerance applied when testing whether a linear-light RGB
// triple sits inside the unit cube.
const gamutEps = 1e-4

// InGamutLinear reports whether a linear-light RGB triple lies within [0,1]
// on every channel, allowing a tiny epsilon (~1e-4) of slack.
func InGamutLinear(r, g, b float64) bool {
	return r >= -gamutEps && r <= 1+gamutEps &&
		g >= -gamutEps && g <= 1+gamutEps &&
		b >= -gamutEps && b <= 1+gamutEps
}

// oklchInGamut reports whether an OKLCh colour maps into the sRGB gamut.
func oklchInGamut(c OKLCh) bool {
	r, g, b := OKLabToLinearRGB(OKLChToOKLab(c))
	return InGamutLinear(r, g, b)
}

// ClampToGamut reduces the chroma of an OKLCh colour, holding L and H fixed,
// via binary search until OKLabToLinearRGB lands inside the sRGB gamut, and
// returns the in-gamut OKLCh. L is clamped into [0,1] first; if that leaves
// L at 0 or 1 (pure black or white) the result has C=0. An already in-gamut
// colour is returned essentially unchanged.
func ClampToGamut(c OKLCh) OKLCh {
	l := c.L
	if l < 0 {
		l = 0
	}
	if l > 1 {
		l = 1
	}

	if l <= 0 || l >= 1 {
		return OKLCh{L: l, C: 0, H: c.H}
	}

	cand := OKLCh{L: l, C: c.C, H: c.H}
	if oklchInGamut(cand) {
		return cand
	}

	lo, hi := 0.0, c.C
	for i := 0; i < 64; i++ {
		mid := (lo + hi) / 2
		if oklchInGamut(OKLCh{L: l, C: mid, H: c.H}) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return OKLCh{L: l, C: lo, H: c.H}
}
