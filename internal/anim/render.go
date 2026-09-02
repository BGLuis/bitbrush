package anim

import (
	"errors"
	"image"
	"math"

	xdraw "golang.org/x/image/draw"

	"bitbrush/internal/filters"
)

// ErrNoKeyframes is returned when a render is asked for with no keyframes.
var ErrNoKeyframes = errors.New("anim: at least one keyframe is required")

// RenderFrames renders every interpolated frame by calling filters.Apply
// with the named effect. src is downscaled once up front when
// Options.MaxDimension applies; each frame is then produced from that
// single scaled source so the work of scaling is not repeated.
func RenderFrames(src *image.RGBA, effectName string, kfs []Keyframe, opt Options) ([]*image.RGBA, error) {
	if len(kfs) == 0 {
		return nil, ErrNoKeyframes
	}

	base := downscale(src, opt.MaxDimension)
	n := opt.frameCount()

	frames := make([]*image.RGBA, 0, n)
	for i := 0; i < n; i++ {
		params := forFilter(Interpolate(kfs, i, n, opt.PingPong))
		frame, err := filters.Apply(effectName, base, params)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// Render is RenderFrames followed by EncodeGIF.
func Render(src *image.RGBA, effectName string, kfs []Keyframe, opt Options) ([]byte, error) {
	frames, err := RenderFrames(src, effectName, kfs, opt)
	if err != nil {
		return nil, err
	}
	return EncodeGIF(frames, opt)
}

// forFilter adapts interpolated params to the filter ABI, which reads
// every number as a float64 (params normally arrive as decoded JSON).
// Interpolate may hand back int-kinded values to keep them whole; convert
// those so the effect actually sees the animated value.
func forFilter(p filters.Params) filters.Params {
	out := make(filters.Params, len(p))
	for k, v := range p {
		if f, ok := toFloat(v); ok {
			out[k] = f
			continue
		}
		out[k] = v
	}
	return out
}

// downscale shrinks src so its longest side is at most maxDim, preserving
// aspect ratio. It never enlarges: maxDim <= 0, or an image already within
// the limit, is returned unchanged.
func downscale(src *image.RGBA, maxDim int) *image.RGBA {
	if maxDim <= 0 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	longest := w
	if h > longest {
		longest = h
	}
	if longest <= maxDim || longest == 0 {
		return src
	}

	scale := float64(maxDim) / float64(longest)
	nw := int(math.Round(float64(w) * scale))
	nh := int(math.Round(float64(h) * scale))
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
	return dst
}
