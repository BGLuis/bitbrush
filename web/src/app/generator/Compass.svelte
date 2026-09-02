<script lang="ts">
  import { generatorStore } from "./generator-store.svelte";
  import { COMPASS_ANGLES } from "./noisefield-data";
</script>

<div class="compass-wrap" class:disabled={!generatorStore.isAngled}>
  <div class="compass-grid">
    {#each COMPASS_ANGLES as deg}
      {#if deg === null}
        <div class="center-deg" title="Ângulo atual">
          {generatorStore.direction}°
        </div>
      {:else}
        <button
          type="button"
          class="dir-btn"
          class:active={generatorStore.direction === deg}
          disabled={!generatorStore.isAngled}
          title={`${deg}°`}
          onclick={() => generatorStore.setDirection(deg)}
        >
          <svg
            viewBox="0 0 24 24"
            style="transform: rotate({deg}deg)"
            aria-hidden="true"
          >
            <path
              d="M12 19V5M12 5l-5 5M12 5l5 5"
              fill="none"
              stroke="currentColor"
              stroke-width="2.2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      {/if}
    {/each}
  </div>
  <div class="slider-row">
    <input
      type="range"
      min="0"
      max="360"
      step="1"
      disabled={!generatorStore.isAngled}
      value={generatorStore.direction}
      oninput={(e) => generatorStore.setDirection(Number((e.target as HTMLInputElement).value))}
    />
    <span class="deg-val">{generatorStore.direction}°</span>
  </div>
</div>

<style>
  .compass-wrap {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .compass-wrap.disabled {
    opacity: 0.4;
    pointer-events: none;
  }
  .compass-grid {
    display: grid;
    grid-template-columns: repeat(3, 34px);
    gap: 5px;
    justify-content: center;
    margin: 2px auto;
  }
  .dir-btn {
    width: 34px;
    height: 34px;
    padding: 0;
    border-radius: 50%;
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--dim);
    display: grid;
    place-items: center;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .dir-btn:hover:not(:disabled) {
    background: var(--s3);
    color: var(--text);
    border-color: var(--accent);
  }
  .dir-btn.active {
    background: var(--accent);
    color: #fff;
    border-color: var(--accent);
    box-shadow: 0 0 10px rgba(155, 109, 255, 0.4);
  }
  .dir-btn svg {
    width: 15px;
    height: 15px;
  }
  .center-deg {
    display: grid;
    place-items: center;
    font-family: var(--mono);
    font-size: 11px;
    color: var(--mute);
    font-weight: 600;
  }
  .slider-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .slider-row input[type="range"] {
    flex: 1;
  }
  .deg-val {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--dim);
    min-width: 36px;
    text-align: right;
  }
</style>
