import react from '@vitejs/plugin-react';
import path from 'path';
import { defineConfig } from 'vitest/config';

// A separate config from vite.config.ts on purpose: the build config carries the
// dev proxy, the tailwind plugin and the rolldown chunking, none of which a test
// run needs, and adding a `test` block there would make `yarn build` depend on
// the test runner's types being installed.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  test: {
    // Playwright owns e2e/. Without this vitest picks the specs up, fails to
    // resolve @playwright/test's runner, and reports a suite-wide error that
    // looks like a broken unit test.
    exclude: ['e2e/**', 'node_modules/**'],
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
    setupFiles: ['./src/test-setup.ts'],
    restoreMocks: true,
  },
});
