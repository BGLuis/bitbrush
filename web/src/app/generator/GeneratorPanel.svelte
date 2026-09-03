<script lang="ts">
  import { onMount } from "svelte";
  import { generatorStore } from "./generator-store.svelte";
  import {
    GENRES,
    TYPES,
    TEXTURES,
    RATIOS,
    TYPE_NOTES,
    stopsFrom,
    HEX_REGEX,
  } from "./noisefield-data";
  import Compass from "./Compass.svelte";
  import PresetsGrid from "./PresetsGrid.svelte";

  let showPresets = $state(true);

  onMount(() => {
    generatorStore.scheduleRender();
    generatorStore.syncAnimation();
    return () => {
      generatorStore.stopAnimation();
    };
  });

  const SPACES = [
    ["oklab", "OKLab"],
    ["oklch", "OKLCh"],
    ["lab", "CIE Lab"],
    ["lch", "CIE LCh"],
    ["hsl", "HSL"],
    ["linear", "RGB linear"],
    ["srgb", "sRGB"],
  ];
  const HUE_ARCS = [
    ["shorter", "arco curto"],
    ["longer", "arco longo"],
    ["increasing", "crescente"],
    ["decreasing", "decrescente"],
  ];
  const CSS_KINDS = [
    ["linear", "linear"],
    ["radial", "radial"],
    ["conic", "conic"],
  ];
  const CYLINDRICAL = new Set(["hsl", "lch", "oklch"]);
</script>

<div class="gen-panel">
  <!-- Sub-mode Selector -->
  <div class="mode-tabs">
    <button
      type="button"
      class="tab-btn"
      class:active={generatorStore.generatorMode === "noise"}
      onclick={() => {
        generatorStore.generatorMode = "noise";
        generatorStore.scheduleRender();
        generatorStore.syncAnimation();
      }}
    >
      Ruído
    </button>
    <button
      type="button"
      class="tab-btn"
      class:active={generatorStore.generatorMode === "multistop"}
      onclick={() => {
        generatorStore.generatorMode = "multistop";
        generatorStore.stopAnimation();
        generatorStore.scheduleRender();
      }}
    >
      CSS
    </button>
  </div>

  {#if generatorStore.statusMessage}
    <div class="status-banner" role="status">
      {generatorStore.statusMessage}
    </div>
  {/if}

  {#if generatorStore.generatorMode === "noise"}
    <!-- Presets Section -->
    <section class="section">
      <div class="section-hd">
        <div class="label-with-badge">
          <span class="sec-title">Recomendações</span>
          {#if generatorStore.matchedPreset}
            <span class="active-badge">{generatorStore.matchedPreset.name}</span>
          {:else}
            <span class="custom-badge">Personalizado</span>
          {/if}
        </div>
        <button
          type="button"
          class="toggle-btn"
          onclick={() => (showPresets = !showPresets)}
        >
          {showPresets ? "Ocultar" : "Mostrar"} ({generatorStore.matchedPreset ? generatorStore.matchedPreset.name : "28"})
        </button>
      </div>
      {#if showPresets}
        <PresetsGrid />
      {/if}
    </section>

    <!-- Shape / Field Selection -->
    <section class="section">
      <div class="sec-title">Forma / Geometria</div>
      <div class="chips-grid g4">
        {#each TYPES as [id, label]}
          <button
            type="button"
            class="chip-btn"
            class:active={generatorStore.field === id}
            onclick={() => generatorStore.setField(id)}
          >
            {label.split(" ")[0]}
          </button>
        {/each}
      </div>
      <p class="field-note">{TYPE_NOTES[generatorStore.field] || ""}</p>
    </section>

    <!-- Style / Genre Selection -->
    <section class="section">
      <div class="sec-title">Estilo Cromático</div>
      <div class="chips-grid g2">
        {#each GENRES as [id, label]}
          {@const styleStops = stopsFrom(id, generatorStore.colors)}
          <button
            type="button"
            class="chip-btn style-btn"
            class:active={generatorStore.style === id}
            onclick={() => generatorStore.setStyle(id)}
          >
            <span class="style-name">{label}</span>
            <span
              class="style-bar"
              style="background: linear-gradient(90deg, {styleStops.join(',')});"
            ></span>
          </button>
        {/each}
      </div>
    </section>

    <!-- Colors Section -->
    <section class="section">
      <div class="section-hd">
        <span class="sec-title">Cores ({generatorStore.colors.length}/8)</span>
        <div class="btn-group">
          <button
            type="button"
            class="action-btn"
            onclick={() => generatorStore.shuffle()}
            title="Sorteia paleta harmônica e posições"
          >
            <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M16 3h5v5M4 20 21 3M21 16v5h-5M15 15l6 6M4 4l5 5"/>
            </svg>
            Sortear
          </button>
          <button
            type="button"
            class="action-btn"
            disabled={generatorStore.colors.length >= 8}
            onclick={() => generatorStore.addColor()}
          >
            + Cor
          </button>
        </div>
      </div>
      <div class="color-list">
        {#each generatorStore.colors as c, i (i)}
          {@const isValid = HEX_REGEX.test(c.trim())}
          <div class="color-row">
            <input
              type="color"
              class="swatch"
              value={isValid ? c : "#FFFFFF"}
              aria-label={`Cor ${i + 1}`}
              oninput={(e) => generatorStore.updateColor(i, (e.target as HTMLInputElement).value.toUpperCase())}
            />
            <input
              type="text"
              class="hex-input"
              class:invalid={!isValid}
              value={c}
              placeholder="#RRGGBB"
              maxlength="7"
              aria-label={`Hexadecimal da cor ${i + 1}`}
              oninput={(e) => generatorStore.updateColor(i, (e.target as HTMLInputElement).value.trim())}
            />
            <span class="color-tag" title="Nome estimado">{generatorStore.colorNames[i] || ""}</span>
            <button
              type="button"
              class="remove-btn"
              disabled={generatorStore.colors.length <= 2}
              title="Remover cor"
              onclick={() => generatorStore.removeColor(i)}
            >
              ✕
            </button>
          </div>
        {/each}
      </div>
    </section>

    <!-- Field parameters (Scale & Distortion) for organic fields -->
    {#if generatorStore.isOrganic}
      <section class="section">
        <div class="sec-title">Campo de Ruído</div>
        <div class="slider-control">
          <div class="slider-hd">
            <span>Escala (Frequência)</span>
            <span class="mono-val">{generatorStore.scale}%</span>
          </div>
          <input
            type="range"
            min="0"
            max="100"
            value={generatorStore.scale}
            oninput={(e) => generatorStore.setScale(Number((e.target as HTMLInputElement).value))}
          />
        </div>
        <div class="slider-control">
          <div class="slider-hd">
            <span>Distorção (Warp)</span>
            <span class="mono-val">{generatorStore.distortion}%</span>
          </div>
          <input
            type="range"
            min="0"
            max="100"
            value={generatorStore.distortion}
            oninput={(e) => generatorStore.setDistortion(Number((e.target as HTMLInputElement).value))}
          />
        </div>
      </section>
    {/if}

    <!-- Direction Compass -->
    {#if generatorStore.isAngled}
      <section class="section">
        <div class="sec-title">Direção</div>
        <Compass />
      </section>
    {/if}

    <!-- Texture -->
    <section class="section">
      <div class="sec-title">Textura da Superfície</div>
      <div class="chips-grid g3">
        {#each TEXTURES as [id, label]}
          <button
            type="button"
            class="chip-btn"
            class:active={generatorStore.texture === id}
            onclick={() => generatorStore.setTexture(id)}
          >
            {label}
          </button>
        {/each}
      </div>
    </section>

    <!-- Aspect Ratio -->
    <section class="section">
      <div class="sec-title">Proporção (Aspect Ratio)</div>
      <div class="chips-grid g3">
        {#each RATIOS as [id, val]}
          <button
            type="button"
            class="chip-btn"
            class:active={generatorStore.ratio === val}
            onclick={() => generatorStore.setRatio(val)}
          >
            {id}
          </button>
        {/each}
      </div>
    </section>

    <!-- Actions & Export -->
    <section class="section export-sec">
      <div class="sec-title">Exportação</div>
      <div class="export-actions">
        <button
          type="button"
          class="export-btn primary"
          onclick={() => generatorStore.exportHighResPNG()}
        >
          <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/>
          </svg>
          PNG 1600px
        </button>
        <button
          type="button"
          class="export-btn"
          onclick={() => generatorStore.exportSVG()}
        >
          Baixar SVG
        </button>
        <button
          type="button"
          class="export-btn"
          onclick={() => generatorStore.copyCSS()}
        >
          {generatorStore.copiedCSS ? "✓ Copiado!" : "Copiar CSS"}
        </button>
      </div>

      {#if generatorStore.isOrganic}
        <div class="anim-control">
          <label class="checkbox-label" title="Pausar / Retomar animação ao vivo [Espaço]">
            <input
              type="checkbox"
              checked={generatorStore.animate}
              onchange={() => generatorStore.toggleAnimation()}
            />
            <span>Animar fluxo no preview vivo</span>
            <kbd style="font-size: 10px; margin-left: 6px; opacity: 0.6; font-family: var(--mono, monospace);">Espaço</kbd>
          </label>
        </div>
      {/if}

      <div class="css-box">
        <textarea readonly spellcheck="false" rows="6" value={generatorStore.cssOutput}></textarea>
      </div>
    </section>
  {:else}
    <!-- Multi-stop Perceptual Gradient Mode -->
    <section class="section">
      <div class="sec-title">Parâmetros Perceptuais</div>
      <div class="form-row">
        <label for="ms-space">Espaço de cor</label>
        <select
          id="ms-space"
          bind:value={generatorStore.msSpace}
          onchange={() => generatorStore.scheduleRender()}
        >
          {#each SPACES as [id, label]}
            <option value={id}>{label}</option>
          {/each}
        </select>
      </div>

      {#if CYLINDRICAL.has(generatorStore.msSpace)}
        <div class="form-row">
          <label for="ms-hue">Arco de matiz</label>
          <select
            id="ms-hue"
            bind:value={generatorStore.msHue}
            onchange={() => generatorStore.scheduleRender()}
          >
            {#each HUE_ARCS as [id, label]}
              <option value={id}>{label}</option>
            {/each}
          </select>
        </div>
      {/if}

      <div class="form-row">
        <label for="ms-easing">Easing</label>
        <input
          id="ms-easing"
          type="text"
          bind:value={generatorStore.msEasing}
          placeholder="linear · ease-in-out · steps(4)"
          oninput={() => generatorStore.scheduleRender()}
        />
      </div>

      <div class="form-row">
        <label for="ms-kind">Tipo CSS</label>
        <select
          id="ms-kind"
          bind:value={generatorStore.msKind}
          onchange={() => generatorStore.scheduleRender()}
        >
          {#each CSS_KINDS as [id, label]}
            <option value={id}>{label}</option>
          {/each}
        </select>
      </div>

      <div class="form-row">
        <label for="ms-angle">Ângulo</label>
        <input
          id="ms-angle"
          type="number"
          min="0"
          max="360"
          bind:value={generatorStore.msAngle}
          oninput={() => generatorStore.scheduleRender()}
        />
      </div>

      <div class="form-row checkbox">
        <label>
          <input
            type="checkbox"
            bind:checked={generatorStore.msNative}
            onchange={() => generatorStore.scheduleRender()}
          />
          <span>CSS nativo (in oklch)</span>
        </label>
      </div>

      {#if !generatorStore.msNative}
        <div class="form-row">
          <label for="ms-samples">Amostras (CSS assado)</label>
          <input
            id="ms-samples"
            type="number"
            min="2"
            max="64"
            bind:value={generatorStore.msSamples}
            oninput={() => generatorStore.scheduleRender()}
          />
        </div>
      {/if}
    </section>

    <!-- Multi-stop color stops -->
    <section class="section">
      <div class="section-hd">
        <span class="sec-title">Stops ({generatorStore.msStops.length})</span>
        <button
          type="button"
          class="action-btn"
          onclick={() => {
            generatorStore.msStops.push({ color: "#8b5cf6", pos: 1 });
            generatorStore.scheduleRender();
          }}
        >
          + Stop
        </button>
      </div>
      <div class="color-list">
        {#each generatorStore.msStops as s, i (i)}
          <div class="color-row">
            <input
              type="color"
              class="swatch"
              bind:value={s.color}
              oninput={() => generatorStore.scheduleRender()}
            />
            <input
              type="number"
              class="pos-input"
              min="0"
              max="1"
              step="0.01"
              bind:value={s.pos}
              oninput={() => generatorStore.scheduleRender()}
            />
            <button
              type="button"
              class="remove-btn"
              disabled={generatorStore.msStops.length <= 2}
              onclick={() => {
                generatorStore.msStops.splice(i, 1);
                generatorStore.scheduleRender();
              }}
            >
              ✕
            </button>
          </div>
        {/each}
      </div>
    </section>

    <!-- Multi-stop export -->
    <section class="section export-sec">
      <div class="sec-title">Exportação</div>
      <div class="export-actions">
        <button
          type="button"
          class="export-btn primary"
          onclick={() => generatorStore.exportHighResPNG()}
        >
          Baixar PNG
        </button>
        <button
          type="button"
          class="export-btn"
          onclick={() => generatorStore.exportSVG()}
        >
          Baixar SVG
        </button>
        <button
          type="button"
          class="export-btn"
          onclick={() => generatorStore.copyCSS()}
        >
          {generatorStore.copiedCSS ? "✓ Copiado!" : "Copiar CSS"}
        </button>
      </div>
        <div class="css-box">
        <textarea readonly spellcheck="false" rows="5" value={generatorStore.cssOutput}></textarea>
      </div>
    </section>
  {/if}
</div>

<style>
  .gen-panel {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .mode-tabs {
    display: grid;
    grid-template-columns: 1fr 1fr;
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius);
    padding: 3px;
    gap: 3px;
  }
  .tab-btn {
    padding: 6px 4px;
    border: 0;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--dim);
    font-size: 11.5px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s ease;
    text-align: center;
  }
  .tab-btn:hover {
    color: var(--text);
  }
  .tab-btn.active {
    background: var(--s3);
    color: #fff;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
  }
  .status-banner {
    padding: 7px 10px;
    background: var(--tint);
    border: 1px solid var(--accent);
    border-radius: var(--radius-sm);
    color: var(--text);
    font-size: 11.5px;
  }
  .section {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-bottom: 12px;
    border-bottom: 1px solid var(--line-soft);
  }
  .section:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }
  .section-hd {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
  }
  .label-with-badge {
    display: flex;
    align-items: center;
    gap: 7px;
  }
  .sec-title {
    font-size: 11.5px;
    font-weight: 600;
    color: var(--dim);
    text-transform: uppercase;
    letter-spacing: 0.8px;
  }
  .active-badge {
    font-size: 10px;
    padding: 2px 7px;
    border-radius: 999px;
    background: var(--tint);
    color: var(--accent);
    font-weight: 600;
    border: 1px solid var(--accent);
  }
  .custom-badge {
    font-size: 10px;
    padding: 2px 7px;
    border-radius: 999px;
    background: var(--s3);
    color: var(--mute);
    font-weight: 500;
  }
  .toggle-btn {
    border: 0;
    background: transparent;
    color: var(--dim);
    font-size: 11px;
    cursor: pointer;
    text-decoration: underline;
  }
  .toggle-btn:hover {
    color: var(--text);
  }
  .chips-grid {
    display: grid;
    gap: 5px;
  }
  .chips-grid.g2 {
    grid-template-columns: repeat(2, 1fr);
  }
  .chips-grid.g3 {
    grid-template-columns: repeat(3, 1fr);
  }
  .chips-grid.g4 {
    grid-template-columns: repeat(4, 1fr);
  }
  .chip-btn {
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--dim);
    border-radius: var(--radius-sm);
    padding: 6px 5px;
    font-size: 11.5px;
    font-weight: 500;
    cursor: pointer;
    text-align: center;
    transition: all 0.12s ease;
  }
  .chip-btn:hover {
    background: var(--s3);
    color: var(--text);
    border-color: var(--line);
  }
  .chip-btn.active {
    background: var(--tint);
    color: #fff;
    border-color: var(--accent);
    box-shadow: inset 0 0 0 1px var(--accent);
  }
  .style-btn {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    padding: 6px 8px 7px;
    gap: 4px;
    text-align: left;
  }
  .style-name {
    font-size: 11.5px;
    font-weight: 500;
  }
  .style-bar {
    width: 100%;
    height: 4px;
    border-radius: 2px;
  }
  .field-note {
    margin: 2px 0 0;
    font-size: 11px;
    color: var(--mute);
    line-height: 1.45;
  }
  .btn-group {
    display: flex;
    gap: 6px;
  }
  .action-btn {
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--text);
    padding: 3px 8px;
    font-size: 11px;
    border-radius: var(--radius-sm);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 5px;
  }
  .action-btn:hover:not(:disabled) {
    background: var(--s3);
    border-color: var(--accent);
  }
  .action-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  .color-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .color-row {
    display: grid;
    grid-template-columns: 32px 82px 1fr 24px;
    gap: 8px;
    align-items: center;
  }
  .swatch {
    width: 32px;
    height: 30px;
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    background: transparent;
    padding: 1px;
    cursor: pointer;
  }
  .hex-input {
    width: 100%;
    font-family: var(--mono);
    font-size: 11.5px;
    padding: 4px 6px;
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    color: var(--text);
    text-transform: uppercase;
  }
  .hex-input.invalid {
    border-color: #e5484d;
  }
  .color-tag {
    font-size: 10px;
    color: var(--mute);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .remove-btn {
    width: 24px;
    height: 24px;
    border: 0;
    background: transparent;
    color: var(--dim);
    cursor: pointer;
    border-radius: 50%;
    display: grid;
    place-items: center;
    font-size: 12px;
  }
  .remove-btn:hover:not(:disabled) {
    background: var(--s3);
    color: var(--text);
  }
  .remove-btn:disabled {
    opacity: 0.2;
    cursor: not-allowed;
  }
  .slider-control {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .slider-hd {
    display: flex;
    justify-content: space-between;
    font-size: 11.5px;
    color: var(--dim);
  }
  .mono-val {
    font-family: var(--mono);
    color: var(--text);
  }
  .export-actions {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 6px;
    margin-bottom: 6px;
  }
  .export-btn {
    padding: 8px 10px;
    font-size: 12px;
    font-weight: 600;
    border-radius: var(--radius-sm);
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--text);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    transition: all 0.15s ease;
  }
  .export-btn:hover {
    background: var(--s3);
    border-color: var(--accent);
  }
  .export-btn.primary {
    background: var(--accent);
    color: #fff;
    border-color: var(--accent);
  }
  .export-btn.primary:hover {
    background: var(--accent-press);
  }
  .anim-control {
    margin: 4px 0;
  }
  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: var(--dim);
    cursor: pointer;
  }
  .css-box textarea {
    width: 100%;
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    padding: 8px;
    font-family: var(--mono);
    font-size: 10.5px;
    color: var(--text);
    line-height: 1.45;
    resize: vertical;
  }
  .form-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 10px;
  }
  .form-row label {
    font-size: 12px;
    color: var(--dim);
  }
  .form-row select,
  .form-row input[type="text"],
  .form-row input[type="number"] {
    width: 55%;
    padding: 4px 7px;
    font-size: 12px;
  }
  .form-row.checkbox label {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
  }
  .pos-input {
    width: 100%;
    font-family: var(--mono);
    font-size: 11.5px;
    padding: 4px 6px;
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
  }
</style>
