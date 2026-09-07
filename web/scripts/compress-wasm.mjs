// Precompress the large build artifacts (mainly main.wasm, ~5.7 MB raw) into
// brotli and gzip siblings so a correctly configured static host can serve
// them without compressing on every request. Zero dependencies — Node's
// built-in zlib does both. Hosts that already compress on the edge (Netlify,
// Cloudflare Pages) ignore these harmlessly; see web/public/_headers.

import { readdirSync, readFileSync, writeFileSync, statSync } from "node:fs";
import { join } from "node:path";
import { brotliCompressSync, gzipSync, constants } from "node:zlib";

const DIST = join(import.meta.dirname, "..", "dist");
const EXT = /\.(wasm|js|css|html|json|svg)$/;
const MIN_BYTES = 1024;

/** @param {string} dir */
function walk(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(p);
    } else if (EXT.test(entry.name) && statSync(p).size >= MIN_BYTES) {
      compress(p);
    }
  }
}

/** @param {string} file */
function compress(file) {
  const raw = readFileSync(file);
  const br = brotliCompressSync(raw, {
    params: {
      [constants.BROTLI_PARAM_QUALITY]: 11,
      [constants.BROTLI_PARAM_SIZE_HINT]: raw.length,
    },
  });
  const gz = gzipSync(raw, { level: 9 });
  writeFileSync(file + ".br", br);
  writeFileSync(file + ".gz", gz);
  const pct = (n) => ((100 * n) / raw.length).toFixed(0) + "%";
  console.log(
    `  ${file.replace(DIST + "/", "")}  ${(raw.length / 1024) | 0}K → br ${pct(br.length)} / gz ${pct(gz.length)}`,
  );
}

console.log("precompressing dist/…");
walk(DIST);
