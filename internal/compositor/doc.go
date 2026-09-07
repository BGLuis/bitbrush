// Package compositor blends an ordered stack of layers into one image.
//
// It is an orchestrator, not a registered effect: it consumes
// internal/filters (each layer runs its own ordered filter chain),
// internal/generators, internal/gradient and internal/noisefield (layer
// sources), then applies a separable blend mode plus Porter-Duff
// source-over compositing per layer — the same way internal/anim
// orchestrates repeated filter application for GIF export.
//
// Nothing here imports syscall/js; everything is plain Go under go test.
// Evaluate is deterministic: the same (Spec, base, extras) always produces
// byte-identical output. Blend maths follow the W3C Compositing and
// Blending Level 1 spec, on gamma-encoded sRGB values, so results match CSS
// mix-blend-mode and Photoshop/Figma.
package compositor
