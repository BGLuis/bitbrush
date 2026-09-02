<script lang="ts">
  import { generatorStore } from "./generator-store.svelte";
  import { colorName, clamp } from "./noisefield-data";

  let containerEl: HTMLDivElement;
  let draggingIndex = $state<number | null>(null);

  function handlePointerDown(e: PointerEvent, index: number) {
    e.preventDefault();
    const target = e.currentTarget as HTMLElement;
    target.setPointerCapture(e.pointerId);
    draggingIndex = index;

    const move = (ev: PointerEvent) => {
      if (!containerEl) return;
      const rect = containerEl.getBoundingClientRect();
      const nx = clamp((ev.clientX - rect.left) / rect.width, 0.02, 0.98);
      const ny = clamp((ev.clientY - rect.top) / rect.height, 0.02, 0.98);
      generatorStore.updateSpot(index, nx, ny);
    };

    const up = (ev: PointerEvent) => {
      try {
        target.releasePointerCapture(ev.pointerId);
      } catch {}
      draggingIndex = null;
      target.removeEventListener("pointermove", move);
      target.removeEventListener("pointerup", up);
      target.removeEventListener("pointercancel", up);
    };

    target.addEventListener("pointermove", move);
    target.addEventListener("pointerup", up);
    target.addEventListener("pointercancel", up);
  }

  function handleKeyDown(e: KeyboardEvent, index: number) {
    const deltaMap: Record<string, [number, number]> = {
      ArrowLeft: [-0.01, 0],
      ArrowRight: [0.01, 0],
      ArrowUp: [0, -0.01],
      ArrowDown: [0, 0.01],
    };
    const d = deltaMap[e.key];
    if (!d) return;
    e.preventDefault();
    const sp = generatorStore.spots[index] ?? [0.5, 0.5];
    generatorStore.updateSpot(
      index,
      clamp(sp[0] + d[0], 0.02, 0.98),
      clamp(sp[1] + d[1], 0.02, 0.98),
    );
  }
</script>

<div class="spots-overlay" bind:this={containerEl}>
  {#each generatorStore.spots as sp, i (i)}
    {@const hex = generatorStore.colors[i] || "#FFFFFF"}
    {@const nm = colorName(hex)}
    {@const isFlip = sp[0] > 0.65}
    {@const isDragging = draggingIndex === i}
    <div
      class="spot"
      class:dragging={isDragging}
      class:flip={isFlip}
      style="left: {sp[0] * 100}%; top: {sp[1] * 100}%; background: {hex};"
      tabindex="0"
      role="slider"
      aria-label={`${i + 1}º ponto de cor (${hex}, ${nm})`}
      aria-valuenow={Math.round(sp[0] * 100)}
      aria-valuemin="0"
      aria-valuemax="100"
      aria-valuetext={`${Math.round(sp[0] * 100)}%, ${Math.round(sp[1] * 100)}%`}
      onpointerdown={(e) => handlePointerDown(e, i)}
      onkeydown={(e) => handleKeyDown(e, i)}
    >
      <div class="spot-tag">
        <i style="background: {hex};"></i>
        <span class="nm">{nm}</span>
        <code>{hex}</code>
      </div>
    </div>
  {/each}
</div>

<style>
  .spots-overlay {
    position: absolute;
    inset: 0;
    pointer-events: none;
    z-index: 10;
  }
  .spot {
    position: absolute;
    width: 22px;
    height: 22px;
    margin: -11px 0 0 -11px;
    border-radius: 50%;
    border: 3px solid #ffffff;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.4), 0 4px 12px rgba(0, 0, 0, 0.6);
    cursor: grab;
    pointer-events: auto;
    touch-action: none;
    transition: transform 0.12s ease;
  }
  .spot:hover,
  .spot:focus-visible {
    transform: scale(1.18);
    outline: none;
  }
  .spot.dragging {
    cursor: grabbing;
    transform: scale(1.25);
  }
  .spot-tag {
    position: absolute;
    left: 28px;
    top: 50%;
    transform: translateY(-50%);
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    border-radius: 999px;
    background: rgba(26, 24, 32, 0.88);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
    border: 1px solid var(--line);
    color: var(--text);
    font-size: 10.5px;
    font-weight: 600;
    white-space: nowrap;
    box-shadow: 0 4px 14px rgba(0, 0, 0, 0.45);
    pointer-events: none;
    transition: opacity 0.15s ease;
  }
  .spot.flip .spot-tag {
    left: auto;
    right: 28px;
  }
  .spot-tag i {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    display: inline-block;
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.3);
  }
  .spot-tag .nm {
    font-size: 10px;
    color: var(--dim);
    letter-spacing: 0.3px;
  }
  .spot-tag code {
    font-family: var(--mono);
    font-weight: 500;
    color: var(--text);
    font-size: 10.5px;
  }
</style>
