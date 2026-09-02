<script lang="ts">
  import { onMount } from "svelte";
  import { ui, refs, effectByName } from "./store.svelte";
  import { panel } from "./lib/panel";
  import { makeGifPanel } from "./lib/wrapped";
  import { openFilePicker } from "./lib/image";
  import { generatorStore } from "./generator/generator-store.svelte";
  import CanvasSpotsOverlay from "./generator/CanvasSpotsOverlay.svelte";

  let canvasEl: HTMLCanvasElement;

  onMount(() => {
    refs.canvas = canvasEl;
    return () => {
      refs.canvas = null;
    };
  });

  const DESCS: Record<string, string> = {
    pixelate: "Reduz a imagem a blocos de cor uniformes — 8-bit.",
    sobel: "Detecção de bordas por gradiente Sobel, com limiar opcional.",
    quantize: "Reduz a paleta por median-cut numa métrica de cor perceptual.",
    dither: "Floyd–Steinberg: difusão de erro para poucos níveis ou uma paleta.",
    glitch: "Deslocamento cromático + faixas horizontais — semente fixa, reproduzível.",
    ascii: "Rasteriza a imagem em células de caractere coloridas.",
  };

  const title = $derived(
    ui.mode === "generator"
      ? "Gradiente + Ruído"
      : ui.mode === "palette"
        ? "Paleta"
        : (effectByName(ui.effectName)?.title ?? ui.effectName),
  );
  const desc = $derived(
    ui.mode === "generator"
      ? "Gradiente multi-stop (CSS Color 4) ou campo de ruído generativo — desenha direto no canvas."
      : ui.mode === "palette"
        ? "Extração por median-cut / k-means, ou geração por regra de harmonia."
        : (DESCS[ui.effectName] ?? ""),
  );

  const needsImage = $derived(ui.mode === "filter" || ui.mode === "palette");
  const showEmpty = $derived(needsImage && ui.original === null);
</script>

<section class="stage">
  <header class="hd">
    <span class="t">{title}</span>
    <span class="d">{desc}</span>
  </header>

  <div class="viewport">
    <div
      class="canvas-frame"
      class:generator-mode={ui.mode === "generator"}
      class:hidden={showEmpty}
      style={ui.mode === "generator"
        ? `--ar: ${generatorStore.ratio}; --arn: ${generatorStore.aspectRatio};`
        : ""}
    >
      <canvas bind:this={canvasEl}></canvas>
      {#if ui.mode === "generator" && generatorStore.generatorMode === "noise" && generatorStore.isOrganic}
        <CanvasSpotsOverlay />
      {/if}
    </div>
    {#if showEmpty}
      <div class="empty">
        <p>Carregue uma imagem para começar.</p>
        <button onclick={openFilePicker}>Carregar imagem</button>
      </div>
    {/if}
  </div>

  {#if ui.mode === "filter"}
    {#key ui.effectName}
      <div class="giframe" use:panel={makeGifPanel}></div>
    {/key}
  {/if}
</section>

<style>
  .stage {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
  }
  .hd {
    display: flex;
    align-items: baseline;
    gap: 14px;
    padding: 13px 22px;
    border-bottom: 1px solid var(--line-soft);
  }
  .t {
    font-family: var(--disp);
    font-weight: 600;
    font-size: 15px;
  }
  .d {
    color: var(--mute);
    font-size: 12px;
  }
  .viewport {
    flex: 1;
    min-height: 0;
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background:
      conic-gradient(
          from 90deg at 50% 50%,
          #211e29 0 25%,
          #191720 0 50%,
          #211e29 0 75%,
          #191720 0
        )
        0 0 / 22px 22px;
  }
  .canvas-frame {
    position: relative;
    max-width: 100%;
    max-height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .canvas-frame.generator-mode {
    aspect-ratio: var(--ar, 16/9);
    width: min(100%, calc(68vh * var(--arn, 1.7778)));
    max-height: 100%;
  }
  .canvas-frame.hidden {
    display: none;
  }
  canvas {
    max-width: 100%;
    max-height: 100%;
    border-radius: 5px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
    display: block;
    image-rendering: pixelated;
  }
  .canvas-frame.generator-mode canvas {
    width: 100%;
    height: 100%;
    object-fit: contain;
    image-rendering: auto;
  }
  .empty {
    text-align: center;
    color: var(--dim);
    display: flex;
    flex-direction: column;
    gap: 12px;
    align-items: center;
  }
  .empty button {
    padding: 8px 14px;
    border-radius: var(--radius);
    border: 1px solid var(--accent);
    background: var(--accent);
    color: #fff;
    font-weight: 500;
    cursor: pointer;
  }
  .giframe {
    border-top: 1px solid var(--line-soft);
    padding: 12px 22px;
    max-height: 40%;
    overflow-y: auto;
  }
</style>
