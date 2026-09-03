<script lang="ts">
  import { generatorStore } from "./generator-store.svelte";
  import PatternControls from "./PatternControls.svelte";

  let showPresets = $state(true);
</script>

<div class="pattern-panel">
  <!-- Presets for current pattern -->
  {#if generatorStore.currentPatternPresets.length > 0}
    <section class="section">
      <div class="section-hd">
        <span class="sec-title">Presets ({generatorStore.currentPatternPresets.length})</span>
        <button
          type="button"
          class="toggle-btn"
          onclick={() => (showPresets = !showPresets)}
        >
          {showPresets ? "Ocultar" : "Mostrar"}
        </button>
      </div>
      {#if showPresets}
        <div class="pattern-presets-list">
          {#each generatorStore.currentPatternPresets as pre (pre.id)}
            <button
              type="button"
              class="pattern-preset-item"
              onclick={() => generatorStore.applyPatternPreset(pre)}
            >
              <span class="pre-name">{pre.name}</span>
              <span class="pre-desc">{pre.description}</span>
            </button>
          {/each}
        </div>
      {/if}
    </section>
  {/if}

  <!-- Quick action bar: Animate & Roll Seed -->
  <section class="section quick-actions-sec">
    <div class="quick-row">
      {#if generatorStore.currentPatternParams && "seed" in generatorStore.currentPatternParams}
        <button
          type="button"
          class="quick-btn"
          onclick={() => generatorStore.rollPatternSeed()}
          title="Sortear nova semente aleatória"
        >
          🎲 Nova Semente
        </button>
      {/if}

      {#if generatorStore.patternHasTime}
        <button
          type="button"
          class="quick-btn"
          class:active={generatorStore.animate}
          onclick={() => generatorStore.toggleAnimation()}
          title="Iniciar ou pausar animação contínua ao vivo [Espaço]"
        >
          {generatorStore.animate ? "⏸ Pausar" : "▶ Animar ao Vivo"}
        </button>
      {/if}
    </div>
  </section>

  <!-- Dynamic Parameter Controls -->
  <section class="section">
    <div class="sec-title">Controles do Padrão</div>
    <PatternControls />
  </section>

  <!-- Export Section -->
  <section class="section export-sec">
    <div class="sec-title">Exportação</div>
    <div class="export-actions">
      <button
        type="button"
        class="export-btn primary"
        onclick={() => generatorStore.exportHighResPNG()}
      >
        Baixar PNG (1600px)
      </button>
    </div>
  </section>
</div>

<style>
  .pattern-panel {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .section-hd {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .sec-title {
    font-size: 11px;
    letter-spacing: 0.8px;
    text-transform: uppercase;
    color: var(--dim);
    font-weight: 600;
  }
  .toggle-btn {
    background: transparent;
    border: none;
    font-size: 11px;
    color: var(--accent);
    cursor: pointer;
    padding: 0;
  }
  .toggle-btn:hover {
    text-decoration: underline;
  }
  .pattern-presets-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .pattern-preset-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    padding: 8px 10px;
    cursor: pointer;
    text-align: left;
    transition: all 0.15s ease;
  }
  .pattern-preset-item:hover {
    background: var(--s3);
    border-color: var(--dim);
  }
  .pattern-preset-item .pre-name {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--txt);
  }
  .pattern-preset-item .pre-desc {
    font-size: 11px;
    color: var(--dim);
    line-height: 1.35;
  }
  .quick-actions-sec {
    margin-top: -4px;
  }
  .quick-row {
    display: flex;
    gap: 8px;
  }
  .quick-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    padding: 8px 10px;
    color: var(--txt);
    font-size: var(--text-xs);
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .quick-btn:hover {
    background: var(--s3);
    border-color: var(--dim);
  }
  .quick-btn.active {
    background: var(--accent);
    color: #fff;
    border-color: var(--accent);
  }
  .export-sec {
    margin-top: 6px;
    padding-top: 12px;
    border-top: 1px solid var(--line);
  }
  .export-actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .export-btn {
    width: 100%;
    padding: 9px 12px;
    border-radius: var(--radius-sm);
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--txt);
  }
  .export-btn.primary {
    background: var(--accent);
    color: #fff;
    border-color: var(--accent);
  }
  .export-btn.primary:hover {
    filter: brightness(1.1);
  }
</style>
