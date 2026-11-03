import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  base: '/static/dist/',
  publicDir: false,
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    cssCodeSplit: false,
    rollupOptions: {
      input: fileURLToPath(new URL('./src/main.jsx', import.meta.url)),
      output: {
        // The Go template loads these stable entry points directly.
        entryFileNames: 'index.js',
        assetFileNames: (asset) => (
          asset.names.some((name) => name.endsWith('.css'))
            ? 'index.css'
            : 'assets/[name]-[hash][extname]'
        ),
      },
    },
  },
});
