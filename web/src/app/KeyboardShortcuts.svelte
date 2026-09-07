<script lang="ts">
  import { onMount } from "svelte";
  import { ui, resetParams } from "./store.svelte";
  import { generatorStore } from "./generator/generator-store.svelte";
  import {
    openImage,
    exportPNG,
    copyShareLink,
    cycleEffect,
  } from "./lib/actions";
  import { nudgeZoom, zoomFit, zoomActual } from "./lib/view.svelte";
  import { showToast } from "./lib/toast.svelte";

  let helpOpen = $state(false);

  const IS_MAC = typeof navigator !== "undefined" && /mac/i.test(navigator.platform);
  const MOD = IS_MAC ? "⌘" : "Ctrl";

  const GROUPS = [
    {
      title: "Arquivo",
      rows: [
        { keys: [MOD, "O"], label: "Carregar imagem" },
        { keys: [MOD, "V"], label: "Colar imagem da área de transferência" },
        { keys: [MOD, "S"], label: "Exportar PNG" },
        { keys: [MOD, "⇧", "C"], label: "Copiar link da receita" },
      ],
    },
    {
      title: "Edição",
      rows: [
        { keys: ["Alt", "R"], label: "Restaurar parâmetros" },
        { keys: ["Alt", "↑"], label: "Filtro anterior" },
        { keys: ["Alt", "↓"], label: "Próximo filtro" },
        { keys: ["Espaço"], label: "Animar / pausar (geradores)" },
      ],
    },
    {
      title: "Visualização",
      rows: [
        { keys: [MOD, "+"], label: "Mais zoom" },
        { keys: [MOD, "−"], label: "Menos zoom" },
        { keys: [MOD, "0"], label: "Ajustar à janela" },
        { keys: [MOD, "1"], label: "Zoom 100%" },
        { keys: [MOD, "roda"], label: "Zoom com a roda do mouse" },
      ],
    },
    {
      title: "Ajuda",
      rows: [{ keys: ["?"], label: "Mostrar / ocultar esta lista" }],
    },
  ];

  function isTypingTarget(el: Element | null): boolean {
    const tag = (el?.tagName ?? "").toLowerCase();
    return (
      tag === "input" ||
      tag === "textarea" ||
      tag === "select" ||
      (el as HTMLElement | null)?.isContentEditable === true
    );
  }

  // Mirrors Stage.svelte's canAnimate.
  function canAnimate(): boolean {
    const g = generatorStore;
    return (
      (ui.mode === "generator" && g.generatorMode === "noise" && g.isOrganic) ||
      ((ui.mode === "generator" || ui.mode === "pattern") &&
        g.generatorMode === "patterns" &&
        g.patternHasTime)
    );
  }

  function onKeydown(e: KeyboardEvent) {
    const mod = e.ctrlKey || e.metaKey;
    const typing = isTypingTarget(document.activeElement);

    if (e.key === "Escape" && helpOpen) {
      helpOpen = false;
      return;
    }
    if (!mod && !e.altKey && e.key === "?") {
      if (typing) return;
      e.preventDefault();
      helpOpen = !helpOpen;
      return;
    }

    if (e.code === "Space" && !mod && !e.altKey && !e.shiftKey) {
      if (typing || !canAnimate()) return;
      e.preventDefault();
      generatorStore.toggleAnimation();
      return;
    }

    if (mod && !e.altKey && !e.shiftKey && (e.key === "o" || e.key === "O")) {
      e.preventDefault();
      openImage();
      return;
    }
    if (mod && !e.altKey && !e.shiftKey && (e.key === "s" || e.key === "S")) {
      e.preventDefault();
      void exportPNG((m) => m && showToast(m));
      return;
    }
    if (mod && e.shiftKey && (e.key === "c" || e.key === "C")) {
      e.preventDefault();
      copyShareLink((m) => m && showToast(m));
      return;
    }
    if (e.altKey && !mod && (e.key === "r" || e.key === "R")) {
      if (ui.mode !== "filter") return;
      e.preventDefault();
      resetParams();
      showToast("Parâmetros restaurados");
      return;
    }
    if (mod && !e.altKey && !typing && ui.mode === "filter") {
      if (e.key === "=" || e.key === "+") {
        e.preventDefault();
        nudgeZoom(1);
        return;
      }
      if (e.key === "-" || e.key === "_") {
        e.preventDefault();
        nudgeZoom(-1);
        return;
      }
      if (e.key === "0") {
        e.preventDefault();
        zoomFit();
        return;
      }
      if (e.key === "1") {
        e.preventDefault();
        zoomActual();
        return;
      }
    }
    if (e.altKey && !mod && (e.key === "ArrowUp" || e.key === "ArrowDown")) {
      if (typing) return;
      e.preventDefault();
      cycleEffect(e.key === "ArrowUp" ? -1 : 1);
      return;
    }
  }

  onMount(() => {
    window.addEventListener("keydown", onKeydown);
    return () => window.removeEventListener("keydown", onKeydown);
  });
</script>

{#if helpOpen}
  <!-- Backdrop click closes; keyboard users close with Esc (handled globally). -->
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="scrim" onclick={(e) => e.target === e.currentTarget && (helpOpen = false)}>
    <div class="sheet" role="dialog" aria-modal="true" aria-label="Atalhos de teclado" tabindex="-1">
      <div class="sheet-hd">
        <h3>Atalhos de teclado</h3>
        <button class="x" aria-label="Fechar" onclick={() => (helpOpen = false)}>✕</button>
      </div>
      {#each GROUPS as g (g.title)}
        <div class="grp">
          <div class="grp-t">{g.title}</div>
          {#each g.rows as r (r.label)}
            <div class="row">
              <span class="lbl">{r.label}</span>
              <span class="keys">
                {#each r.keys as k (k)}<kbd>{k}</kbd>{/each}
              </span>
            </div>
          {/each}
        </div>
      {/each}
    </div>
  </div>
{/if}

<style>
  .scrim {
    position: fixed;
    inset: 0;
    z-index: 200;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(10, 8, 14, 0.6);
    backdrop-filter: blur(2px);
    animation: fade 0.12s ease;
  }
  .sheet {
    width: 420px;
    max-width: calc(100vw - 32px);
    max-height: calc(100vh - 64px);
    overflow-y: auto;
    background: var(--s1);
    border: 1px solid var(--line);
    border-radius: var(--radius);
    box-shadow: 0 30px 80px rgba(0, 0, 0, 0.55);
    padding: 16px 18px 18px;
  }
  .sheet-hd {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }
  .sheet-hd h3 {
    margin: 0;
    font-family: var(--disp);
    font-weight: 600;
    font-size: 14px;
  }
  .x {
    border: 0;
    background: transparent;
    color: var(--dim);
    font-size: 13px;
    cursor: pointer;
  }
  .x:hover {
    color: var(--text);
  }
  .grp + .grp {
    margin-top: 14px;
  }
  .grp-t {
    font-size: 10px;
    letter-spacing: 1.4px;
    text-transform: uppercase;
    color: var(--mute);
    font-weight: 600;
    margin-bottom: 6px;
  }
  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 5px 0;
  }
  .lbl {
    font-size: 12.5px;
    color: var(--dim);
  }
  .keys {
    display: flex;
    gap: 4px;
    flex: none;
  }
  kbd {
    font-family: var(--mono);
    font-size: 10.5px;
    line-height: 1;
    padding: 4px 6px;
    background: var(--s2);
    border: 1px solid var(--line);
    border-bottom-width: 2px;
    border-radius: 5px;
    color: var(--text);
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
</style>
