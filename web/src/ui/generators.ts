// Declarative descriptors for the algorithmic generators (the Go
// internal/generators registry). Same shape and philosophy as
// ./controls.ts: adding a generator is three touch points — a Go package
// under internal/, one generators.Register(...) in its init(), and one
// entry here — with no bespoke DOM code.
//
// Reuses the Control type from ./controls.ts. A generator produces a whole
// image from parameters alone (no input image), so it also carries a
// default output size.

import type { Control } from "./controls";

export interface GeneratorUI {
  /** Registry name in Go (internal/generators). */
  name: string;
  /** Human label for the generator picker. */
  title: string;
  /** Canvas size to use when this generator is first selected. */
  defaultSize: { width: number; height: number };
  controls: Control[];
}

export const generators: GeneratorUI[] = [
  // Phase 3 generators append their descriptors here:
  //   attractor, harmonograph, truchet, contours, flowfield, lsystem, flame
];
