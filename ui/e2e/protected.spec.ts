import { expect, test } from '@playwright/test';

import { PROTECTED_PAGES } from './pages';

// A session is seeded directly into localStorage rather than driven through the
// login form. Logging in would need a live backend, a seeded user and a working
// SMTP-free confirmation — turning a layout test into an integration test, and
// making a rendering regression indistinguishable from a backend outage.
//
// The shape must match AuthSession in src/providers/auth-provider.tsx; a drift
// there shows up as every one of these redirecting to /login, which the first
// assertion reports explicitly rather than leaving as a puzzling timeout.
const SESSION = {
  accessToken: 'e2e-access-token',
  refreshToken: 'e2e-refresh-token',
  organizationId: '00000000-0000-0000-0000-000000000001',
  organization: 'E2E Org',
  expiresAt: Date.now() + 60 * 60 * 1000,
};

test.beforeEach(async ({ page }) => {
  await page.addInitScript((s) => {
    window.localStorage.setItem('apiary_session', JSON.stringify(s));
  }, SESSION);
});

for (const p of PROTECTED_PAGES) {
  test.describe(p.name, () => {
    test('renders behind the app shell without redirecting to login', async ({ page }) => {
      await page.goto(p.path);
      await page.waitForLoadState('networkidle');

      expect(
        new URL(page.url()).pathname,
        'redirected to /login — the seeded session no longer matches AuthSession ' +
          'in auth-provider.tsx, so this suite is testing nothing',
      ).not.toBe('/login');

      await page.screenshot({ path: `e2e/__screenshots__/${p.name}.png`, fullPage: true });
      await expect(page.locator('body')).not.toBeEmpty();
    });

    test('does not scroll sideways', async ({ page }) => {
      await page.goto(p.path);
      await page.waitForLoadState('networkidle');
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      );
      expect(overflow, 'the page scrolls horizontally').toBeLessThanOrEqual(0);
    });
  });
}
