// Factories for the three hand-built imperative panels, closed over the shared
// `ui` state. Mounted through the `panel` action; migrating any of these to a
// native Svelte component later means deleting one factory here plus its source.

import { renderGifPanel } from "../../gif";
import { renderGeneratorPanel } from "../../generators-panel";
import { renderPalettePanel } from "../../palette-panel";
import { getBackend } from "../../backend";
import { putImageData } from "../../canvas";
import { ui, refs } from "../store.svelte";

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

export function makeGeneratorPanel() {
  const p = renderGeneratorPanel({
    renderGradient: async (params) => (await getBackend(ui.settings.backend)).renderGradient(params),
    gradientCSS: async (params, opt) => (await getBackend(ui.settings.backend)).gradientCSS(params, opt),
    renderNoiseField: async (params, w, h) =>
      (await getBackend(ui.settings.backend)).renderNoiseField(params, w, h),
    show: (img) => {
      if (refs.canvas) putImageData(refs.canvas, img);
    },
  });
  (p.element as HTMLDetailsElement).open = true;
  return p;
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
