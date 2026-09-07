// Tiny transient-notice store. Shortcuts and drag-and-drop have no inline label
// to update, so they confirm through here. Not URL state, not persisted.

export interface Toast {
  id: number;
  msg: string;
  kind: "info" | "error";
}

let nextId = 1;

export const toastState = $state<{ items: Toast[] }>({ items: [] });

export function showToast(msg: string, kind: "info" | "error" = "info", ms = 2200): void {
  const id = nextId++;
  toastState.items = [...toastState.items, { id, msg, kind }];
  setTimeout(() => dismissToast(id), ms);
}

export function dismissToast(id: number): void {
  toastState.items = toastState.items.filter((t) => t.id !== id);
}
