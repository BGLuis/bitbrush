package palette

import "image"

// Method selects the palette-extraction algorithm.
type Method int

const (
	// MethodMedianCut recursively splits colour boxes (fast, stable).
	MethodMedianCut Method = iota
	// MethodKMeans clusters sampled pixels (slower, often tighter fit).
	MethodKMeans
)

// ExtractOptions configures Extract. The zero value extracts a
// median-cut palette with no sorting, ignoring only fully transparent
// pixels.
type ExtractOptions struct {
	// Count is the requested number of palette entries. The result may be
	// shorter when the image holds fewer distinct colours.
	Count int
	// Method chooses the extraction algorithm.
	Method Method
	// Space is the colour space passed on to k-means (unused by median-cut).
	Space Space
	// Sort orders the returned palette.
	Sort SortKey
	// AlphaThreshold drops pixels whose alpha is below it. 0 keeps every
	// pixel that is not fully transparent, matching the default sampler.
	AlphaThreshold uint8
	// Seed seeds k-means++ when Method is MethodKMeans.
	Seed int64
}

// Extract pulls a palette from img: it samples pixels once (honouring
// AlphaThreshold), dispatches to the chosen Method, then applies the chosen
// Sort. It is the single entry point the app should call.
func Extract(img *image.RGBA, opt ExtractOptions) []RGB {
	pixels := samplePixelsAlpha(img, opt.AlphaThreshold)

	var pal []RGB
	switch opt.Method {
	case MethodKMeans:
		pal = kmeansPixels(pixels, opt.Count, KMeansOptions{
			Space: opt.Space,
			Seed:  opt.Seed,
		})
	default:
		pal = medianCutPixels(pixels, opt.Count)
	}

	return SortPalette(pal, opt.Sort, img)
}
