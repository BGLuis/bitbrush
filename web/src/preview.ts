// Live-preview downscaling. Dragging a slider re-runs a filter on every
// frame; on a GPU-less device a 12-megapixel source makes that painful even
// with the work off the main thread. The preview runs on a copy capped to a
// chosen longest-side length; exports (ASCII text, and later GIF / "save")
// use the full-resolution original.

/**
 * Return `src` scaled so its longest side is at most `maxDim`, preserving
 * aspect ratio. `maxDim <= 0` or a non-finite value means "no cap" and
 * returns `src` unchanged. Never upscales.
 */
export function downscaleImageData(src: ImageData, maxDim: number): ImageData {
  const longest = Math.max(src.width, src.height);
  if (!Number.isFinite(maxDim) || maxDim <= 0 || longest <= maxDim) return src;

  const scale = maxDim / longest;
  const w = Math.max(1, Math.round(src.width * scale));
  const h = Math.max(1, Math.round(src.height * scale));

  const [, sctx] = scratch(src.width, src.height);
  sctx.putImageData(src, 0, 0);

  const [, dctx] = scratch(w, h);
  dctx.imageSmoothingEnabled = true;
  dctx.imageSmoothingQuality = "high";
  dctx.drawImage(sctx.canvas, 0, 0, w, h);
  return dctx.getImageData(0, 0, w, h);
}

function scratch(w: number, h: number): [HTMLCanvasElement, CanvasRenderingContext2D] {
  const c = document.createElement("canvas");
  c.width = w;
  c.height = h;
  const ctx = c.getContext("2d", { willReadFrequently: true });
  if (!ctx) throw new Error("2d context unavailable");
  return [c, ctx];
}
