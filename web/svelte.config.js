import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

// Svelte 5. No SvelteKit — this is a plain Vite SPA that mounts one
// component tree into web/index.html. vitePreprocess() lets <script lang="ts">
// blocks use TypeScript.
export default {
  preprocess: vitePreprocess(),
};
