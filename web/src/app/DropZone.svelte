<script lang="ts">
  import { onMount } from "svelte";
  import { loadFile } from "./lib/image";
  import { showToast } from "./lib/toast.svelte";

  // Drop an image anywhere on the window, or paste one from the clipboard.
  // `depth` counts dragenter/dragleave so the overlay doesn't flicker as the
  // pointer crosses child elements.
  let depth = $state(0);
  const active = $derived(depth > 0);

  function hasFiles(e: DragEvent): boolean {
    return Array.from(e.dataTransfer?.types ?? []).includes("Files");
  }

  function onDragEnter(e: DragEvent) {
    if (!hasFiles(e)) return;
    e.preventDefault();
    depth += 1;
  }

  function onDragOver(e: DragEvent) {
    if (!hasFiles(e)) return;
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = "copy";
    if (depth === 0) depth = 1; // some browsers skip dragenter
  }

  function onDragLeave(e: DragEvent) {
    if (!hasFiles(e)) return;
    // relatedTarget === null means the pointer left the window entirely — some
    // browsers won't balance the counter in that case, so snap it shut.
    depth = e.relatedTarget === null ? 0 : Math.max(0, depth - 1);
  }

  function onDrop(e: DragEvent) {
    depth = 0;
    if (!hasFiles(e)) return;
    e.preventDefault();
    const file = Array.from(e.dataTransfer?.files ?? []).find((f) =>
      f.type.startsWith("image/"),
    );
    if (file) void loadFile(file);
    else showToast("Nenhuma imagem no que foi solto", "error");
  }

  function onPaste(e: ClipboardEvent) {
    const target = e.target as HTMLElement | null;
    const tag = (target?.tagName ?? "").toLowerCase();
    if (tag === "input" || tag === "textarea" || target?.isContentEditable) return;
    const file = Array.from(e.clipboardData?.items ?? [])
      .find((it) => it.type.startsWith("image/"))
      ?.getAsFile();
    if (file) {
      e.preventDefault();
      void loadFile(file);
    }
  }

  onMount(() => {
    window.addEventListener("dragenter", onDragEnter);
    window.addEventListener("dragover", onDragOver);
    window.addEventListener("dragleave", onDragLeave);
    window.addEventListener("drop", onDrop);
    window.addEventListener("paste", onPaste);
    return () => {
      window.removeEventListener("dragenter", onDragEnter);
      window.removeEventListener("dragover", onDragOver);
      window.removeEventListener("dragleave", onDragLeave);
      window.removeEventListener("drop", onDrop);
      window.removeEventListener("paste", onPaste);
    };
  });
</script>

{#if active}
  <div class="dropz" aria-hidden="true">
    <svg class="ring" viewBox="0 0 100 100" preserveAspectRatio="none">
      <rect x="1.5" y="1.5" width="97" height="97" rx="3" />
    </svg>
    <div class="card">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 16V4M7 9l5-5 5 5" />
        <path d="M4 15v3a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-3" />
      </svg>
      <span class="t">Solte a imagem para carregar</span>
      <span class="s">substitui a imagem atual</span>
    </div>
  </div>
{/if}

<style>
  .dropz {
    position: fixed;
    inset: 0;
    z-index: 150;
    pointer-events: none;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(12, 10, 16, 0.62);
    backdrop-filter: blur(2px);
    animation: fade 0.12s ease;
  }
  .ring {
    position: absolute;
    inset: 10px;
    width: calc(100% - 20px);
    height: calc(100% - 20px);
  }
  .ring rect {
    fill: none;
    stroke: var(--accent);
    stroke-width: 1;
    stroke-dasharray: 4 3;
    animation: march 0.6s linear infinite;
    vector-effect: non-scaling-stroke;
  }
  .card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 26px 40px;
    border-radius: var(--radius);
    background: var(--s1);
    border: 1px solid var(--accent);
    box-shadow: 0 24px 70px rgba(0, 0, 0, 0.5);
    color: var(--text);
  }
  .card svg {
    width: 34px;
    height: 34px;
    color: var(--accent);
  }
  .card .t {
    font-family: var(--disp);
    font-weight: 600;
    font-size: 15px;
  }
  .card .s {
    font-size: 12px;
    color: var(--mute);
  }
  @keyframes march {
    to {
      stroke-dashoffset: -7;
    }
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
</style>
