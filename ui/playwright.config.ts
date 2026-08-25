import { defineConfig, devices } from '@playwright/test';

// Port chosen to avoid colliding with a dev server the developer already has up
// on 5173 — these tests start their own and must not attach to whatever happens
// to be running, which would make results depend on someone's local state.
const PORT = 5199;

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : [['list']],

  use: {
    baseURL: `http://localhost:${PORT}`,
    trace: 'on-first-retry',
    // A fixed viewport: layout assertions about wrapping and overflow are
    // meaningless if the window size drifts between runs.
    viewport: { width: 1280, height: 800 },
  },

  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],

  webServer: {
    command: `yarn dev --port ${PORT} --strictPort`,
    url: `http://localhost:${PORT}`,
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
