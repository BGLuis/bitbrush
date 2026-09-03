<script lang="ts">
  import { ui, effectByName } from "../store.svelte";
  import { generatorStore } from "../generator/generator-store.svelte";
  import { getBackend } from "../../backend";
  import type { FilterParams, GifKeyframe, GifOptions, NoiseFieldParams } from "../../wasm";
  import { calculateOutputDimensions } from "../generator/noisefield-data";

  // Tab mode: 'sweep' = Animate single UI parameter (Loop / Mover Opção), 'keyframes' = A <-> B capture
  let activeTab = $state<"sweep" | "keyframes">("sweep");

  // General GIF settings
  let frames = $state(24);
  let fps = $state(15);
  let maxDim = $state(360);
  let loopForever = $state(true);
  let animateSeed = $state(true);

  // Sweep / Mover Opção state
  let selectedParam = $state("");
  let startVal = $state(0);
  let endVal = $state(100);
  // 'pingpong' = A -> B -> A; 'continuous' = A -> B (for 360 loops); 'forward' = A -> B
  let loopMode = $state<"pingpong" | "continuous" | "forward">("pingpong");

  // Freeform Keyframes state
  let kfA = $state<FilterParams | null>(null);
  let kfB = $state<FilterParams | null>(null);

  // Progress / Status
  let isGenerating = $state(false);
  let statusText = $state("");

  // Inspectable parameters for the current mode
  const currentFilterControls = $derived.by(() => {
    if (ui.mode !== "filter") return [];
    const eff = effectByName(ui.effectName);
    if (!eff) return [];
    return eff.controls.filter((c) => c.type === "range" || c.type === "number" || c.type === "seed");
  });

  const generatorParamsList = [
    { key: "time", label: "Tempo (Fluxo Orgânico)", min: 0, max: 3, step: 0.1, defStart: 0, defEnd: 2.5, defLoop: "pingpong" as const },
    { key: "angle", label: "Direção (Ângulo 0° → 360°)", min: 0, max: 360, step: 5, defStart: 0, defEnd: 360, defLoop: "continuous" as const },
    { key: "distortion", label: "Distorção (Warp)", min: 0, max: 100, step: 1, defStart: 10, defEnd: 90, defLoop: "pingpong" as const },
    { key: "scale", label: "Escala (Frequência)", min: 0, max: 100, step: 1, defStart: 15, defEnd: 85, defLoop: "pingpong" as const },
    { key: "seed", label: "Semente Aleatória", min: 0, max: 100, step: 1, defStart: 0, defEnd: 24, defLoop: "forward" as const },
  ];

  // Auto-init selectedParam whenever effect or mode changes
  $effect(() => {
    if (ui.mode === "filter") {
      const ctrls = currentFilterControls;
      if (ctrls.length > 0) {
        const found = ctrls.find((c) => c.key === selectedParam);
        if (!found) {
          applyFilterParam(ctrls[0].key);
        }
      }
    } else if (ui.mode === "generator") {
      if (generatorStore.generatorMode === "noise") {
        if (!["time", "angle", "distortion", "scale", "seed"].includes(selectedParam)) {
          applyNoiseParam(generatorStore.isOrganic ? "time" : "angle");
        }
      } else {
        if (selectedParam !== "angle") {
          applyPerceptualParam("angle");
        }
      }
    }
  });

  function applyFilterParam(key: string) {
    selectedParam = key;
    const c = currentFilterControls.find((ctrl) => ctrl.key === key);
    if (!c) return;
    const min = c.min !== undefined ? c.min : 0;
    const max = c.max !== undefined ? c.max : min + 100;

    if (key.toLowerCase().includes("angle") || key.toLowerCase().includes("deg")) {
      startVal = 0;
      endVal = 360;
      loopMode = "continuous";
    } else {
      startVal = min;
      endVal = max;
      loopMode = "pingpong";
    }
  }

  function applyNoiseParam(key: string) {
    selectedParam = key;
    const p = generatorParamsList.find((item) => item.key === key);
    if (!p) return;
    startVal = p.defStart;
    endVal = p.defEnd;
    loopMode = p.defLoop;
  }

  function applyPerceptualParam(key: string) {
    selectedParam = key;
    if (key === "angle") {
      startVal = 0;
      endVal = 360;
      loopMode = "continuous";
    }
  }

  // Fast 1-click presets for generators
  function presetOrganicFlow() {
    applyNoiseParam("time");
    frames = 24;
    fps = 15;
    loopMode = "pingpong";
  }

  function preset360Rotation() {
    if (ui.mode === "generator") {
      if (generatorStore.generatorMode === "noise") {
        applyNoiseParam("angle");
      } else {
        applyPerceptualParam("angle");
      }
    } else {
      const angleCtrl = currentFilterControls.find((c) => c.key.toLowerCase().includes("angle"));
      if (angleCtrl) {
        applyFilterParam(angleCtrl.key);
      }
    }
    frames = 24;
    fps = 15;
    loopMode = "continuous";
  }

  // Keyframes capture
  function captureA() {
    kfA = { ...ui.params };
    statusText = "Keyframe A capturado com valores atuais.";
  }
  function captureB() {
    kfB = { ...ui.params };
    statusText = "Keyframe B capturado com valores atuais.";
  }
  function clearKeyframes() {
    kfA = null;
    kfB = null;
    statusText = "";
  }

  async function generateGif() {
    isGenerating = true;
    statusText = "Iniciando renderização do GIF...";

    try {
      const backend = await getBackend(ui.settings.backend);
      const opt: GifOptions = {
        frames: Math.max(2, Math.min(120, frames)),
        fps: Math.max(2, Math.min(30, fps)),
        loop: loopForever,
        pingPong: loopMode === "pingpong",
        maxDimension: maxDim,
      };

      let bytes: Uint8Array;
      let filename: string;

      if (ui.mode === "filter") {
        const src = ui.original ?? ui.preview;
        if (!src) {
          statusText = "Carregue uma imagem antes de gerar o GIF.";
          isGenerating = false;
          return;
        }

        const name = ui.effectName;
        let kfs: GifKeyframe[];

        if (activeTab === "sweep") {
          let a: FilterParams = { ...ui.params, [selectedParam]: startVal };
          let b: FilterParams = { ...ui.params, [selectedParam]: endVal };

          // If loopMode is continuous 360, adjust end value slightly so the last frame doesn't repeat frame 0
          if (loopMode === "continuous" && Math.abs(endVal - startVal) === 360) {
            const step = 360 / opt.frames;
            b = { ...b, [selectedParam]: startVal + 360 - step };
          }

          if (animateSeed && ("seed" in a || "seed" in b)) {
            const base = Number(a.seed ?? 0);
            a = { ...a, seed: base };
            b = { ...b, seed: base + opt.frames - 1 };
          }

          kfs = [
            { t: 0, params: a },
            { t: 1, params: b },
          ];
        } else {
          let a = { ...(kfA ?? ui.params) };
          let b = { ...(kfB ?? ui.params) };
          if (animateSeed && ("seed" in a || "seed" in b)) {
            const base = Number(a.seed ?? 0);
            a = { ...a, seed: base };
            b = { ...b, seed: base + opt.frames - 1 };
          }
          kfs = [
            { t: 0, params: a },
            { t: 1, params: b },
          ];
        }

        statusText = `Renderizando ${opt.frames} quadros (${opt.maxDimension}px máx)...`;
        bytes = await backend.renderGIF(name, src, kfs, opt);
        filename = `bitbrush-${name}-anim.gif`;
      } else {
        // Generator Mode (NoiseField / Multi-stop)
        statusText = `Renderizando ${opt.frames} quadros do gradiente generativo...`;

        const { w, h } = calculateOutputDimensions(generatorStore.ratio, maxDim);
        const baseNoise = generatorStore.getNoiseParams();

        let startP: NoiseFieldParams = { ...baseNoise };
        let endP: NoiseFieldParams = { ...baseNoise };

        if (selectedParam === "time") {
          startP.time = startVal;
          endP.time = endVal;
        } else if (selectedParam === "angle") {
          startP.angle = startVal;
          endP.angle = endVal;
        } else if (selectedParam === "distortion") {
          startP.distortion = startVal;
          endP.distortion = endVal;
        } else if (selectedParam === "scale") {
          startP.scale = startVal;
          endP.scale = endVal;
        } else if (selectedParam === "seed") {
          startP.seed = startVal;
          endP.seed = endVal;
        }

        bytes = await backend.renderNoiseFieldGIF(startP, endP, w, h, opt);
        filename = `bitbrush-gradient-${generatorStore.field}-${selectedParam}-loop.gif`;
      }

      // Download generated GIF
      const buf = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;
      const blob = new Blob([buf], {
        type: "image/gif",
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 15_000);

      statusText = `✓ GIF pronto! ${(bytes.length / 1024).toFixed(0)} KB · ${opt.frames} quadros a ${opt.fps} FPS`;
    } catch (err) {
      console.error("Erro na geração do GIF:", err);
      statusText = `Falha ao gerar GIF: ${String(err)}`;
    } finally {
      isGenerating = false;
    }
  }
</script>

<details class="gif-panel">
  <summary class="panel-hd">
    <div class="summary-left">
      <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="2" y="2" width="20" height="20" rx="3" />
        <circle cx="8.5" cy="8.5" r="1.5" />
        <path d="M21 15l-5-5L5 21" />
      </svg>
      <span>Exportar GIF Animado {ui.mode === "generator" ? "· Gradiente" : `· ${effectByName(ui.effectName)?.title ?? ui.effectName}`}</span>
    </div>
    {#if statusText}
      <span class="pill-status">{statusText}</span>
    {/if}
  </summary>

  <div class="control-panel">
    <!-- Presets de 1 clique para animação -->
    {#if ui.mode === "generator" && generatorStore.generatorMode === "noise"}
      <div class="quick-presets">
        <span class="preset-label">Loops rápidos:</span>
        <button type="button" class="preset-chip" onclick={presetOrganicFlow}>
          🌊 Loop de Fluxo Orgânico (Tempo)
        </button>
        <button type="button" class="preset-chip" onclick={preset360Rotation}>
          🧭 Rotação 360° Contínua
        </button>
      </div>
    {/if}

    <!-- Tabs: Mover Opção vs Keyframes A/B -->
    <div class="gif-tabs">
      <button
        type="button"
        class="tab-item"
        class:active={activeTab === "sweep"}
        onclick={() => (activeTab = "sweep")}
      >
        Mover Opção da UI (Loop / Varredura)
      </button>
      <button
        type="button"
        class="tab-item"
        class:active={activeTab === "keyframes"}
        onclick={() => (activeTab = "keyframes")}
      >
        Keyframes A ↔ B (Livre)
      </button>
    </div>

    {#if activeTab === "sweep"}
      <!-- Sweep Mode -->
      <div class="sweep-grid">
        <div class="field-col">
          <label for="gif-param-select">Opção da UI a mover</label>
          {#if ui.mode === "filter"}
            <select
              id="gif-param-select"
              value={selectedParam}
              onchange={(e) => applyFilterParam(e.currentTarget.value)}
            >
              {#each currentFilterControls as c (c.key)}
                <option value={c.key}>{c.label} ({c.key})</option>
              {/each}
            </select>
          {:else}
            <select
              id="gif-param-select"
              value={selectedParam}
              onchange={(e) => applyNoiseParam(e.currentTarget.value)}
            >
              {#each generatorParamsList as item (item.key)}
                <option value={item.key}>{item.label}</option>
              {/each}
            </select>
          {/if}
        </div>

        <div class="field-row">
          <div class="field-col">
            <label for="gif-start-val">Início (A)</label>
            <input
              id="gif-start-val"
              type="number"
              step="any"
              bind:value={startVal}
            />
          </div>
          <div class="field-col">
            <label for="gif-end-val">Fim (B)</label>
            <input
              id="gif-end-val"
              type="number"
              step="any"
              bind:value={endVal}
            />
          </div>
          <div class="field-col">
            <label for="gif-loop-mode">Tipo de Loop</label>
            <select id="gif-loop-mode" bind:value={loopMode}>
              <option value="pingpong">🔁 Ping-Pong (A → B → A)</option>
              <option value="continuous">🔄 Circular Contínuo (A → B)</option>
              <option value="forward">⏩ Direto Linear (A → B)</option>
            </select>
          </div>
        </div>
      </div>
    {:else}
      <!-- Keyframes A/B Mode -->
      <div class="kfs-actions">
        <button type="button" class="btn-sub" onclick={captureA}>
          Capturar Início (A) {kfA ? "✓" : ""}
        </button>
        <button type="button" class="btn-sub" onclick={captureB}>
          Capturar Fim (B) {kfB ? "✓" : ""}
        </button>
        <button type="button" class="btn-sub" onclick={clearKeyframes}>
          Limpar Keyframes
        </button>
      </div>
      <div class="kfs-summary">
        {kfA ? "A gravado" : "A vazio (usará atual)"} · {kfB ? "B gravado" : "B vazio (usará atual)"}
      </div>
    {/if}

    <!-- GIF Technical Settings -->
    <div class="settings-row">
      <div class="setting-item">
        <label for="gif-frames">Quadros</label>
        <select id="gif-frames" bind:value={frames}>
          <option value={12}>12 quadros</option>
          <option value={16}>16 quadros</option>
          <option value={24}>24 quadros (padrão)</option>
          <option value={32}>32 quadros</option>
          <option value={48}>48 quadros (fluido)</option>
        </select>
      </div>

      <div class="setting-item">
        <label for="gif-fps">FPS</label>
        <select id="gif-fps" bind:value={fps}>
          <option value={10}>10 fps</option>
          <option value={12}>12 fps</option>
          <option value={15}>15 fps (padrão)</option>
          <option value={20}>20 fps</option>
          <option value={24}>24 fps</option>
          <option value={30}>30 fps</option>
        </select>
      </div>

      <div class="setting-item">
        <label for="gif-max-dim">Resolução</label>
        <select id="gif-max-dim" bind:value={maxDim}>
          <option value={240}>240 px (leve)</option>
          <option value={360}>360 px (equilibrado)</option>
          <option value={480}>480 px (nítido)</option>
          <option value={720}>720 px (HD)</option>
        </select>
      </div>

      <div class="setting-item checkbox">
        <label>
          <input type="checkbox" bind:checked={loopForever} />
          <span>Loop infinito</span>
        </label>
      </div>

      {#if ui.mode === "filter" && (ui.effectName === "glitch" || ui.effectName === "stipple")}
        <div class="setting-item checkbox">
          <label>
            <input type="checkbox" bind:checked={animateSeed} />
            <span>Variar semente a cada quadro</span>
          </label>
        </div>
      {/if}
    </div>

    <!-- Action Bar -->
    <div class="action-row">
      <button
        type="button"
        class="gif-generate-btn"
        disabled={isGenerating}
        onclick={generateGif}
      >
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="5 3 19 12 5 21 5 3" />
        </svg>
        {isGenerating ? "Gerando GIF Animado..." : "Gerar e Baixar GIF"}
      </button>
    </div>
  </div>
</details>

<style>
  .gif-panel {
    background: var(--s1);
    border: 1px solid var(--line);
    border-radius: var(--radius);
    padding: 10px 14px;
  }
  .panel-hd {
    display: flex;
    align-items: center;
    justify-content: space-between;
    cursor: pointer;
    user-select: none;
    font-size: 13px;
    font-weight: 600;
    color: var(--text);
  }
  .summary-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .summary-left svg {
    color: var(--accent);
  }
  .pill-status {
    font-size: 11px;
    font-family: var(--mono);
    color: var(--dim);
    background: var(--s2);
    padding: 3px 8px;
    border-radius: 999px;
    border: 1px solid var(--line);
  }
  .control-panel {
    margin-top: 12px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    border-top: 1px solid var(--line-soft);
    padding-top: 12px;
  }
  .quick-presets {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }
  .preset-label {
    font-size: 11px;
    color: var(--mute);
    font-weight: 500;
  }
  .preset-chip {
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: 999px;
    padding: 4px 10px;
    font-size: 11px;
    color: var(--dim);
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .preset-chip:hover {
    background: var(--tint);
    border-color: var(--accent);
    color: var(--text);
  }
  .gif-tabs {
    display: grid;
    grid-template-columns: 1fr 1fr;
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    padding: 3px;
    gap: 4px;
  }
  .tab-item {
    border: 0;
    background: transparent;
    color: var(--dim);
    font-size: 12px;
    font-weight: 500;
    padding: 5px 8px;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .tab-item:hover {
    color: var(--text);
  }
  .tab-item.active {
    background: var(--s3);
    color: #fff;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
  }
  .sweep-grid {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .field-row {
    display: grid;
    grid-template-columns: 1fr 1fr 1.2fr;
    gap: 8px;
  }
  .field-col {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .field-col label,
  .setting-item label {
    font-size: 11px;
    color: var(--dim);
  }
  .field-col select,
  .field-col input,
  .setting-item select {
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    padding: 5px 8px;
    color: var(--text);
    font-size: 12px;
  }
  .field-col input {
    font-family: var(--mono);
  }
  .kfs-actions {
    display: flex;
    gap: 8px;
  }
  .btn-sub {
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    padding: 6px 12px;
    font-size: 11.5px;
    color: var(--text);
    cursor: pointer;
  }
  .btn-sub:hover {
    border-color: var(--accent);
    background: var(--s3);
  }
  .kfs-summary {
    font-size: 11px;
    color: var(--mute);
    font-family: var(--mono);
  }
  .settings-row {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
    background: var(--s2);
    padding: 8px 12px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--line-soft);
  }
  .setting-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .setting-item.checkbox {
    flex-direction: row;
    align-items: center;
    margin-top: 14px;
  }
  .setting-item.checkbox label {
    display: flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
    font-size: 11.5px;
    color: var(--text);
  }
  .action-row {
    display: flex;
    justify-content: flex-end;
  }
  .gif-generate-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    background: var(--accent);
    color: #fff;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 8px 18px;
    font-size: 12.5px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s ease;
  }
  .gif-generate-btn:hover:not(:disabled) {
    background: var(--accent-press);
  }
  .gif-generate-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
</style>
