// Image loading + preview downscaling, wired to the shared store. A hidden
// <input type=file> is created lazily so any control (topbar button, empty-
// state button) can call openFilePicker().

import { loadImageFile, drawImageToCanvas } from "../../canvas";
import { downscaleImageData } from "../../preview";
import { ui, refs } from "../store.svelte";
import { showToast } from "./toast.svelte";
import { zoomFit } from "./view.svelte";

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
  if (!file.type.startsWith("image/")) {
    showToast(`"${file.name}" não é uma imagem`, "error");
    return;
  }
  // Generator / pattern modes drive the canvas themselves — hand control back
  // to the filter path so the dropped image actually shows.
  if (ui.mode !== "filter" && ui.mode !== "palette") ui.mode = "filter";
  try {
    const img = await loadImageFile(file);
    const canvas = refs.canvas;
    if (!canvas) return;
    ui.original = drawImageToCanvas(canvas, img);
    ui.preview = downscaleImageElement(img, ui.settings.previewMaxDim);
    zoomFit();
    showToast(`${img.naturalWidth}×${img.naturalHeight} · ${file.name}`);
  } catch (err) {
    console.error(err);
    showToast("Não foi possível decodificar a imagem", "error");
  }
}

function downscaleImageElement(img: HTMLImageElement, maxDim: number): ImageData {
  const longest = Math.max(img.naturalWidth, img.naturalHeight);
  if (!Number.isFinite(maxDim) || maxDim <= 0 || longest <= maxDim) {
    const c = document.createElement("canvas");
    c.width = img.naturalWidth;
    c.height = img.naturalHeight;
    const ctx = c.getContext("2d", { willReadFrequently: true });
    if (!ctx) throw new Error("2d context unavailable");
    ctx.drawImage(img, 0, 0);
    return ctx.getImageData(0, 0, c.width, c.height);
  }
  const scale = maxDim / longest;
  const w = Math.max(1, Math.round(img.naturalWidth * scale));
  const h = Math.max(1, Math.round(img.naturalHeight * scale));
  const c = document.createElement("canvas");
  c.width = w;
  c.height = h;
  const ctx = c.getContext("2d", { willReadFrequently: true });
  if (!ctx) throw new Error("2d context unavailable");
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = "high";
  ctx.drawImage(img, 0, 0, w, h);
  return ctx.getImageData(0, 0, w, h);
}

/** Re-derive the capped working copy — call after the preview-resolution
 *  setting changes. Exports always use `ui.original`, never this. */
export function recomputePreview(): void {
  ui.preview = ui.original ? downscaleImageData(ui.original, ui.settings.previewMaxDim) : null;
}
