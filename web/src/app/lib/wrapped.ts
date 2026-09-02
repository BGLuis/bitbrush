// Factories for the three hand-built imperative panels, closed over the shared
// `ui` state. Mounted through the `panel` action; migrating any of these to a
// native Svelte component later means deleting one factory here plus its source.

import { renderGifPanel } from "../../gif";
import { renderPalettePanel } from "../../palette-panel";
import { getBackend } from "../../backend";
import { ui } from "../store.svelte";

export function makeGifPanel() {
  return renderGifPanel({
    currentEffect: () => ui.effectName,
    // params is a flat record of primitives — a shallow spread is plain enough.
    currentParams: () => ({ ...ui.params }),
    source: () => ui.original,
    render: async (name, img, kfs, opt) =>
      (await getBackend(ui.settings.backend)).renderGIF(name, img, kfs, opt),
  });
}

export function makePalettePanel() {
  const p = renderPalettePanel({
    extractPalette: async (img, opt) => (await getBackend(ui.settings.backend)).extractPalette(img, opt),
    genPalette: async (opt) => (await getBackend(ui.settings.backend)).genPalette(opt),
    source: () => ui.original,
  });
  (p.element as HTMLDetailsElement).open = true;
  return p;
}
