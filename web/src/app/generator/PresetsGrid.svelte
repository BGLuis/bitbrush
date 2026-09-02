<script lang="ts">
  import { PRESETS, render2DPreview, type Preset, defaultSpots } from "./noisefield-data";
  import { generatorStore } from "./generator-store.svelte";

  function drawThumbnail(canvas: HTMLCanvasElement, p: Preset) {
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    const s = 44;
    canvas.width = s;
    canvas.height = s;
    const spots = defaultSpots(p.colors.length);
    render2DPreview(ctx, s, s, p.type, p.genre, p.direction, p.colors, spots);
  }

  function actionThumbnail(node: HTMLCanvasElement, p: Preset) {
    drawThumbnail(node, p);
    return {
      update(newP: Preset) {
        drawThumbnail(node, newP);
      },
    };
  }
</script>

<div class="presets-container">
  <div class="presets-grid">
    {#each PRESETS as p (p.name)}
      {@const isActive = generatorStore.matchedPreset?.name === p.name}
      <button
        type="button"
        class="preset-card"
        class:active={isActive}
        title={`${p.name} (${p.jp}) — ${p.type} / ${p.genre} / ${p.texture}`}
        onclick={() => generatorStore.applyPreset(p)}
      >
        <canvas class="thumb" use:actionThumbnail={p}></canvas>
        <div class="meta">
          <span class="p-name">{p.name}</span>
          <span class="p-jp">{p.jp}</span>
        </div>
      </button>
    {/each}
  </div>
</div>

<style>
  .presets-container {
    max-height: 230px;
    overflow-y: auto;
    padding: 2px 4px 6px 0;
  }
  .presets-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 8px 6px;
  }
  .preset-card {
    border: 1px solid transparent;
    background: transparent;
    padding: 5px 2px 6px;
    margin: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: background 0.15s ease, transform 0.12s ease;
  }
  .preset-card:hover {
    background: var(--s2);
    transform: translateY(-1px);
  }
  .preset-card.active {
    background: var(--tint);
    border-color: var(--accent);
  }
  .thumb {
    width: 42px;
    height: 42px;
    border-radius: 50%;
    box-shadow: 0 0 0 1px var(--line), 0 2px 6px rgba(0, 0, 0, 0.4);
    pointer-events: none;
    transition: transform 0.15s ease, box-shadow 0.15s ease;
  }
  .preset-card:hover .thumb {
    transform: scale(1.06);
  }
  .preset-card.active .thumb {
    box-shadow: 0 0 0 2px var(--bg), 0 0 0 3.5px var(--accent);
  }
  .meta {
    display: flex;
    flex-direction: column;
    align-items: center;
    line-height: 1.15;
  }
  .p-name {
    font-size: 10.5px;
    font-weight: 600;
    color: var(--text);
    text-align: center;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 58px;
  }
  .p-jp {
    font-size: 9.5px;
    color: var(--mute);
    text-align: center;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 58px;
  }
</style>
