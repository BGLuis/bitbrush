// @ts-nocheck
// Classic Web Worker that owns a second Go/WASM instance, so pixel work
// never blocks the page. Deliberately a plain classic script (no ESM
// import/export) so it can `importScripts('/wasm_exec.js')` — the same Go
// runtime the page loads — and call the bitbrush* globals it registers.
//
// This is the one untyped file on the web side, by the same logic as
// cmd/wasm/main.go on the Go side: it's a thin message router across a
// boundary, and its contract is enforced from the typed WorkerBackend in
// ../backends/worker.ts. Protocol: one request -> one reply keyed by `id`,
// RGBA / GIF buffers transferred (not copied) over postMessage.

importScripts(new URL("/wasm_exec.js", self.location.origin).href);

let wasmReady = null;
function ensureWasm() {
  if (!wasmReady) {
    wasmReady = (async () => {
      const go = new Go();
      const { instance } = await WebAssembly.instantiateStreaming(fetch("/main.wasm"), go.importObject);
      void go.run(instance); // resolves only when the module exits
    })();
  }
  return wasmReady;
}

self.onmessage = async (e) => {
  const req = e.data;
  try {
    await ensureWasm();

    if (req.op === "init") {
      self.postMessage({ id: req.id, ok: true });
      return;
    }

    // Ops with no input buffer.
    if (req.op === "gradient") {
      const r = bitbrushRenderGradient(req.params);
      if (!r.ok || !r.data) throw new Error(r.error || "gradient render failed");
      const out = r.data.slice();
      self.postMessage({ id: req.id, ok: true, buf: out.buffer, width: req.width, height: req.height }, [out.buffer]);
      return;
    }
    if (req.op === "noisefield") {
      const r = bitbrushRenderNoiseField(req.params, req.width, req.height);
      if (!r.ok || !r.data) throw new Error(r.error || "noise field render failed");
      const out = r.data.slice();
      self.postMessage({ id: req.id, ok: true, buf: out.buffer, width: req.width, height: req.height }, [out.buffer]);
      return;
    }
    if (req.op === "gradientCSS") {
      const r = bitbrushGradientCSS(req.params, req.options);
      if (!r.ok) throw new Error(r.error || "gradient css failed");
      self.postMessage({ id: req.id, ok: true, text: r.css });
      return;
    }
    if (req.op === "genPalette") {
      const r = bitbrushGenPalette(req.options);
      if (!r.ok) throw new Error(r.error || "palette generate failed");
      self.postMessage({ id: req.id, ok: true, colors: r.colors });
      return;
    }

    const bytes = new Uint8Array(req.buf);

    if (req.op === "extractPalette") {
      const r = bitbrushExtractPalette(bytes, req.width, req.height, req.options);
      if (!r.ok) throw new Error(r.error || "palette extract failed");
      self.postMessage({ id: req.id, ok: true, colors: r.colors });
      return;
    }

    if (req.op === "ascii") {
      const r = bitbrushAsciiText(bytes, req.width, req.height, req.params);
      if (!r.ok) throw new Error(r.error || "ascii text failed");
      self.postMessage({ id: req.id, ok: true, text: r.text });
      return;
    }

    if (req.op === "gif") {
      const r = bitbrushRenderGIF(req.name, bytes, req.width, req.height, req.keyframes, req.options);
      if (!r.ok || !r.data) throw new Error(r.error || "gif render failed");
      const out = r.data.slice();
      self.postMessage({ id: req.id, ok: true, buf: out.buffer }, [out.buffer]);
      return;
    }

    const r = bitbrushApplyFilter(req.name, bytes, req.width, req.height, req.params);
    if (!r.ok || !r.data) throw new Error(r.error || "filter failed");
    const out = r.data.slice(); // detach from Go's view before transferring
    self.postMessage({ id: req.id, ok: true, buf: out.buffer, width: req.width, height: req.height }, [out.buffer]);
  } catch (err) {
    self.postMessage({ id: req.id, ok: false, error: String(err) });
  }
};
