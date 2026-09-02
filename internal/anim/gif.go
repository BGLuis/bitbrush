package anim

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"

	"bitbrush/internal/palette"
)

// ErrNoFrames is returned when EncodeGIF is handed an empty frame slice.
var ErrNoFrames = errors.New("anim: no frames to encode")

const (
	// gifColors is the GIF colour-table ceiling.
	gifColors = 256
	// maxPaletteSamples caps how many pixels, summed across every frame,
	// feed the shared median-cut palette.
	maxPaletteSamples = 50000
)

// EncodeGIF builds ONE shared colour palette for the whole animation
// (median-cut, via internal/palette, over a stride-sample of all frames),
// maps every frame onto it with a single reused OKLab mapper, and writes
// an animated GIF. If any frame has a fully transparent pixel, one palette
// slot is reserved for transparency. All frames get the same delay,
// derived from Options.FPS; LoopCount follows Options.LoopForever.
func EncodeGIF(frames []*image.RGBA, opt Options) ([]byte, error) {
	if len(frames) == 0 {
		return nil, ErrNoFrames
	}

	samples, hasTransparent := samplePixels(frames)

	want := gifColors
	if hasTransparent {
		want = gifColors - 1
	}
	pal := palette.MedianCut(scratchImage(samples), want)
	if len(pal) == 0 {
		pal = []palette.RGB{{R: 0, G: 0, B: 0}}
	}

	cp := make(color.Palette, 0, len(pal)+1)
	for _, c := range pal {
		cp = append(cp, color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff})
	}
	transIndex := -1
	if hasTransparent {
		transIndex = len(cp)
		cp = append(cp, color.RGBA{})
	}

	mapper := palette.NewMapper(pal, palette.SpaceOKLab)

	bounds := frames[0].Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	delay := opt.delayCentiseconds()
	out := &gif.GIF{
		Image:     make([]*image.Paletted, len(frames)),
		Delay:     make([]int, len(frames)),
		LoopCount: opt.loopCount(),
		Config: image.Config{
			ColorModel: cp,
			Width:      w,
			Height:     h,
		},
	}

	for i, frame := range frames {
		out.Image[i] = paletteFrame(frame, cp, mapper, transIndex)
		out.Delay[i] = delay
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// paletteFrame maps one RGBA frame onto cp. Fully transparent pixels take
// the reserved transparent slot when there is one; every other pixel goes
// to its nearest palette entry via the shared mapper.
func paletteFrame(src *image.RGBA, cp color.Palette, mapper *palette.Mapper, transIndex int) *image.Paletted {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewPaletted(image.Rect(0, 0, w, h), cp)

	for y := 0; y < h; y++ {
		srcRow := src.PixOffset(b.Min.X, b.Min.Y+y)
		dstRow := dst.PixOffset(0, y)
		for x := 0; x < w; x++ {
			s := srcRow + x*4
			if transIndex >= 0 && src.Pix[s+3] == 0 {
				dst.Pix[dstRow+x] = uint8(transIndex)
				continue
			}
			idx := mapper.Index(palette.RGB{R: src.Pix[s], G: src.Pix[s+1], B: src.Pix[s+2]})
			dst.Pix[dstRow+x] = uint8(idx)
		}
	}
	return dst
}

// samplePixels stride-samples opaque pixels across every frame, down to
// maxPaletteSamples total, and reports whether any fully transparent pixel
// was seen anywhere.
func samplePixels(frames []*image.RGBA) (samples []palette.RGB, hasTransparent bool) {
	total := 0
	for _, f := range frames {
		b := f.Bounds()
		total += b.Dx() * b.Dy()
	}
	if total == 0 {
		return nil, false
	}
	step := 1
	if total > maxPaletteSamples {
		step = (total + maxPaletteSamples - 1) / maxPaletteSamples
	}

	samples = make([]palette.RGB, 0, min(total, maxPaletteSamples)+1)
	idx := 0
	for _, f := range frames {
		b := f.Bounds()
		w, h := b.Dx(), b.Dy()
		for y := 0; y < h; y++ {
			row := f.PixOffset(b.Min.X, b.Min.Y+y)
			for x := 0; x < w; x++ {
				i := row + x*4
				if f.Pix[i+3] == 0 {
					hasTransparent = true
				} else if idx%step == 0 {
					samples = append(samples, palette.RGB{R: f.Pix[i], G: f.Pix[i+1], B: f.Pix[i+2]})
				}
				idx++
			}
		}
	}
	return samples, hasTransparent
}

// scratchImage packs sampled colours into a 1-row RGBA image so it can be
// handed to palette.MedianCut.
func scratchImage(samples []palette.RGB) *image.RGBA {
	if len(samples) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	img := image.NewRGBA(image.Rect(0, 0, len(samples), 1))
	for i, c := range samples {
		o := i * 4
		img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = c.R, c.G, c.B, 0xff
	}
	return img
}
