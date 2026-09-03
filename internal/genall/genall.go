// Package genall blank-imports every algorithmic generator package so that
// importing this one alone links all of them (and runs their registering
// init functions). cmd/wasm imports it; each new generator package adds one
// blank-import line here.
package genall

import (
	_ "bitbrush/internal/attractor"
	_ "bitbrush/internal/chladni"
	_ "bitbrush/internal/contours"
	_ "bitbrush/internal/flame"
	_ "bitbrush/internal/flowfield"
	_ "bitbrush/internal/harmonograph"
	_ "bitbrush/internal/lsystem"
	_ "bitbrush/internal/reactiondiffusion"
	_ "bitbrush/internal/truchet"
)
