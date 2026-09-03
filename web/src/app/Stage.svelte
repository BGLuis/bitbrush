<script lang="ts">
  import { onMount } from "svelte";
  import { ui, refs, effectByName } from "./store.svelte";
  import GifPanel from "./gif/GifPanel.svelte";
  import { openFilePicker } from "./lib/image";
  import { generatorStore } from "./generator/generator-store.svelte";
  import CanvasSpotsOverlay from "./generator/CanvasSpotsOverlay.svelte";

  let canvasEl: HTMLCanvasElement;

  onMount(() => {
    refs.canvas = canvasEl;

    const handleKeydown = (e: KeyboardEvent) => {
      if (e.code === "Space" && canAnimate) {
        const activeTag = (document.activeElement?.tagName || "").toLowerCase();
        if (activeTag === "input" || activeTag === "textarea" || activeTag === "select") {
          return;
        }
        e.preventDefault();
        generatorStore.toggleAnimation();
      }
    };

    window.addEventListener("keydown", handleKeydown);

    return () => {
      refs.canvas = null;
      window.removeEventListener("keydown", handleKeydown);
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
    ui.mode === "pattern"
      ? (generatorStore.currentPatternUI?.title ?? "Padrões Algorítmicos")
      : ui.mode === "generator"
        ? "Gradiente + Ruído"
        : ui.mode === "palette"
          ? "Paleta"
          : (effectByName(ui.effectName)?.title ?? ui.effectName),
  );
  const desc = $derived(
    ui.mode === "pattern"
      ? "Geração algorítmica determinística pura — curvas de nível, campo de fluxo, atratores e fractais."
      : ui.mode === "generator"
        ? "Gradiente multi-stop (CSS Color 4) ou campo de ruído generativo — desenha direto no canvas."
        : ui.mode === "palette"
          ? "Extração por median-cut / k-means, ou geração por regra de harmonia."
          : (DESCS[ui.effectName] ?? ""),
  );

  const canAnimate = $derived(
    (ui.mode === "generator" && generatorStore.generatorMode === "noise" && generatorStore.isOrganic) ||
    ((ui.mode === "generator" || ui.mode === "pattern") && generatorStore.generatorMode === "patterns" && generatorStore.patternHasTime)
  );

  const needsImage = $derived(ui.mode === "filter" || ui.mode === "palette");
  const showEmpty = $derived(needsImage && ui.original === null);
</script>

<section class="stage">
  <header class="hd">
    <div class="titles">
      <span class="t">{title}</span>
      <span class="d">{desc}</span>
    </div>
    {#if canAnimate}
      <button
        type="button"
        class="play-toggle-btn"
        class:playing={generatorStore.animate}
        onclick={() => generatorStore.toggleAnimation()}
        title="Pausar / Retomar animação ao vivo [Espaço]"
      >
        <span class="icon">{generatorStore.animate ? "⏸" : "▶"}</span>
        <span>{generatorStore.animate ? "Pausar" : "Animar"}</span>
        <kbd>Espaço</kbd>
      </button>
    {/if}
  </header>

  <div class="viewport">
    <div
      class="canvas-frame"
      class:generator-mode={ui.mode === "generator" || ui.mode === "pattern"}
      class:hidden={showEmpty}
      style={ui.mode === "generator" || ui.mode === "pattern"
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

  {#if ui.mode === "filter" || ui.mode === "generator"}
    <div class="giframe">
      <GifPanel />
    </div>
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
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    padding: 10px 22px;
    border-bottom: 1px solid var(--line-soft);
  }
  .titles {
    display: flex;
    align-items: baseline;
    gap: 14px;
    min-width: 0;
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
  .play-toggle-btn {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    background: var(--surface-2, #18191f);
    color: var(--fg, #f0f0f4);
    border: 1px solid var(--line-soft, #2e303c);
    border-radius: 6px;
    padding: 5px 12px;
    font-size: 12px;
    font-family: inherit;
    cursor: pointer;
    transition: all 0.15s ease;
    user-select: none;
  }
  .play-toggle-btn:hover {
    background: var(--surface-3, #22232c);
    border-color: var(--accent, #a78bfa);
  }
  .play-toggle-btn.playing {
    border-color: var(--accent, #a78bfa);
    color: var(--accent, #a78bfa);
  }
  .play-toggle-btn .icon {
    font-size: 11px;
  }
  .play-toggle-btn kbd {
    font-size: 10px;
    background: rgba(255, 255, 255, 0.08);
    padding: 1px 5px;
    border-radius: 4px;
    border: 1px solid rgba(255, 255, 255, 0.12);
    color: var(--mute, #888);
    font-family: var(--mono, monospace);
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
