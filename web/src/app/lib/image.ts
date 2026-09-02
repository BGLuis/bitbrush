// Image loading + preview downscaling, wired to the shared store. A hidden
// <input type=file> is created lazily so any control (topbar button, empty-
// state button) can call openFilePicker().

import { loadImageFile, drawImageToCanvas } from "../../canvas";
import { downscaleImageData } from "../../preview";
import { ui, refs } from "../store.svelte";

let input: HTMLInputElement | null = null;

export function openFilePicker(): void {
  if (!input) {
    input = document.createElement("input");
    input.type = "file";
    input.accept = "image/*";
    input.hidden = true;
    input.addEventListener("change", () => {
      const file = input!.files?.[0];
      if (file) void loadFile(file);
      input!.value = ""; // let the same file be picked again
    });
    document.body.appendChild(input);
  }
  input.click();
}

export async function loadFile(file: File): Promise<void> {
  const img = await loadImageFile(file);
  const canvas = refs.canvas;
  if (!canvas) return;
  ui.original = drawImageToCanvas(canvas, img);
  recomputePreview();
}

/** Re-derive the capped working copy — call after the preview-resolution
 *  setting changes. Exports always use `ui.original`, never this. */
export function recomputePreview(): void {
  ui.preview = ui.original ? downscaleImageData(ui.original, ui.settings.previewMaxDim) : null;
}
