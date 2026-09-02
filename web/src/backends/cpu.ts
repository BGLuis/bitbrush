// Main-thread CPU backend: the Go/WASM core running directly on the page.
// Always available, accelerates every effect, and blocks the UI thread
// while it works — it's the guaranteed fallback and the reference path.
// The Worker backend is preferred whenever it loads (see ./worker.ts).

import {
  initWasm,
  applyFilter as wasmApplyFilter,
  asciiText as wasmAsciiText,
  renderGIF as wasmRenderGIF,
  renderGradient as wasmRenderGradient,
  gradientCSS as wasmGradientCSS,
  renderNoiseField as wasmRenderNoiseField,
  renderGenerator as wasmRenderGenerator,
  extractPalette as wasmExtractPalette,
  genPalette as wasmGenPalette,
  type FilterParams,
  type GifKeyframe,
  type GifOptions,
  type GradientParams,
  type GradientCSSOptions,
  type NoiseFieldParams,
  type GeneratorParams,
  type PaletteExtractOptions,
  type PaletteHarmonyOptions,
} from "../wasm";
import { registerBackend, type FilterBackend } from "../backend";

class CpuBackend implements FilterBackend {
  readonly kind = "cpu" as const;

  init(): Promise<void> {
    return initWasm();
  }

  async applyFilter(name: string, img: ImageData, params: FilterParams): Promise<ImageData> {
    return wasmApplyFilter(name, img, params);
  }

  async asciiText(img: ImageData, params: FilterParams): Promise<string> {
    return wasmAsciiText(img, params);
  }

  async renderGIF(
    name: string,
    img: ImageData,
    keyframes: GifKeyframe[],
    options: GifOptions,
  ): Promise<Uint8Array> {
    return wasmRenderGIF(name, img, keyframes, options);
  }

  async renderGradient(params: GradientParams): Promise<ImageData> {
    return wasmRenderGradient(params);
  }

  async gradientCSS(params: GradientParams, options: GradientCSSOptions): Promise<string> {
    return wasmGradientCSS(params, options);
  }

  async renderNoiseField(params: NoiseFieldParams, w: number, h: number): Promise<ImageData> {
    return wasmRenderNoiseField(params, w, h);
  }

  async renderGenerator(
    name: string,
    params: GeneratorParams,
    w: number,
    h: number,
  ): Promise<ImageData> {
    return wasmRenderGenerator(name, params, w, h);
  }

  async extractPalette(img: ImageData, options: PaletteExtractOptions): Promise<string[]> {
    return wasmExtractPalette(img, options);
  }

  async genPalette(options: PaletteHarmonyOptions): Promise<string[]> {
    return wasmGenPalette(options);
  }

  accelerates(): boolean {
    return true;
  }
}

registerBackend("cpu", () => new CpuBackend());
