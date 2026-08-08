import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// The playground is a plain SPA embedded in a static Hugo page, so it is built with a relative base
// and never assumes it is served from the site root.
//
// The artifact lives in public/ rather than being imported as an asset: it is 15 MB, there is nothing
// for the bundler to do to it, and Hugo will serve it from static/ the same way.
export default defineConfig({
  base: './',
  plugins: [svelte()],
  server: { port: 5174, open: false },
  build: {
    target: 'es2022',
    assetsDir: 'assets',
    rollupOptions: {
      output: {
        // The entry pair keeps fixed names so a Hugo shortcode can write a <script> and a <link>
        // without reading a manifest. Everything else stays hashed: the lazily-loaded chunks are
        // fetched by the entry itself, which knows their names, and hashing is what lets them be
        // cached hard.
        entryFileNames: 'assets/playground.js',
        // Only the entry's own stylesheet, which Vite names after the html entry. Swagger UI's is a
        // lazy chunk's asset: the entry fetches it by the name in the bundle, so it stays hashed.
        assetFileNames: (info) =>
          info.names?.includes('index.css') ? 'assets/playground.css' : 'assets/[name]-[hash][extname]',
      },
    },
    // Swagger UI is 1.4 MB in one chunk and the warning is right about that in general and wrong
    // here: the chunk is behind a dynamic import and is never fetched unless its tab is opened.
    // Splitting it further would only mean more requests for the reader who does open it.
    chunkSizeWarningLimit: 1600,
  },
  worker: { format: 'es' },
});
