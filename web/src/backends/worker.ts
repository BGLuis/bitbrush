// Web Worker CPU backend: the same Go/WASM core as ./cpu.ts, but on a
// worker thread so a slow filter (palette-mode dither ~100ms, a big source)
// never freezes scrolling or the sliders. This is the preferred backend;
// backend.ts falls back to the main-thread one if the worker can't start
// (no Worker constructor, WASM blocked in workers, strict CSP).

import { registerBackend, type FilterBackend } from "../backend";
import type {
  FilterParams,
  GifKeyframe,
  GifOptions,
  GradientParams,
  GradientCSSOptions,
  NoiseFieldParams,
  GeneratorParams,
  PaletteExtractOptions,
  PaletteHarmonyOptions,
} from "../wasm";

interface Reply {
  id: number;
  ok: boolean;
  error?: string;
  text?: string;
  colors?: string[];
  buf?: ArrayBuffer;
  width?: number;
  height?: number;
}

class WorkerBackend implements FilterBackend {
  readonly kind = "cpu-worker" as const;

  #worker: Worker | null = null;
  #seq = 1;
  #pending = new Map<number, { resolve: (r: Reply) => void; reject: (e: unknown) => void }>();

  init(): Promise<void> {
    if (typeof Worker === "undefined") {
      return Promise.reject(new Error("Web Workers unavailable"));
    }
    const worker = new Worker(new URL("../worker/filter-worker.ts", import.meta.url), {
      type: "classic",
    });
    worker.onmessage = (e: MessageEvent) => {
      const r = e.data as Reply;
      const waiter = this.#pending.get(r.id);
      if (!waiter) return;
      this.#pending.delete(r.id);
      r.ok ? waiter.resolve(r) : waiter.reject(new Error(r.error || "worker error"));
    };
    worker.onerror = (e) => {
      // Fail every in-flight call; new calls will reject on a null #worker.
      for (const [, w] of this.#pending) w.reject(new Error(e.message || "worker crashed"));
      this.#pending.clear();
      this.#worker = null;
    };
    this.#worker = worker;
    return this.#send({ op: "init" }).then(() => undefined);
  }

  async applyFilter(name: string, img: ImageData, params: FilterParams): Promise<ImageData> {
    const copy = img.data.slice(); // don't neuter the caller's ImageData on transfer
    const r = await this.#send(
      { op: "filter", name, buf: copy.buffer, width: img.width, height: img.height, params: JSON.stringify(params) },
      [copy.buffer],
    );
    return new ImageData(new Uint8ClampedArray(r.buf!), r.width!, r.height!);
  }

  async asciiText(img: ImageData, params: FilterParams): Promise<string> {
    const copy = img.data.slice();
    const r = await this.#send(
      { op: "ascii", buf: copy.buffer, width: img.width, height: img.height, params: JSON.stringify(params) },
      [copy.buffer],
    );
    return r.text ?? "";
  }

  async renderGIF(
    name: string,
    img: ImageData,
    keyframes: GifKeyframe[],
    options: GifOptions,
  ): Promise<Uint8Array> {
    const copy = img.data.slice();
    const r = await this.#send(
      {
        op: "gif",
        name,
        buf: copy.buffer,
        width: img.width,
        height: img.height,
        keyframes: JSON.stringify(keyframes),
        options: JSON.stringify(options),
      },
      [copy.buffer],
    );
    return new Uint8Array(r.buf!);
  }

  async renderGradient(params: GradientParams): Promise<ImageData> {
    const r = await this.#send({
      op: "gradient",
      params: JSON.stringify(params),
      width: params.width,
      height: params.height,
    });
    return new ImageData(new Uint8ClampedArray(r.buf!), r.width!, r.height!);
  }

  async gradientCSS(params: GradientParams, options: GradientCSSOptions): Promise<string> {
    const r = await this.#send({
      op: "gradientCSS",
      params: JSON.stringify(params),
      options: JSON.stringify(options),
    });
    return r.text ?? "";
  }

  async renderNoiseField(params: NoiseFieldParams, w: number, h: number): Promise<ImageData> {
    const r = await this.#send({
      op: "noisefield",
      params: JSON.stringify(params),
      width: w,
      height: h,
    });
    return new ImageData(new Uint8ClampedArray(r.buf!), r.width!, r.height!);
  }

  async renderGenerator(
    name: string,
    params: GeneratorParams,
    w: number,
    h: number,
  ): Promise<ImageData> {
    const r = await this.#send({
      op: "generator",
      name,
      params: JSON.stringify(params),
      width: w,
      height: h,
    });
    return new ImageData(new Uint8ClampedArray(r.buf!), r.width!, r.height!);
  }

  async extractPalette(img: ImageData, options: PaletteExtractOptions): Promise<string[]> {
    const copy = img.data.slice();
    const r = await this.#send(
      { op: "extractPalette", buf: copy.buffer, width: img.width, height: img.height, options: JSON.stringify(options) },
      [copy.buffer],
    );
    return r.colors ?? [];
  }

  async genPalette(options: PaletteHarmonyOptions): Promise<string[]> {
    const r = await this.#send({ op: "genPalette", options: JSON.stringify(options) });
    return r.colors ?? [];
  }

  accelerates(): boolean {
    return true;
  }

  #send(msg: Record<string, unknown>, transfer: Transferable[] = []): Promise<Reply> {
    const worker = this.#worker;
    if (!worker) return Promise.reject(new Error("worker not running"));
    const id = this.#seq++;
    return new Promise<Reply>((resolve, reject) => {
      this.#pending.set(id, { resolve, reject });
      worker.postMessage({ id, ...msg }, transfer);
    });
  }
}

registerBackend("cpu-worker", () => new WorkerBackend());
