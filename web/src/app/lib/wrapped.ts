// Factories for the three hand-built imperative panels, closed over the shared
// `ui` state. Mounted through the `panel` action; migrating any of these to a
// native Svelte component later means deleting one factory here plus its source.

import { renderPalettePanel } from "../../palette-panel";
import { getBackend } from "../../backend";
import { ui } from "../store.svelte";

export function makePalettePanel() {
  const p = renderPalettePanel({
    extractPalette: async (img, opt) => (await getBackend(ui.settings.backend)).extractPalette(img, opt),
    genPalette: async (opt) => (await getBackend(ui.settings.backend)).genPalette(opt),
    source: () => ui.preview ?? ui.original,
  });
  (p.element as HTMLDetailsElement).open = true;
  return p;
}
