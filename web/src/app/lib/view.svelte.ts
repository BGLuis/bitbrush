// Canvas zoom / pan view state for filter mode. Per-viewer view preference —
// not URL state, not persisted. "fit" scales the canvas down to the viewport;
// any explicit scale lets the viewport scroll so big images can be inspected
// 1:1 or larger.

const STEPS = [0.1, 0.15, 0.25, 0.33, 0.5, 0.67, 1, 1.5, 2, 3, 4, 6, 8, 12, 16];
export const MIN_ZOOM = STEPS[0];
export const MAX_ZOOM = STEPS[STEPS.length - 1];

export const viewState = $state<{ fit: boolean; scale: number }>({ fit: true, scale: 1 });

export function zoomFit(): void {
  viewState.fit = true;
}

export function zoomActual(): void {
  viewState.fit = false;
  viewState.scale = 1;
}

export function setZoom(scale: number): void {
  viewState.fit = false;
  viewState.scale = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, scale));
}

/** Nudge zoom to the next / previous step. From "fit" we step off 1×. */
export function nudgeZoom(dir: 1 | -1): void {
  const s = viewState.fit ? 1 : viewState.scale;
  const next =
    dir === 1
      ? STEPS.find((v) => v > s + 1e-3)
      : [...STEPS].reverse().find((v) => v < s - 1e-3);
  setZoom(next ?? (dir === 1 ? MAX_ZOOM : MIN_ZOOM));
}
