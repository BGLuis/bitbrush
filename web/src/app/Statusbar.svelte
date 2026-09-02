<script lang="ts">
  import { ui } from "./store.svelte";
  import { getBackend, type BackendKind } from "../backend";

  const BACKEND_LABEL: Record<string, string> = {
    auto: "Automático",
    cpu: "CPU (WASM)",
    gpu: "GPU (WebGL2)",
  };
  const KIND_LABEL: Record<BackendKind, string> = {
    cpu: "main-thread",
    "cpu-worker": "Worker",
    gpu: "WebGL2",
  };

  let resolved = $state<BackendKind | null>(null);
  let copied = $state(false);

  $effect(() => {
    const pref = ui.settings.backend;
    resolved = null;
    void getBackend(pref)
      .then((b) => (resolved = b.kind))
      .catch(() => (resolved = null));
  });

  function copyLink() {
    void navigator.clipboard?.writeText(location.href);
    copied = true;
    setTimeout(() => (copied = false), 1200);
  }
</script>

<footer class="status">
  <span>
    <span class="k">Motor:</span>
    {BACKEND_LABEL[ui.settings.backend] ?? ui.settings.backend}
    {#if resolved}→ {KIND_LABEL[resolved]}{/if}
  </span>
  <span>
    <span class="k">Preview:</span>
    {ui.settings.previewMaxDim === 0 ? "resolução plena" : `${ui.settings.previewMaxDim} px`}
  </span>
  <span class="url">
    <span class="k">Receita:</span>
    {ui.query || "(padrões)"}
    <button onclick={copyLink}>{copied ? "copiado!" : "⧉ copiar link"}</button>
  </span>
</footer>

<style>
  .status {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 0 18px;
    background: var(--s1);
    border-top: 1px solid var(--line);
    font-family: var(--mono);
    font-size: 10.5px;
    color: var(--mute);
  }
  .k {
    color: var(--dim);
  }
  .url {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 8px;
    max-width: 60ch;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  button {
    border: 0;
    background: transparent;
    color: #bfa3ff;
    font: inherit;
    cursor: pointer;
    flex: none;
  }
</style>
