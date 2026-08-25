import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, path.resolve(import.meta.dirname, '..'), 'VITE_');
  const apiUrl = env.VITE_API_URL || 'http://localhost:8080';

  // The Go server proxies everything that is not /api or /docs to VITE_DEV_URL,
  // so the port Vite listens on has to be the port that URL names. Deriving it
  // from the same variable keeps one source of truth: a separate VITE_PORT could
  // be changed on its own, leaving the server proxying to a port nothing is
  // listening on.
  //
  // loadEnv returns strings and server.port is typed number, so Number() is what
  // keeps `tsc -b` happy — which yarn build runs, and which does cover this file
  // even though `tsc --noEmit` does not.
  const devUrl = env.VITE_DEV_URL || 'http://localhost:5174';
  const vitePort = Number(new URL(devUrl).port) || 5174;

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: {
        '@': path.resolve(import.meta.dirname, './src'),
      },
    },
    server: {
      port: vitePort,
      // Fail rather than drift to the next free port: a silent move to 5175
      // leaves the Go server proxying to nothing, which surfaces as a blank page
      // rather than an error.
      strictPort: true,
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
