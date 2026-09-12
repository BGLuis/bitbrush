package filters

import (
	"fmt"
	"image"
	"math"
	"strconv"
)

// chromaKey zeros the alpha channel of pixels whose color is within
// `tolerance` of `color` (in sRGB Euclidean distance, 0..1 per channel).
// An optional `feather` (0..1) softens the edge by blending partial alpha.
//
// Params:
//
//	color      string  "#rrggbb" target chroma color (default "#00ff00")
//	tolerance  float   0..1 matching radius in sRGB (default 0.3)
//	feather    float   0..1 softening zone outside tolerance (default 0.05)
//	invert     bool    keep matched pixels instead of removing them (default false)
func chromaKey(src *image.RGBA, p Params) (*image.RGBA, error) {
	colorHex := p.String("color", "#00ff00")
	tolerance := clampF(p.Float("tolerance", 0.3), 0, 1)
	feather := clampF(p.Float("feather", 0.05), 0, 1)
	invert := p.Bool("invert", false)

	tr, tg, tb, err := parseHex(colorHex)
	if err != nil {
		return nil, fmt.Errorf("chroma-key: %w", err)
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)

	inner := tolerance
	outer := tolerance + feather

	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			r, g, bl, a := src.Pix[si], src.Pix[si+1], src.Pix[si+2], src.Pix[si+3]

			dr := (float64(r) - float64(tr)) / 255.0
			dg := (float64(g) - float64(tg)) / 255.0
			db := (float64(bl) - float64(tb)) / 255.0
			dist := math.Sqrt(dr*dr+dg*dg+db*db) / math.Sqrt(3)

			var newAlpha uint8
			if dist <= inner {
				// fully matched
				if invert {
					newAlpha = a
				} else {
					newAlpha = 0
				}
			} else if outer > inner && dist <= outer {
				// feather zone
				t := (dist - inner) / (outer - inner) // 0 at inner, 1 at outer
				if invert {
					newAlpha = uint8(float64(a) * (1 - t))
				} else {
					newAlpha = uint8(float64(a) * t)
				}
			} else {
				// outside: keep original
				if invert {
					newAlpha = 0
				} else {
					newAlpha = a
				}
			}

			dst.Pix[di] = r
			dst.Pix[di+1] = g
			dst.Pix[di+2] = bl
			dst.Pix[di+3] = newAlpha
			si += 4
			di += 4
		}
	}
	return dst, nil
}

// lumaKey zeros the alpha of pixels whose luminance (Rec.709) is below
// `threshold`. Useful for removing black or white backgrounds from images
// that already have a solid-tone bg.
//
// Params:
//
//	threshold  float   0..1 luminance cut-off (default 0.1)
//	feather    float   0..1 softening zone above threshold (default 0.05)
//	invert     bool    keep dark pixels, remove bright ones (default false)
func lumaKey(src *image.RGBA, p Params) (*image.RGBA, error) {
	threshold := clampF(p.Float("threshold", 0.1), 0, 1)
	feather := clampF(p.Float("feather", 0.05), 0, 1)
	invert := p.Bool("invert", false)

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)

	inner := threshold
	outer := threshold + feather

	for y := 0; y < h; y++ {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		di := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			r, g, bl, a := src.Pix[si], src.Pix[si+1], src.Pix[si+2], src.Pix[si+3]

			// Rec.709 luma, gamma-encoded (fast approximation)
			lumF := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(bl)) / 255.0

			var newAlpha uint8
			if lumF <= inner {
				if invert {
					newAlpha = a
				} else {
					newAlpha = 0
				}
			} else if outer > inner && lumF <= outer {
				t := (lumF - inner) / (outer - inner)
				if invert {
					newAlpha = uint8(float64(a) * (1 - t))
				} else {
					newAlpha = uint8(float64(a) * t)
				}
			} else {
				if invert {
					newAlpha = 0
				} else {
					newAlpha = a
				}
			}

			dst.Pix[di] = r
			dst.Pix[di+1] = g
			dst.Pix[di+2] = bl
			dst.Pix[di+3] = newAlpha
			si += 4
			di += 4
		}
	}
	return dst, nil
}

// parseHex decodes "#rrggbb" or "rrggbb" into r, g, b uint8.
func parseHex(s string) (uint8, uint8, uint8, error) {
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	if len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("bad hex color %q", "#"+s)
	}
	rv, err := strconv.ParseUint(s[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("bad hex color %q: %w", "#"+s, err)
	}
	gv, err := strconv.ParseUint(s[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("bad hex color %q: %w", "#"+s, err)
	}
	bv, err := strconv.ParseUint(s[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("bad hex color %q: %w", "#"+s, err)
	}
	return uint8(rv), uint8(gv), uint8(bv), nil
}

func init() {
	Register("chroma-key", chromaKey)
	Register("luma-key", lumaKey)
}
