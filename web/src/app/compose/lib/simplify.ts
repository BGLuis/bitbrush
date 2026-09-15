// Douglas-Peucker polyline simplification, used to bound a freehand lasso's
// point count before it's committed to a mask (and, from there, a shareable
// URL) — a raw pointermove capture can easily produce hundreds of points.

export interface Pt {
  x: number;
  y: number;
}

function distToSegment(p: Pt, a: Pt, b: Pt): number {
  const dx = b.x - a.x;
  const dy = b.y - a.y;
  const lenSq = dx * dx + dy * dy;
  if (lenSq === 0) return Math.hypot(p.x - a.x, p.y - a.y);
  const t = Math.max(0, Math.min(1, ((p.x - a.x) * dx + (p.y - a.y) * dy) / lenSq));
  return Math.hypot(p.x - (a.x + t * dx), p.y - (a.y + t * dy));
}

function rdp(points: Pt[], epsilon: number): Pt[] {
  if (points.length < 3) return points;
  let maxDist = 0;
  let idx = 0;
  const first = points[0];
  const last = points[points.length - 1];
  for (let i = 1; i < points.length - 1; i++) {
    const d = distToSegment(points[i], first, last);
    if (d > maxDist) {
      maxDist = d;
      idx = i;
    }
  }
  if (maxDist > epsilon) {
    const left = rdp(points.slice(0, idx + 1), epsilon);
    const right = rdp(points.slice(idx), epsilon);
    return left.slice(0, -1).concat(right);
  }
  return [first, last];
}

/** Simplifies a polyline, then re-runs with a larger epsilon (up to a few
 *  tries) until it's at or under maxPoints — bounds the worst case
 *  deterministically for a URL-shareable mask. */
export function simplifyPolygon(points: Pt[], startEpsilon: number, maxPoints = 100): Pt[] {
  let epsilon = startEpsilon;
  let out = rdp(points, epsilon);
  let tries = 0;
  while (out.length > maxPoints && tries < 12) {
    epsilon *= 1.5;
    out = rdp(points, epsilon);
    tries++;
  }
  return out;
}
