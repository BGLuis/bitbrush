<script lang="ts">
  import { onMount } from "svelte";
  import { refs } from "../store.svelte";
  import { selectionStore } from "./selection-store.svelte";
  import { simplifyPolygon, type Pt } from "./lib/simplify";
  import type { MaskParams } from "../../wasm";

  const MIN_POINT_DIST = 0.004; // normalized gap while capturing pointermove
  const MAX_POLYGON_POINTS = 100;
  const MIN_DRAG_SIZE = 0.01; // guards a stray click from committing a 0-size mask

  let rootEl: HTMLDivElement | undefined = $state();
  let rect = $state({ left: 0, top: 0, width: 0, height: 0 });
  let dragging = $state(false);
  let p0 = $state<Pt | null>(null);
  let p1 = $state<Pt | null>(null);
  let poly = $state<Pt[]>([]);

  function syncRect() {
    const canvas = refs.canvas;
    const frame = canvas?.closest(".canvas-frame") as HTMLElement | null;
    if (!canvas || !frame) return;
    const cr = canvas.getBoundingClientRect();
    const fr = frame.getBoundingClientRect();
    rect = { left: cr.left - fr.left, top: cr.top - fr.top, width: cr.width, height: cr.height };
  }

  onMount(() => {
    syncRect();
    const ro = new ResizeObserver(syncRect);
    if (refs.canvas) ro.observe(refs.canvas);
    window.addEventListener("resize", syncRect);
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") reset();
    };
    window.addEventListener("keydown", onKey);
    return () => {
      ro.disconnect();
      window.removeEventListener("resize", syncRect);
      window.removeEventListener("keydown", onKey);
    };
  });

  const clamp01 = (v: number) => Math.min(1, Math.max(0, v));
  function toNorm(e: PointerEvent): Pt {
    const r = rootEl!.getBoundingClientRect(); // == canvas rect, by construction
    return { x: clamp01((e.clientX - r.left) / r.width), y: clamp01((e.clientY - r.top) / r.height) };
  }
  function dist(a: Pt, b: Pt): number {
    return Math.hypot(a.x - b.x, a.y - b.y);
  }

  function onDown(e: PointerEvent) {
    rootEl!.setPointerCapture(e.pointerId);
    const pt = toNorm(e);
    dragging = true;
    if (selectionStore.tool === "polygon") poly = [pt];
    else {
      p0 = pt;
      p1 = pt;
    }
  }
  function onMove(e: PointerEvent) {
    if (!dragging) return;
    const pt = toNorm(e);
    if (selectionStore.tool === "polygon") {
      const last = poly[poly.length - 1];
      if (!last || dist(last, pt) >= MIN_POINT_DIST) poly = [...poly, pt];
    } else {
      p1 = pt;
    }
  }
  function onUp(e: PointerEvent) {
    if (!dragging) return;
    dragging = false;
    try {
      rootEl!.releasePointerCapture(e.pointerId);
    } catch {
      // already released (e.g. pointercancel raced us) — nothing to do
    }
    commit();
  }

  function reset() {
    dragging = false;
    p0 = null;
    p1 = null;
    poly = [];
    selectionStore.cancel();
  }

  function commit() {
    let mask: MaskParams | null = null;
    if (selectionStore.tool === "polygon") {
      let eps = 0.004;
      let pts = simplifyPolygon(poly, eps, MAX_POLYGON_POINTS);
      let tries = 0;
      while (pts.length > MAX_POLYGON_POINTS && tries < 8) {
        eps *= 1.5;
        pts = simplifyPolygon(poly, eps, MAX_POLYGON_POINTS);
        tries++;
      }
      if (pts.length >= 3) mask = { kind: "polygon", points: pts, feather: 0, invert: false };
    } else if (p0 && p1) {
      const x = Math.min(p0.x, p1.x);
      const y = Math.min(p0.y, p1.y);
      const w = Math.abs(p1.x - p0.x);
      const h = Math.abs(p1.y - p0.y);
      if (w >= MIN_DRAG_SIZE && h >= MIN_DRAG_SIZE) {
        mask = { kind: selectionStore.tool, x, y, w, h, feather: 0.1, invert: false };
      }
    }
    if (mask) selectionStore.commit(mask);
    else selectionStore.cancel(); // stray click / too-small drag: leave any existing mask untouched
    p0 = null;
    p1 = null;
    poly = [];
  }
</script>

<!-- A freehand drawing surface has no meaningful keyboard equivalent to a
     pointer drag; Escape (handled above) is the keyboard affordance. -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  bind:this={rootEl}
  class="selection-overlay"
  style="left:{rect.left}px; top:{rect.top}px; width:{rect.width}px; height:{rect.height}px"
  onpointerdown={onDown}
  onpointermove={onMove}
  onpointerup={onUp}
  onpointercancel={reset}
>
  <svg viewBox="0 0 1 1" preserveAspectRatio="none">
    {#if selectionStore.tool === "rect" && p0 && p1}
      <rect
        x={Math.min(p0.x, p1.x)}
        y={Math.min(p0.y, p1.y)}
        width={Math.abs(p1.x - p0.x)}
        height={Math.abs(p1.y - p0.y)}
        vector-effect="non-scaling-stroke"
      />
    {:else if selectionStore.tool === "ellipse" && p0 && p1}
      <ellipse
        cx={(p0.x + p1.x) / 2}
        cy={(p0.y + p1.y) / 2}
        rx={Math.abs(p1.x - p0.x) / 2}
        ry={Math.abs(p1.y - p0.y) / 2}
        vector-effect="non-scaling-stroke"
      />
    {:else if selectionStore.tool === "polygon" && poly.length > 1}
      <polygon points={poly.map((p) => `${p.x},${p.y}`).join(" ")} vector-effect="non-scaling-stroke" />
    {/if}
  </svg>
</div>

<style>
  .selection-overlay {
    position: absolute;
    z-index: 12;
    cursor: crosshair;
    touch-action: none;
  }
  .selection-overlay svg {
    width: 100%;
    height: 100%;
    display: block;
  }
  .selection-overlay svg :global(rect),
  .selection-overlay svg :global(ellipse),
  .selection-overlay svg :global(polygon) {
    fill: color-mix(in srgb, var(--accent) 18%, transparent);
    stroke: var(--accent);
    stroke-width: 1.5px;
  }
</style>
