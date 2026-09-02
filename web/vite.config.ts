import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

// The Go toolchain writes main.wasm and wasm_exec.js into public/, so
// Vite serves them from the site root untouched.
export default defineConfig({
  plugins: [svelte()],
  server: { fs: { strict: true } },
  build: { target: "es2022" },
});
