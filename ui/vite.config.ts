import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, path.resolve(import.meta.dirname, '..'), 'VITE_');
  const apiUrl = env.VITE_API_URL || 'http://localhost:8080';

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: {
        '@': path.resolve(import.meta.dirname, './src'),
      },
    },
    server: {
      proxy: {
        '/api': {
          target: apiUrl,
          changeOrigin: true,
        },
        '/docs': {
          target: apiUrl,
          changeOrigin: true,
        },
      },
    },
    build: {
      chunkSizeWarningLimit: 1000,
      outDir: '../internal/api/webdist',
      emptyOutDir: true,
      sourcemap: mode !== 'production',
      cssMinify: 'lightningcss',
      // React is split out because it is the one dependency that dwarfs
      // everything else and changes least: keeping it in its own chunk means an
      // application edit does not invalidate half a megabyte of vendor code in
      // every visitor's cache.
      rolldownOptions: {
        output: {
          codeSplitting: {
            groups: [
              {
                name: 'react',
                test: /react|react-dom|react-router/,
              },
            ],
          },
        },
      },
    },
  };
});
