// Svelte action to host a hand-built imperative panel — one that returns an
// `{ element }` (gif.ts, generators-panel.ts, palette-panel.ts). `make` runs
// once when the node mounts; the element is pulled out on destroy. To force a
// fresh panel (e.g. drop captured GIF keyframes when the effect changes), wrap
// the host node in `{#key ...}`.

export function panel(node: HTMLElement, make: () => { element: HTMLElement }) {
  const inst = make();
  node.appendChild(inst.element);
  return {
    destroy() {
      inst.element.remove();
    },
  };
}
