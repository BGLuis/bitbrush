package filters

import (
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// asciiDefaultRamp goes dark -> light; index 0 is the densest glyph.
const asciiDefaultRamp = " .:-=+*#%@"

// ASCII renders src as coloured ASCII art: each cellWidth x cellHeight
// block is replaced by one glyph from `ramp`, chosen by the block's mean
// luma, tinted by the block's mean colour (or a flat foreground) and
// composited over `background`. Alpha is carried through from src.
//
// Params:
//
//	cellWidth   int,    px, default 8
//	cellHeight  int,    px, default 12
//	ramp        string, dark-to-light glyph ramp, default " .:-=+*#%@"
//	colored     bool,   tint by the block's mean colour vs a flat grey (default true)
//	invert      bool,   reverse the ramp (default false)
//	background  string, hex colour "#rgb" or "#rrggbb" (default "#000000")
func ASCII(src *image.RGBA, p Params) (*image.RGBA, error) {
	cellW, cellH, ramp := asciiParams(p)

	bg, ok := parseHexColor(p.String("background", "#000000"))
	if !ok {
		bg = color.RGBA{A: 255}
	}
	colored := p.Bool("colored", true)
	flatFG := color.RGBA{R: 224, G: 224, B: 224, A: 255}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := newLike(src)
	if w == 0 || h == 0 {
		return dst, nil
	}

	glyphs := renderGlyphMasks(ramp)
	cells, cols, rows := asciiCells(src, cellW, cellH, len(ramp))

	for ry := 0; ry < rows; ry++ {
		cy := ry * cellH
		bh := cellH
		if cy+bh > h {
			bh = h - cy
		}
		for rx := 0; rx < cols; rx++ {
			cx := rx * cellW
			bw := cellW
			if cx+bw > w {
				bw = w - cx
			}

			cell := cells[ry][rx]
			fg := flatFG
			if colored {
				fg = color.RGBA{R: cell.avg.R, G: cell.avg.G, B: cell.avg.B, A: 255}
			}
			blitGlyph(dst, src, cx, cy, bw, bh, glyphs[ramp[cell.idx]], fg, bg)
		}
	}
	return dst, nil
}

func init() { Register("ascii", ASCII) }

// ASCIIText renders src as plain-text ASCII art — the same cell/ramp
// selection as ASCII (no font, no colour), one line per row of cells. It
// is not an image Filter, so it isn't in the registry; the WASM adapter
// exposes it through its own global for the "copy as text" export.
func ASCIIText(src *image.RGBA, p Params) (string, error) {
	cellW, cellH, ramp := asciiParams(p)

	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return "", nil
	}
	cells, cols, rows := asciiCells(src, cellW, cellH, len(ramp))

	var sb strings.Builder
	sb.Grow(rows * (cols + 1))
	for ry := 0; ry < rows; ry++ {
		for rx := 0; rx < cols; rx++ {
			sb.WriteRune(ramp[cells[ry][rx].idx])
		}
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}

// asciiParams reads and clamps the parameters ASCII and ASCIIText share.
func asciiParams(p Params) (cellW, cellH int, ramp []rune) {
	cellW = p.Int("cellWidth", 8)
	cellH = p.Int("cellHeight", 12)
	if cellW < 1 {
		cellW = 1
	}
	if cellH < 1 {
		cellH = 1
	}

	ramp = []rune(p.String("ramp", asciiDefaultRamp))
	if len(ramp) == 0 {
		ramp = []rune(asciiDefaultRamp)
	}
	if p.Bool("invert", false) {
		reverseRunes(ramp)
	}
	return cellW, cellH, ramp
}

// asciiCell is what one grid cell needs: which ramp glyph it picked and
// the cell's mean colour.
type asciiCell struct {
	idx int
	avg color.RGBA
}

// asciiCells buckets src into cellW x cellH cells (the last row/column may
// be smaller) and picks a ramp index per cell from its mean luma.
func asciiCells(src *image.RGBA, cellW, cellH, rampLen int) (cells [][]asciiCell, cols, rows int) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	cols = (w + cellW - 1) / cellW
	rows = (h + cellH - 1) / cellH

	cells = make([][]asciiCell, rows)
	for ry := 0; ry < rows; ry++ {
		row := make([]asciiCell, cols)
		cy := ry * cellH
		for rx := 0; rx < cols; rx++ {
			cx := rx * cellW
			avg := blockAverage(src, b.Min.X+cx, b.Min.Y+cy, cellW, cellH)
			// math.Round, not truncation: plain int() would drop luma=255
			// to the second-brightest glyph on a hair of float64 error
			// (0.2126+0.7152+0.0722 isn't exactly 1 in float64).
			idx := int(math.Round(luma(avg.R, avg.G, avg.B) / 255 * float64(rampLen-1)))
			switch {
			case idx < 0:
				idx = 0
			case idx >= rampLen:
				idx = rampLen - 1
			}
			row[rx] = asciiCell{idx: idx, avg: avg}
		}
		cells[ry] = row
	}
	return cells, cols, rows
}

// renderGlyphMasks rasterises each distinct rune in runes once, using
// golang.org/x/image/font/basicfont's built-in bitmap face (no font file
// needed — it works in the WASM build same as anywhere else), and returns
// each as an alpha coverage mask sized to that glyph's own metrics.
func renderGlyphMasks(runes []rune) map[rune]*image.Alpha {
	face := basicfont.Face7x13
	metrics := face.Metrics()
	gh := metrics.Height.Ceil()
	if gh < 1 {
		gh = 1
	}

	out := make(map[rune]*image.Alpha, len(runes))
	for _, r := range runes {
		if _, done := out[r]; done {
			continue
		}
		gw := gh/2 + 1 // fallback advance for a glyph the face has no metric for
		if adv, ok := face.GlyphAdvance(r); ok && adv.Ceil() > 0 {
			gw = adv.Ceil()
		}

		mask := image.NewAlpha(image.Rect(0, 0, gw, gh))
		d := font.Drawer{
			Dst:  mask,
			Src:  image.White,
			Face: face,
			Dot:  fixed.Point26_6{X: 0, Y: metrics.Ascent},
		}
		d.DrawString(string(r))
		out[r] = mask
	}
	return out
}

// blitGlyph paints the bw x bh cell at (cx,cy) in dst, nearest-neighbour
// sampling glyph as a coverage mask and lerping between bg and fg by that
// coverage. dst's alpha is src's, pixel for pixel — the glyph never
// changes opacity, only colour.
func blitGlyph(dst, src *image.RGBA, cx, cy, bw, bh int, glyph *image.Alpha, fg, bg color.RGBA) {
	gb := glyph.Bounds()
	gw, gh := gb.Dx(), gb.Dy()
	b := src.Bounds()
	for y := 0; y < bh; y++ {
		gy := gb.Min.Y + y*gh/bh
		di := dst.PixOffset(cx, cy+y)
		si := src.PixOffset(b.Min.X+cx, b.Min.Y+cy+y)
		for x := 0; x < bw; x++ {
			gx := gb.Min.X + x*gw/bw
			cov := float64(glyph.Pix[glyph.PixOffset(gx, gy)]) / 255

			dst.Pix[di] = lerpByte(bg.R, fg.R, cov)
			dst.Pix[di+1] = lerpByte(bg.G, fg.G, cov)
			dst.Pix[di+2] = lerpByte(bg.B, fg.B, cov)
			dst.Pix[di+3] = src.Pix[si+3]
			di += 4
			si += 4
		}
	}
}

func lerpByte(a, b uint8, t float64) uint8 {
	return clampU8(float64(a) + (float64(b)-float64(a))*t)
}

func reverseRunes(r []rune) {
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
}

// parseHexColor parses "#rgb" or "#rrggbb" (the leading # is optional).
// The result is always fully opaque; ok is false for anything else.
func parseHexColor(s string) (color.RGBA, bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 {
		expanded := make([]byte, 0, 6)
		for i := 0; i < 3; i++ {
			expanded = append(expanded, s[i], s[i])
		}
		s = string(expanded)
	}
	if len(s) != 6 {
		return color.RGBA{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.RGBA{}, false
	}
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}, true
}
