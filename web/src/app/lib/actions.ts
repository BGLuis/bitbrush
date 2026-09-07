// Shared top-level actions. Both the Topbar buttons and the keyboard shortcuts
// drive these; the Topbar passes an `onStatus` callback to animate its own
// button labels, the shortcuts pass one that raises a toast instead.

import { effects } from "../../ui/controls";
import { getBackend } from "../../backend";
import { ui, refs, selectEffect } from "../store.svelte";
import { generatorStore } from "../generator/generator-store.svelte";
import { openFilePicker } from "./image";
import { showToast } from "./toast.svelte";

type StatusFn = (label: string | null) => void;

export function openImage(): void {
  openFilePicker();
}

/** Cycle through the filter list. Switches into filter mode from anywhere. */
export function cycleEffect(dir: 1 | -1): void {
  const i = Math.max(0, effects.findIndex((e) => e.name === ui.effectName));
  const next = effects[(i + dir + effects.length) % effects.length];
  selectEffect(next.name);
  showToast(next.title);
}

export function copyShareLink(onStatus?: StatusFn): void {
  void navigator.clipboard?.writeText(location.href);
  onStatus?.("Link copiado!");
  setTimeout(() => onStatus?.(null), 1400);
}

function downloadCanvas(canvas: HTMLCanvasElement, filename: string): void {
  canvas.toBlob((blob) => {
    if (!blob) return;
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 15_000);
  }, "image/png");
}

export async function exportPNG(onStatus?: StatusFn): Promise<void> {
  if (ui.mode === "generator" || ui.mode === "pattern") {
    void generatorStore.exportHighResPNG();
    return;
  }

  const src = ui.original ?? ui.preview;
  if (!src) {
    if (refs.canvas) downloadCanvas(refs.canvas, `bitbrush-${ui.mode}.png`);
    return;
  }

  onStatus?.("Processando...");
  try {
    const backend = await getBackend(ui.settings.backend);
    const out = await backend.applyFilter(ui.effectName, src, ui.params);
    const tempCanvas = document.createElement("canvas");
    tempCanvas.width = out.width;
    tempCanvas.height = out.height;
    const ctx = tempCanvas.getContext("2d")!;
    ctx.putImageData(out, 0, 0);
    downloadCanvas(tempCanvas, `bitbrush-${ui.effectName}-${out.width}x${out.height}.png`);
    onStatus?.("✓ Salvo!");
    setTimeout(() => onStatus?.(null), 1500);
  } catch (err) {
    console.error(err);
    onStatus?.("Erro ao exportar");
    setTimeout(() => onStatus?.(null), 2000);
  }
}
