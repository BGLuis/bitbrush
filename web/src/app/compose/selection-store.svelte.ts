// Transient "drawing a selection on the canvas" state for the "Compor" mode.
// Deliberately a store OF ITS OWN, not fields on ComposeStore: it holds no
// layer/stage identity at all — just the active tool and whichever
// MaskEditor's `onchange` callback armed it — so it never needs to be swept
// into ComposeStore's undo/redo snapshotting (#noteEdit() only ever diffs
// `layers`/`selected`/`ratio`; a sibling object is structurally invisible to
// it). The in-progress drag geometry itself lives in SelectionOverlay's own
// local state, never here and never in a layer/stage's mask — only the
// finished MaskParams crosses over, once, on commit.

import type { MaskParams } from "../../wasm";

export type SelectionTool = "rect" | "ellipse" | "polygon";

class SelectionStore {
  active = $state(false);
  tool = $state<SelectionTool>("rect");
  // Bumped on every arm() so a MaskEditor instance can tell whether IT is
  // the one currently being drawn (more than one can be open at once — a
  // layer mask and a chain-stage mask on the same layer, for instance).
  token = $state(0);
  #onCommit: ((mask: MaskParams) => void) | null = null;

  /** Called by a MaskEditor's "Desenhar no canvas" button. Returns the new
   *  token so the caller can compare it against `selectionStore.token`. */
  arm(tool: SelectionTool, onCommit: (mask: MaskParams) => void): number {
    this.tool = tool;
    this.#onCommit = onCommit;
    this.token++;
    this.active = true;
    return this.token;
  }

  cancel(): void {
    this.active = false;
    this.#onCommit = null;
  }

  commit(mask: MaskParams): void {
    this.#onCommit?.(mask);
    this.active = false;
    this.#onCommit = null;
  }
}

export const selectionStore = new SelectionStore();
