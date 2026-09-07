<script lang="ts">
  import { effects } from "../ui/controls";
  import { generators } from "../ui/generators";
  import { ui, selectEffect, selectPatternTool, selectGenerator } from "./store.svelte";
  import { generatorStore } from "./generator/generator-store.svelte";
</script>

{#snippet ico(name: string)}
  {#if name === "pixelate"}
    <svg viewBox="0 0 24 24"><rect x="3.5" y="3.5" width="7" height="7" /><rect x="13.5" y="3.5" width="7" height="7" /><rect x="3.5" y="13.5" width="7" height="7" /><rect x="13.5" y="13.5" width="7" height="7" /></svg>
  {:else if name === "grayscale"}
    <svg viewBox="0 0 24 24"><rect x="4" y="5" width="16" height="14" rx="1.5" /><path d="M9.3 5v14M14.6 5v14" /><path d="M4 5h5.3v14H4z" fill="currentColor" stroke="none" /></svg>
  {:else if name === "paper"}
    <svg viewBox="0 0 24 24"><path d="M6 3h8l4 4v14H6z" /><path d="M14 3v4h4" /><path d="M9 12h6M9 15h6M9 18h4" /></svg>
  {:else if name === "pencil"}
    <svg viewBox="0 0 24 24"><path d="M4 20l2-6L16 4l4 4L10 18z" /><path d="M14 6l4 4M4 20l6-2" /></svg>
  {:else if name === "sobel"}
    <svg viewBox="0 0 24 24"><path d="M4 20 20 4M4 20V9M4 20h11" /></svg>
  {:else if name === "quantize"}
    <svg viewBox="0 0 24 24"><path d="M4 19h4v-8H4zM10 19h4V6h-4zM16 19h4v-5h-4z" /></svg>
  {:else if name === "dither"}
    <svg viewBox="0 0 24 24"><circle cx="6" cy="6" r="1.3" /><circle cx="12" cy="7" r="1.3" /><circle cx="18" cy="6" r="1.3" /><circle cx="7" cy="13" r="1.3" /><circle cx="14" cy="14" r="1.3" /><circle cx="9" cy="19" r="1.3" /><circle cx="19" cy="18" r="1.3" /></svg>
  {:else if name === "halftone"}
    <svg viewBox="0 0 24 24"><circle cx="6" cy="12" r="1" fill="currentColor" stroke="none" /><circle cx="11" cy="12" r="1.9" fill="currentColor" stroke="none" /><circle cx="17.5" cy="12" r="3.2" fill="currentColor" stroke="none" /></svg>
  {:else if name === "stipple"}
    <svg viewBox="0 0 24 24"><g fill="currentColor" stroke="none"><circle cx="5" cy="6" r="1" /><circle cx="8" cy="10" r="1" /><circle cx="5" cy="13" r="1" /><circle cx="9" cy="15" r="1" /><circle cx="6" cy="19" r="1" /><circle cx="12" cy="7" r="1" /><circle cx="13" cy="13" r="1" /><circle cx="12" cy="19" r="1" /><circle cx="17" cy="10" r="1" /><circle cx="18" cy="17" r="1" /></g></svg>
  {:else if name === "glitch"}
    <svg viewBox="0 0 24 24"><rect x="3" y="6" width="13" height="4" /><rect x="8" y="14" width="13" height="4" /></svg>
  {:else if name === "ascii"}
    <svg viewBox="0 0 24 24"><rect x="4" y="4" width="16" height="16" rx="1.5" /><path d="M9 16l3-8 3 8M10.2 13h3.6" /></svg>
  {:else if name === "gradient"}
    <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="8" /><path d="M12 4a8 8 0 0 1 0 16z" fill="currentColor" stroke="none" /></svg>
  {:else if name === "palette"}
    <svg viewBox="0 0 24 24"><rect x="3.5" y="4" width="4.5" height="16" /><rect x="9.7" y="4" width="4.5" height="16" /><rect x="16" y="4" width="4.5" height="16" /></svg>
  {/if}
{/snippet}

{#snippet patternIco(name: string)}
  {#if name === "contours"}
    <svg viewBox="0 0 24 24"><path d="M3 8c4-3 14-3 18 0M3 13c4-3 14-3 18 0M3 18c4-3 14-3 18 0" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" /></svg>
  {:else if name === "flowfield"}
    <svg viewBox="0 0 24 24"><path d="M3 17c5-1 9-8 18-5M3 12c5-1 7-4 13-3s4 4 2 6M3 7c8 0 11-4 18-1" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" /></svg>
  {:else if name === "chladni"}
    <svg viewBox="0 0 24 24"><path d="M12 3v18M3 12h18M6 6c6 2 6 10 0 12M18 6c-6 2-6 10 0 12" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" /></svg>
  {:else if name === "reactiondiffusion"}
    <svg viewBox="0 0 24 24"><rect x="4" y="4" width="6" height="6" rx="3" fill="none" stroke="currentColor" stroke-width="1.7" /><rect x="13" y="11" width="7" height="9" rx="3.5" fill="none" stroke="currentColor" stroke-width="1.7" /><circle cx="7" cy="17" r="2.8" fill="none" stroke="currentColor" stroke-width="1.7" /><circle cx="16" cy="6" r="2.3" fill="none" stroke="currentColor" stroke-width="1.7" /></svg>
  {:else if name === "truchet"}
    <svg viewBox="0 0 24 24"><path d="M4 12a8 8 0 0 1 8-8M12 20a8 8 0 0 1 8-8" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" /></svg>
  {:else if name === "harmonograph"}
    <svg viewBox="0 0 24 24"><path d="M5 12c0-5 3-8 7-8s7 3 7 8-3 8-7 8-7-3-7-8zm2-3c2-4 6-4 8 0s-2 10-4 10-6-6-4-10z" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round" /></svg>
  {:else if name === "attractor"}
    <svg viewBox="0 0 24 24"><path d="M12 12c-4-6-8-4-8 0s5 6 8 0c3-6 7-4 8 0s-4 6-8 0z" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" /></svg>
  {:else if name === "lsystem"}
    <svg viewBox="0 0 24 24"><path d="M12 21v-8m0 0l-5-4m5 4l5-4m-8-2l-3-3m3 3l3-3m5 2l3-3m-3 3l-3-3" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" /></svg>
  {:else if name === "flame"}
    <svg viewBox="0 0 24 24"><path d="M12 3v3m0 12v3M3 12h3m12 0h3m-3.5-5.5l-2.1 2.1m-8.8 8.8l-2.1 2.1m0-13l2.1 2.1m8.8 8.8l2.1 2.1" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" /></svg>
  {:else}
    <svg viewBox="0 0 24 24"><path d="M2 12c3-4 6-4 9 0s6 4 9 0M2 7c3-4 6-4 9 0s6 4 9 0" fill="none" stroke="currentColor" stroke-width="1.8" /></svg>
  {/if}
{/snippet}

<nav class="rail">
  <h4>Filtros</h4>
  {#each effects as e (e.name)}
    <button
      class="tool"
      class:on={ui.mode === "filter" && ui.effectName === e.name}
      onclick={() => selectEffect(e.name)}
    >
      {@render ico(e.name)}
      <span>{e.title}</span>
    </button>
  {/each}

  <div class="sep"></div>
  <h4>Padrões</h4>
  {#each generators as g (g.name)}
    <button
      class="tool"
      class:on={ui.mode === "pattern" && generatorStore.selectedPattern === g.name}
      onclick={() => selectPatternTool(g.name)}
    >
      {@render patternIco(g.name)}
      <span>{g.title}</span>
    </button>
  {/each}

  <div class="sep"></div>
  <h4>Geradores</h4>
  <button
    class="tool"
    class:on={ui.mode === "generator"}
    onclick={() => selectGenerator("generator")}
  >
    {@render ico("gradient")}
    <span>Gradiente + Ruído</span>
  </button>
  <button
    class="tool"
    class:on={ui.mode === "palette"}
    onclick={() => selectGenerator("palette")}
  >
    {@render ico("palette")}
    <span>Paleta</span>
  </button>
</nav>

<style>
  .rail {
    background: var(--s1);
    border-right: 1px solid var(--line);
    padding: 14px 12px;
    display: flex;
    flex-direction: column;
    gap: 1px;
    overflow-y: auto;
  }
  h4 {
    margin: 8px 8px 7px;
    font-size: 10.5px;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--mute);
    font-weight: 600;
  }
  .tool {
    display: flex;
    align-items: center;
    gap: 11px;
    padding: 8px 9px;
    border: 0;
    border-radius: 7px;
    background: transparent;
    color: var(--dim);
    font-weight: 500;
    font-size: 13px;
    text-align: left;
    cursor: pointer;
  }
  .tool:hover {
    background: var(--s2);
    color: var(--text);
  }
  .tool.on {
    background: var(--tint);
    color: #fff;
    box-shadow: inset 2px 0 0 var(--accent);
  }
  .tool :global(svg) {
    width: 16px;
    height: 16px;
    flex: none;
    stroke: currentColor;
    fill: none;
    stroke-width: 1.6;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .sep {
    height: 1px;
    background: var(--line-soft);
    margin: 12px 8px;
  }
</style>
