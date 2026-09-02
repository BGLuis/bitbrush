// Package genall blank-imports every algorithmic generator package so that
// importing this one alone links all of them (and runs their registering
// init functions). cmd/wasm imports it; each new generator package adds one
// blank-import line here.
package genall

// Generator packages are added here as they land:
//   _ "bitbrush/internal/attractor"
//   _ "bitbrush/internal/harmonograph"
//   ...
