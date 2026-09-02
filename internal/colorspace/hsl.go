package colorspace

import "math"

// HSL is a hue-saturation-lightness colour. H is in DEGREES in [0,360);
// S and L are in [0,1].
type HSL struct{ H, S, L float64 }

// RGBToHSL converts a gamma-encoded 8-bit sRGB triple to HSL. The
// conversion is purely geometric in gamma-encoded space (the usual web
// definition); it does not linearise. Achromatic inputs return H=0, S=0.
func RGBToHSL(r, g, b uint8) HSL {
	rf := float64(r) / 255
	gf := float64(g) / 255
	bf := float64(b) / 255

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	l := (max + min) / 2

	d := max - min
	if d == 0 {
		return HSL{H: 0, S: 0, L: l}
	}

	var s float64
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}

	var h float64
	switch max {
	case rf:
		h = (gf - bf) / d
		if gf < bf {
			h += 6
		}
	case gf:
		h = (bf-rf)/d + 2
	default:
		h = (rf-gf)/d + 4
	}
	h *= 60

	return HSL{H: h, S: s, L: l}
}

// HSLToRGB converts HSL back to a gamma-encoded 8-bit sRGB triple, with the
// channels rounded and clamped to [0,255].
func HSLToRGB(c HSL) (r, g, b uint8) {
	if c.S <= 0 {
		v := to8(c.L)
		return v, v, v
	}

	var q float64
	if c.L < 0.5 {
		q = c.L * (1 + c.S)
	} else {
		q = c.L + c.S - c.L*c.S
	}
	p := 2*c.L - q

	h := c.H / 360
	return to8(hue2rgb(p, q, h+1.0/3.0)),
		to8(hue2rgb(p, q, h)),
		to8(hue2rgb(p, q, h-1.0/3.0))
}

func hue2rgb(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}
