// Package noisefield is a deterministic, pure-Go port of the WebGL
// fragment shader behind gurade.netlify.app's "Gradient Studio": a
// generative gradient renderer with noise-warped organic fields, per-style
// colour grades, and surface textures.
//
// It has no GPU, no cgo and no syscall/js dependency; it is the algorithm
// core that the WASM adapter calls and that a GLSL shader mirrors on the
// GPU backend. Render is a pure function of its Params: the same Params
// (Seed and Time included) always produce a byte-identical *image.RGBA.
//
// # Conventions
//
// Pixel/GL-origin: the renderer works in a top-left pixel origin. For pixel
// (x, y) it uses uv = ((x+0.5)/w, (y+0.5)/h) and the shader's p =
// (uv.x, 1-uv.y). The Grain/Frosted/Wrinkle/Paper textures read GL's
// gl_FragCoord, which is bottom-left; that is reconstructed explicitly as
// fragCoord = (x+0.5, h-0.5-y) so the hash and row maths match the shader
// exactly.
//
// Seed reduction: the shader wants a float u_seed. It is derived as
//
//	u_seed = float64(((Seed % 100000) + 100000) % 100000) + 7.3
//
// The non-negative modular reduction keeps huge or negative Seed values in
// a range where snoise stays well-conditioned; the +7.3 baseline offsets
// Seed == 0 from the noise lattice origin so it still yields a varied field.
package noisefield

import (
	"image"
	"math"
)

// Field selects the gradient geometry. The order is the shader's u_type
// and must not be reordered.
type Field int

const (
	FieldLinear    Field = iota // 0
	FieldRadial                 // 1
	FieldConic                  // 2
	FieldReflected              // 3
	FieldDiamond                // 4
	FieldMesh                   // 5
	FieldFreeform               // 6
	FieldFlow                   // 7
)

// Style selects the per-field colour grade. The order is the shader's
// u_genre.
type Style int

const (
	StyleMetallic    Style = iota // 0
	StyleChrome                   // 1
	StyleIridescent               // 2
	StyleHolographic              // 3
	StyleNeon                     // 4
	StylePastel                   // 5
	StyleDuotone                  // 6 -- passthrough: the shader has no u_genre==6 branch
	StyleRainbow                  // 7
)

// Texture selects the surface texture pass. The order is the shader's
// u_texture.
type Texture int

const (
	TextureSmooth  Texture = iota // 0
	TextureGrain                  // 1
	TextureFrosted                // 2
	TextureWave                   // 3
	TextureWrinkle                // 4
	TexturePaper                  // 5
)

// RGB is an 8-bit opaque colour.
type RGB struct{ R, G, B uint8 }

// Spot is a colour placed at (X, Y) in [0,1] canvas coordinates, used by
// the organic fields (Mesh / Freeform / Flow). X and Y are in the shader's
// p-space, i.e. Y increases upward.
type Spot struct {
	Color RGB
	X, Y  float64
}

// Params fully describes a frame. Every field is URL-encodable so a result
// is shareable and reproducible.
type Params struct {
	Field      Field
	Style      Style
	Texture    Texture
	Stops      []RGB  // ramp for the geometric fields (>=1; a single stop => flat)
	Spots      []Spot // colour spots for the organic fields (>=1); capped at 8
	AngleDeg   float64
	Scale      float64 // 0..100 UI slider -> noise frequency
	Distortion float64 // 0..100 UI slider -> warp amount
	Seed       int64
	Time       float64 // animation clock in seconds; 0 == static frame
}

// Render draws the field into a new opaque RGBA of size w x h with origin
// (0, 0). w or h <= 0 yields an empty image.
func Render(p Params, w, h int) *image.RGBA {
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	lin := linearizeStops(p.Stops)
	spots := prepSpots(&p)

	uAngle := p.AngleDeg * math.Pi / 180.0
	dirX := math.Sin(uAngle)
	dirY := -math.Cos(uAngle)

	// u_freq = 3.2 + (0.7 - 3.2) * (Scale/100)
	uFreq := 3.2 + (0.7-3.2)*(p.Scale/100.0)

	// u_warp = (Distortion/100) * per-field factor
	warpFactor := 0.6
	switch p.Field {
	case FieldFlow:
		warpFactor = 1.1
	case FieldMesh:
		warpFactor = 0.35
	}
	uWarp := (p.Distortion / 100.0) * warpFactor

	uSeed := seedReduce(p.Seed)
	t := p.Time

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := renderPixel(&p, lin, spots, x, y, w, h, dirX, dirY, uFreq, uWarp, uSeed, uAngle, t)
			o := img.PixOffset(x, y)
			img.Pix[o+0] = uint8(c.x*255.0 + 0.5)
			img.Pix[o+1] = uint8(c.y*255.0 + 0.5)
			img.Pix[o+2] = uint8(c.z*255.0 + 0.5)
			img.Pix[o+3] = 255
		}
	}
	return img
}

// seedReduce maps an arbitrary Seed to the shader's float u_seed. See the
// package doc for the rationale.
func seedReduce(seed int64) float64 {
	m := seed % 100000
	if m < 0 {
		m += 100000
	}
	return float64(m) + 7.3
}

// prepSpots returns the spot list the organic fields should use: nil for a
// geometric field, a single centre white spot when none were supplied, and
// at most the first 8 otherwise.
func prepSpots(p *Params) []Spot {
	if p.Field <= FieldDiamond {
		return nil
	}
	sp := p.Spots
	if len(sp) == 0 {
		return []Spot{{Color: RGB{255, 255, 255}, X: 0.5, Y: 0.5}}
	}
	if len(sp) > 8 {
		sp = sp[:8]
	}
	return sp
}

// renderPixel evaluates the fragment shader for one pixel and returns the
// final colour clamped to [0,1].
func renderPixel(p *Params, lin []linRGB, spots []Spot, x, y, w, h int,
	dirX, dirY, uFreq, uWarp, uSeed, uAngle, t float64) vec3 {

	fw, fh := float64(w), float64(h)
	minDim := math.Min(fw, fh)

	uvx := (float64(x) + 0.5) / fw
	uvy := (float64(y) + 0.5) / fh
	pX := uvx
	pY := 1.0 - uvy

	// Texture pre-warp on p.x, applied before the field is evaluated.
	switch p.Texture {
	case TextureWave:
		amp := math.Max(6.0, minDim*0.016) / fw
		f := 2.0 * math.Pi / math.Max(180.0/fh, 0.34)
		pX += math.Sin(pY*f)*amp + math.Sin(pY*f*0.46)*amp*0.45
	case TextureWrinkle:
		amp := math.Max(4.0, minDim*0.011) / fw
		f := 2.0 * math.Pi / math.Max(140.0/fh, 0.28)
		pX += math.Sin(pY*f)*amp + math.Sin(pY*f*1.9)*amp*0.35
	}

	pxX := (pX - 0.5) * fw
	pxY := (pY - 0.5) * fh

	var col vec3

	if p.Field <= FieldDiamond {
		// Geometric fields: a scalar ramp coordinate through palette().
		var tt float64
		switch p.Field {
		case FieldLinear, FieldReflected:
			// The reference shader treats Reflected identically to Linear
			// here; kept faithful rather than "fixed".
			ln := math.Abs(dirX)*fw*0.5 + math.Abs(dirY)*fh*0.5
			tt = (pxX*dirX+pxY*dirY)/ln*0.5 + 0.5
		case FieldRadial:
			tt = math.Hypot(pxX, pxY) / (math.Hypot(fw, fh) * 0.5)
		case FieldConic:
			a := math.Atan2(pxX, -pxY)
			tt = fract((a - uAngle) / (2.0 * math.Pi))
		case FieldDiamond:
			tt = (math.Abs(pxX)/(fw*0.5) + math.Abs(pxY)/(fh*0.5)) * 0.5
		}
		col = paletteSRGB(lin, clampf(tt, 0.0, 1.0))
	} else {
		// Organic fields: noise-warped inverse-distance blend of spots.
		aspect := fw / fh
		paX := pX * aspect
		paY := pY
		qX, qY := paX, paY

		npX := (qX+dirX*t*0.03)*uFreq*0.75 + uSeed
		npY := (qY+dirY*t*0.03)*uFreq*0.75 + uSeed

		w1x := fbm(npX+t*0.05, npY+t*0.05)
		w1y := fbm(npX+5.2-t*0.04, npY+1.3-t*0.04)
		qX += (w1x - 0.5) * uWarp
		qY += (w1y - 0.5) * uWarp

		if p.Field == FieldFlow {
			w2x := fbm(qX*uFreq*1.15+3.1+t*0.03, qY*uFreq*1.15+3.1+t*0.03)
			w2y := fbm(qX*uFreq*1.15+7.7-t*0.02, qY*uFreq*1.15+7.7-t*0.02)
			qX += (w2x - 0.5) * uWarp * 0.55
			qY += (w2y - 0.5) * uWarp * 0.55
		}

		pw, eps := 2.0, 0.002
		switch p.Field {
		case FieldMesh:
			pw, eps = 2.4, 0.006
		case FieldFreeform:
			pw, eps = 3.6, 0.002
		case FieldFlow:
			pw, eps = 2.0, 0.012
		}

		var accX, accY, accZ, ws float64
		for i := 0; i < len(spots); i++ {
			s := spots[i]
			sx := s.X * aspect
			sy := s.Y
			d := math.Hypot(qX-sx, qY-sy)
			wgt := 1.0 / (math.Pow(d, pw) + eps)
			accX += float64(s.Color.R) / 255.0 * wgt
			accY += float64(s.Color.G) / 255.0 * wgt
			accZ += float64(s.Color.B) / 255.0 * wgt
			ws += wgt
		}
		inv := 1.0 / math.Max(ws, 1e-6)
		col = vec3{accX * inv, accY * inv, accZ * inv}

		n := fbm(paX*uFreq*0.8+uSeed*0.37+t*0.02, paY*uFreq*0.8+uSeed*0.37+t*0.02)
		s := (paX-aspect*0.5)*dirX + (paY-0.5)*dirY

		switch p.Style {
		case StyleMetallic:
			band := math.Sin((s*1.6 + n*0.6) * math.Pi)
			col = col.mulf(0.88 + 0.12*band)
			col = col.addf(math.Pow(math.Max(band, 0.0), 3.0) * 0.12)
		case StyleChrome:
			col = mixv(grey(lum(col)), col, 0.6)
			band := math.Sin((s*1.8 + n*0.8) * math.Pi)
			col = col.mulf(0.76 + 0.24*band)
			col = col.addf(math.Pow(math.Max(band, 0.0), 4.0) * 0.26)
		case StyleIridescent:
			col = hueShift(col, (n-0.5)*2.2)
		case StyleHolographic:
			rb := hsv2rgb(fract(s*1.4+n*0.7+t*0.02), 0.55, 1.0)
			col = mixv(col, rb, 0.4).addf(0.06)
		case StyleNeon:
			col = mixv(grey(lum(col)), col, 1.4)
			col = vec3{(col.x-0.5)*1.15 + 0.5, (col.y-0.5)*1.15 + 0.5, (col.z-0.5)*1.15 + 0.5}
			col = col.add(col.mulf(math.Pow(n, 3.0) * 0.25))
		case StylePastel:
			col = mixv(col, grey(1.0), 0.42)
		case StyleDuotone:
			// Passthrough: the shader has no u_genre==6 branch.
		case StyleRainbow:
			rb := hsv2rgb(fract(n*1.5+s*0.8), 0.75, 1.0)
			col = mixv(col, rb, 0.65)
		}
	}

	// Texture post-pass. fragCoord is GL's bottom-left gl_FragCoord.
	fcx := float64(x) + 0.5
	fcy := fh - 0.5 - float64(y)
	switch p.Texture {
	case TextureGrain:
		col = col.addf((hash(fcx, fcy) - 0.5) * 0.086)
	case TextureFrosted:
		col = mixv(col, grey(1.0), 0.22).addf((hash(fcx, fcy) - 0.5) * 0.039)
	case TextureWave:
		col = mixv(col, grey(1.0), 0.10)
	case TextureWrinkle:
		st := math.Max(54.0, fw/14.0)
		u := fcx + (fh-fcy)*0.85 + math.Sin(fcy*0.02)*7.0
		m := modx(u, st)
		lt := 1.0 - smoothstep(0.0, 1.4, math.Abs(m-0.7))
		dk := 1.0 - smoothstep(0.0, 1.2, math.Abs(m-3.5))
		col = col.addf(lt*0.06 - dk*0.045)
	case TexturePaper:
		col = col.addf((hash(fcx, fcy) - 0.5) * 0.047)
		row := modx(math.Floor(fcy), 4.0)
		if row < 0.5 {
			col = col.addf(0.016)
		} else if math.Abs(row-2.0) < 0.5 {
			col = col.addf(-0.012)
		}
	}

	return vec3{
		clampf(col.x, 0.0, 1.0),
		clampf(col.y, 0.0, 1.0),
		clampf(col.z, 0.0, 1.0),
	}
}
