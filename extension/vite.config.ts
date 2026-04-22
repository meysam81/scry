import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import { crx } from '@crxjs/vite-plugin';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import manifest from './manifest.config';

const here = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [vue(), crx({ manifest })],
  resolve: {
    alias: {
      '@': resolve(here, 'src'),
    },
  },
  build: {
    target: 'esnext',
    sourcemap: true,
    emptyOutDir: true,
  },
  server: {
    port: 5174,
    strictPort: true,
    hmr: {
      port: 5174,
    },
  },
});
