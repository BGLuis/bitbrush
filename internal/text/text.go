package text

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	fontRegular *opentype.Font
	fontMono    *opentype.Font
)

func init() {
	var err error
	fontRegular, err = opentype.Parse(goregular.TTF)
	if err != nil {
		panic("text: failed to parse goregular font: " + err.Error())
	}
	fontMono, err = opentype.Parse(gomono.TTF)
	if err != nil {
		panic("text: failed to parse gomono font: " + err.Error())
	}
}

// Params defines the properties for rendering text onto an RGBA canvas.
type Params struct {
	Content    string  `json:"content"`
	FontFamily string  `json:"fontFamily"` // "go" | "gomono"
	Size       float64 `json:"size"`       // Font size in points (DPI 72)
	Color      string  `json:"color"`      // Hex string #RGB, #RGBA, #RRGGBB, #RRGGBBAA
	X          float64 `json:"x"`          // Horizontal anchor 0..1
	Y          float64 `json:"y"`          // Vertical anchor 0..1
	Align      string  `json:"align"`      // "left" | "center" | "right"
	Opacity    float64 `json:"opacity"`    // 0..1
}

// Render draws the text specified by p onto a transparent RGBA image of size w x h.
func Render(p Params, w, h int) (*image.RGBA, error) {
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("text.Render: invalid dimensions %dx%d", w, h)
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	if strings.TrimSpace(p.Content) == "" {
		return dst, nil
	}

	size := p.Size
	if size <= 0 {
		size = 48
	}

	align := strings.ToLower(p.Align)
	if align != "left" && align != "right" {
		align = "center"
	}

	textColor, err := parseColor(p.Color)
	if err != nil {
		textColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}

	f := fontRegular
	if strings.EqualFold(p.FontFamily, "gomono") || strings.EqualFold(p.FontFamily, "mono") {
		f = fontMono
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("text.Render: failed to create face: %w", err)
	}
	defer face.Close()

	lines := strings.Split(p.Content, "\n")
	metrics := face.Metrics()
	ascent := float64(metrics.Ascent.Round())
	descent := float64(metrics.Descent.Round())
	lineHeight := float64(metrics.Height.Round())
	if lineHeight <= 0 {
		lineHeight = ascent + descent
	}
	if lineHeight <= 0 {
		lineHeight = size * 1.25
	}

	totalTextHeight := float64(len(lines)-1)*lineHeight + (ascent + descent)
	firstBaselineY := (p.Y * float64(h)) - (totalTextHeight / 2.0) + ascent

	for i, line := range lines {
		lineY := firstBaselineY + float64(i)*lineHeight
		lineWidth := font.MeasureString(face, line).Round()

		var lineX float64
		switch align {
		case "left":
			lineX = p.X * float64(w)
		case "right":
			lineX = (p.X * float64(w)) - float64(lineWidth)
		default: // "center"
			lineX = (p.X * float64(w)) - (float64(lineWidth) / 2.0)
		}

		dot := fixed.Point26_6{
			X: fixed.I(int(math.Round(lineX))),
			Y: fixed.I(int(math.Round(lineY))),
		}

		d := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(textColor),
			Face: face,
			Dot:  dot,
		}
		d.DrawString(line)
	}

	if p.Opacity > 0 && p.Opacity < 1.0 {
		alphaMult := p.Opacity
		for i := 3; i < len(dst.Pix); i += 4 {
			if dst.Pix[i] > 0 {
				dst.Pix[i] = uint8(math.Round(float64(dst.Pix[i]) * alphaMult))
			}
		}
	} else if p.Opacity < 0 {
		for i := 3; i < len(dst.Pix); i += 4 {
			dst.Pix[i] = 0
		}
	}

	return dst, nil
}

// parseColor parses hex color strings into color.RGBA.
// Supports #RGB, #RGBA, #RRGGBB, #RRGGBBAA with or without '#'.
func parseColor(s string) (color.RGBA, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		s = s[1:]
	}
	switch len(s) {
	case 3: // RGB -> RRGGBB
		r, err := strconv.ParseUint(string([]byte{s[0], s[0]}), 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		g, err := strconv.ParseUint(string([]byte{s[1], s[1]}), 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		b, err := strconv.ParseUint(string([]byte{s[2], s[2]}), 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}, nil
	case 4: // RGBA -> RRGGBBAA
		r, err := strconv.ParseUint(string([]byte{s[0], s[0]}), 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		g, err := strconv.ParseUint(string([]byte{s[1], s[1]}), 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		b, err := strconv.ParseUint(string([]byte{s[2], s[2]}), 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		a, err := strconv.ParseUint(string([]byte{s[3], s[3]}), 16, 8)
		if err != nil {
			return color.RGBA{}, err
		}
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}, nil
	case 6: // RRGGBB
		val, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return color.RGBA{}, err
		}
		return color.RGBA{
			R: uint8(val >> 16),
			G: uint8(val >> 8),
			B: uint8(val),
			A: 255,
		}, nil
	case 8: // RRGGBBAA
		val, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return color.RGBA{}, err
		}
		return color.RGBA{
			R: uint8(val >> 24),
			G: uint8(val >> 16),
			B: uint8(val >> 8),
			A: uint8(val),
		}, nil
	default:
		return color.RGBA{}, fmt.Errorf("invalid color format %q", s)
	}
}
