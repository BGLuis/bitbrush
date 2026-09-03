// Canvas <-> ImageData helpers. No effect logic here.

export function drawImageToCanvas(
  canvas: HTMLCanvasElement,
  source: HTMLImageElement,
): ImageData {
  canvas.width = source.naturalWidth;
  canvas.height = source.naturalHeight;
  const ctx = canvas.getContext("2d", { willReadFrequently: true });
  if (!ctx) throw new Error("2d context unavailable");
  ctx.drawImage(source, 0, 0);
  return ctx.getImageData(0, 0, canvas.width, canvas.height);
}

export function putImageData(canvas: HTMLCanvasElement, img: ImageData): void {
  if (canvas.width !== img.width) canvas.width = img.width;
  if (canvas.height !== img.height) canvas.height = img.height;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("2d context unavailable");
  ctx.putImageData(img, 0, 0);
}

export function blitCanvas(target: HTMLCanvasElement, source: HTMLCanvasElement): void {
  if (target.width !== source.width) target.width = source.width;
  if (target.height !== source.height) target.height = source.height;
  const ctx = target.getContext("2d");
  if (!ctx) throw new Error("2d context unavailable");
  ctx.drawImage(source, 0, 0);
}

export async function loadImageFile(file: File): Promise<HTMLImageElement> {
  const url = URL.createObjectURL(file);
  try {
    const img = new Image();
    img.src = url;
    await img.decode();
    return img;
  } finally {
    URL.revokeObjectURL(url);
  }
}
