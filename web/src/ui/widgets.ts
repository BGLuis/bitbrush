// Small imperative form widgets shared by the hand-built panels (gradient,
// noise field, palette). The descriptor-driven app/ParamControls.svelte covers
// the filter effects; these panels have bespoke bits (stop lists, swatches,
// live CSS) that don't fit that shape.

export function labelledRow(text: string): { row: HTMLElement; slot: HTMLElement } {
  const row = document.createElement("label");
  row.className = "control";
  const span = document.createElement("span");
  span.className = "control-label";
  span.textContent = text;
  const slot = document.createElement("span");
  slot.style.display = "contents";
  row.append(span, slot);
  return { row, slot };
}

export function rowOf(...els: HTMLElement[]): HTMLElement {
  const d = document.createElement("div");
  d.className = "control widget-row";
  d.append(...els);
  return d;
}

export function numberInput(def: number, min?: number, max?: number, step?: number) {
  const el = document.createElement("input");
  el.type = "number";
  if (min !== undefined) el.min = String(min);
  if (max !== undefined) el.max = String(max);
  if (step !== undefined) el.step = String(step);
  el.value = String(def);
  return { el, get: () => (el.value === "" ? def : Number(el.value)) };
}

export function selectInput(options: Array<[string, string]>, def: string) {
  const el = document.createElement("select");
  for (const [value, label] of options) {
    const o = document.createElement("option");
    o.value = value;
    o.textContent = label;
    el.append(o);
  }
  el.value = def;
  return { el, get: () => el.value };
}

export function checkboxInput(def: boolean) {
  const el = document.createElement("input");
  el.type = "checkbox";
  el.checked = def;
  return { el, get: () => el.checked };
}

export function colorInput(def: string) {
  const el = document.createElement("input");
  el.type = "color";
  el.value = def;
  return { el, get: () => el.value };
}

export function textInput(def: string) {
  const el = document.createElement("input");
  el.type = "text";
  el.value = def;
  return { el, get: () => el.value };
}

export function button(label: string, onClick: () => void): HTMLButtonElement {
  const b = document.createElement("button");
  b.type = "button";
  b.textContent = label;
  b.addEventListener("click", onClick);
  return b;
}

/**
 * Render a colour list as clickable swatches (click copies the hex). Returns
 * the container; call `update(colors)` to replace its contents.
 */
export function swatchStrip(): { element: HTMLElement; update(colors: string[]): void } {
  const element = document.createElement("div");
  element.className = "swatches";
  const update = (colors: string[]) => {
    element.replaceChildren();
    for (const hex of colors) {
      const chip = document.createElement("button");
      chip.type = "button";
      chip.className = "swatch";
      chip.style.background = hex;
      chip.title = `${hex} — clique para copiar`;
      const label = document.createElement("span");
      label.textContent = hex;
      chip.append(label);
      chip.addEventListener("click", () => {
        void navigator.clipboard?.writeText(hex);
        chip.classList.add("copied");
        setTimeout(() => chip.classList.remove("copied"), 700);
      });
      element.append(chip);
    }
  };
  return { element, update };
}
