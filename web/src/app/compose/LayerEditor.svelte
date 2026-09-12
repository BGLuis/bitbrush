<script lang="ts">
  import {
    composeStore,
    FIT_MODES,
    SOURCE_LABELS,
    generatorControls,
    type LayerUI,
  } from "./compose-store.svelte";
  import { generators } from "../../ui/generators";
  import ParamForm from "../ParamForm.svelte";
  import ChainEditor from "./ChainEditor.svelte";
  import type { LayerSource } from "../../wasm";

  let { layer, index }: { layer: LayerUI; index: number } = $props();

  const SOURCES: LayerSource[] = ["base", "image", "generator", "gradient", "noisefield", "text"];

  const NOISE_FIELDS = [
    "linear",
    "radial",
    "conic",
    "reflected",
    "diamond",
    "mesh",
    "freeform",
    "flow",
  ];
  const NOISE_STYLES = [
    "metallic",
    "chrome",
    "iridescent",
    "holographic",
    "neon",
    "pastel",
    "duotone",
    "rainbow",
  ];
  const NOISE_TEXTURES = ["smooth", "grain", "frosted", "wave", "wrinkle", "paper"];
  const GRAD_SPACES = ["srgb", "linear", "hsl", "lab", "lch", "oklab", "oklch"];

  let fileInput: HTMLInputElement | undefined = $state();

  function pickFile() {
    fileInput?.click();
  }
  function onFile(e: Event) {
    const f = (e.target as HTMLInputElement).files?.[0];
    if (f) void composeStore.addImageForSelectedLayer(f);
    (e.target as HTMLInputElement).value = "";
  }

  // Nested genParams edits (gradient stops, noise fields) mutate the layer
  // object in place, then ask the store to repaint.
  function touch() {
    composeStore.scheduleRender();
  }
</script>

<div class="editor">
  <div class="hdr">Camada · {SOURCE_LABELS[layer.source]}</div>

  <label class="fld">
    <span>Fonte</span>
    <select
      value={layer.source}
      onchange={(e) => composeStore.setSource(index, e.currentTarget.value as LayerSource)}
    >
      {#each SOURCES as s (s)}
        <option value={s}>{SOURCE_LABELS[s]}</option>
      {/each}
    </select>
  </label>

  {#if layer.source === "generator"}
    <label class="fld">
      <span>Gerador</span>
      <select
        value={layer.generator}
        onchange={(e) => composeStore.setGenerator(index, e.currentTarget.value)}
      >
        {#each generators as g (g.name)}
          <option value={g.name}>{g.title}</option>
        {/each}
      </select>
    </label>
  {/if}

  {#if layer.source !== "base"}
    <label class="fld">
      <span>Encaixe</span>
      <select
        value={layer.fit}
        onchange={(e) => composeStore.setFit(index, e.currentTarget.value as (typeof FIT_MODES)[number])}
      >
        {#each FIT_MODES as f (f)}
          <option value={f}>{f}</option>
        {/each}
      </select>
    </label>
  {/if}

  {#if layer.source === "image"}
    <div class="imgpick">
      <button type="button" onclick={pickFile}>Carregar imagem…</button>
      <input
        bind:this={fileInput}
        type="file"
        accept="image/*"
        hidden
        onchange={onFile}
      />
      {#if composeStore.slots.length}
        <div class="slots">
          {#each composeStore.slots as s (s.id)}
            <button
              type="button"
              class="slot"
              class:on={layer.slotId === s.id}
              title={s.name}
              onclick={() => composeStore.bindSlot(index, s.id)}
            >
              {s.name}
            </button>
          {/each}
        </div>
      {/if}
      {#if !layer.slotId}
        <p class="hint">Solte uma imagem nesta camada, ou use “Carregar imagem…”.</p>
      {/if}
    </div>
  {:else if layer.source === "generator"}
    <ParamForm
      controls={generatorControls(layer.generator)}
      params={layer.genParams}
      onchange={() => composeStore.scheduleRender()}
    />
  {:else if layer.source === "gradient"}
    <div class="mini">
      <label class="fld">
        <span>Cor inicial</span>
        <input
          type="color"
          value={layer.genParams.stops?.[0]?.color ?? "#0ea5e9"}
          oninput={(e) => {
            layer.genParams.stops[0].color = e.currentTarget.value;
            touch();
          }}
        />
      </label>
      <label class="fld">
        <span>Cor final</span>
        <input
          type="color"
          value={layer.genParams.stops?.[1]?.color ?? "#8b5cf6"}
          oninput={(e) => {
            layer.genParams.stops[1].color = e.currentTarget.value;
            touch();
          }}
        />
      </label>
      <label class="fld">
        <span>Espaço</span>
        <select
          value={layer.genParams.space}
          onchange={(e) => {
            layer.genParams.space = e.currentTarget.value;
            touch();
          }}
        >
          {#each GRAD_SPACES as s (s)}
            <option value={s}>{s}</option>
          {/each}
        </select>
      </label>
      <label class="fld">
        <span>Ângulo · {Math.round(layer.genParams.angle ?? 90)}°</span>
        <input
          type="range"
          min="0"
          max="360"
          step="1"
          value={layer.genParams.angle ?? 90}
          oninput={(e) => {
            layer.genParams.angle = Number(e.currentTarget.value);
            touch();
          }}
        />
      </label>
    </div>
  {:else if layer.source === "noisefield"}
    <div class="mini">
      <label class="fld">
        <span>Campo</span>
        <select
          value={layer.genParams.field}
          onchange={(e) => {
            layer.genParams.field = e.currentTarget.value;
            touch();
          }}
        >
          {#each NOISE_FIELDS as f (f)}<option value={f}>{f}</option>{/each}
        </select>
      </label>
      <label class="fld">
        <span>Estilo</span>
        <select
          value={layer.genParams.style}
          onchange={(e) => {
            layer.genParams.style = e.currentTarget.value;
            touch();
          }}
        >
          {#each NOISE_STYLES as s (s)}<option value={s}>{s}</option>{/each}
        </select>
      </label>
      <label class="fld">
        <span>Textura</span>
        <select
          value={layer.genParams.texture}
          onchange={(e) => {
            layer.genParams.texture = e.currentTarget.value;
            touch();
          }}
        >
          {#each NOISE_TEXTURES as t (t)}<option value={t}>{t}</option>{/each}
        </select>
      </label>
      <label class="fld">
        <span>Cor A</span>
        <input
          type="color"
          value={layer.genParams.stops?.[0] ?? "#101828"}
          oninput={(e) => {
            layer.genParams.stops[0] = e.currentTarget.value;
            touch();
          }}
        />
      </label>
      <label class="fld">
        <span>Cor B</span>
        <input
          type="color"
          value={layer.genParams.stops?.[1] ?? "#dc78a0"}
          oninput={(e) => {
            layer.genParams.stops[1] = e.currentTarget.value;
            touch();
          }}
        />
      </label>
      <label class="fld">
        <span>Semente</span>
        <input
          type="number"
          value={layer.genParams.seed ?? 0}
          oninput={(e) => {
            layer.genParams.seed = Number(e.currentTarget.value);
            touch();
          }}
        />
      </label>
      <label class="fld">
        <span>Escala · {Math.round(layer.genParams.scale ?? 50)}</span>
        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value={layer.genParams.scale ?? 50}
          oninput={(e) => {
            layer.genParams.scale = Number(e.currentTarget.value);
            touch();
          }}
        />
      </label>
      <label class="fld">
        <span>Distorção · {Math.round(layer.genParams.distortion ?? 55)}</span>
        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value={layer.genParams.distortion ?? 55}
          oninput={(e) => {
            layer.genParams.distortion = Number(e.currentTarget.value);
            touch();
          }}
        />
      </label>
      <label class="fld">
        <span>Ângulo · {Math.round(layer.genParams.angle ?? 0)}°</span>
        <input
          type="range"
          min="0"
          max="360"
          step="1"
          value={layer.genParams.angle ?? 0}
          oninput={(e) => {
            layer.genParams.angle = Number(e.currentTarget.value);
            touch();
          }}
        />
      </label>
    </div>
  {:else if layer.source === "text"}
    <div class="mini">
      <label class="fld" style="grid-column: 1 / -1">
        <span>Texto</span>
        <textarea
          rows="3"
          class="txt-area"
          value={layer.genParams.content ?? "BitBrush"}
          oninput={(e) => {
            layer.genParams.content = (e.currentTarget as HTMLTextAreaElement).value;
            touch();
          }}
        ></textarea>
      </label>
      <label class="fld">
        <span>Fonte</span>
        <select
          value={layer.genParams.fontFamily ?? "go"}
          onchange={(e) => {
            layer.genParams.fontFamily = e.currentTarget.value;
            touch();
          }}
        >
          <option value="go">Go Sans</option>
          <option value="gomono">Go Mono</option>
        </select>
      </label>
      <label class="fld">
        <span>Tamanho (pt) · {layer.genParams.size ?? 72}</span>
        <input
          type="range"
          min="12"
          max="240"
          step="2"
          value={layer.genParams.size ?? 72}
          oninput={(e) => {
            layer.genParams.size = Number(e.currentTarget.value);
            touch();
          }}
        />
      </label>
      <label class="fld">
        <span>Cor</span>
        <input
          type="color"
          value={layer.genParams.color ?? "#ffffff"}
          oninput={(e) => {
            layer.genParams.color = e.currentTarget.value;
            touch();
          }}
        />
      </label>
      <label class="fld">
        <span>Alinhamento</span>
        <select
          value={layer.genParams.align ?? "center"}
          onchange={(e) => {
            layer.genParams.align = e.currentTarget.value;
            touch();
          }}
        >
          <option value="left">Esquerda</option>
          <option value="center">Centro</option>
          <option value="right">Direita</option>
        </select>
      </label>
      <label class="fld" style="grid-column: 1 / -1">
        <span>Posição X · {Math.round((layer.genParams.x ?? 0.5) * 100)}%</span>
        <input
          type="range"
          min="0"
          max="1"
          step="0.01"
          value={layer.genParams.x ?? 0.5}
          oninput={(e) => {
            layer.genParams.x = Number(e.currentTarget.value);
            touch();
          }}
        />
      </label>
      <label class="fld" style="grid-column: 1 / -1">
        <span>Posição Y · {Math.round((layer.genParams.y ?? 0.5) * 100)}%</span>
        <input
          type="range"
          min="0"
          max="1"
          step="0.01"
          value={layer.genParams.y ?? 0.5}
          oninput={(e) => {
            layer.genParams.y = Number(e.currentTarget.value);
            touch();
          }}
        />
      </label>
    </div>
  {/if}

  <ChainEditor {layer} {index} />
</div>

<style>
  .editor {
    display: flex;
    flex-direction: column;
    gap: 11px;
    border-top: 1px solid var(--line);
    padding-top: 12px;
  }
  .hdr {
    font-family: var(--disp);
    font-weight: 600;
    font-size: 12.5px;
  }
  .fld {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 12px;
    color: var(--dim);
  }
  .fld select,
  .fld input[type="number"] {
    width: 100%;
  }
  .mini {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 9px;
  }
  .imgpick {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .imgpick > button {
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--text);
    padding: 7px;
    border-radius: var(--radius-sm);
    font-size: 12px;
    cursor: pointer;
  }
  .slots {
    display: flex;
    flex-wrap: wrap;
    gap: 5px;
  }
  .slot {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--dim);
    padding: 4px 8px;
    border-radius: var(--radius-sm);
    font-size: 11px;
    cursor: pointer;
  }
  .slot.on {
    border-color: var(--accent);
    color: var(--text);
  }
  .hint {
    margin: 0;
    font-size: 11px;
    color: var(--mute);
  }
  .txt-area {
    width: 100%;
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    color: var(--text);
    padding: 6px 8px;
    font-size: 12px;
    resize: vertical;
    font-family: inherit;
  }
</style>
