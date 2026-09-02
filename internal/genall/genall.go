// Package genall blank-imports every algorithmic generator package so that
// importing this one alone links all of them (and runs their registering
// init functions). cmd/wasm imports it; each new generator package adds one
// blank-import line here.
package genall

import (
	_ "bitbrush/internal/attractor"
	_ "bitbrush/internal/harmonograph"
	_ "bitbrush/internal/truchet"
)
