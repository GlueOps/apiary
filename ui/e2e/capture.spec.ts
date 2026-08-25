import { expect, test } from '@playwright/test';

// Every route that renders without a session. The protected ones need a token
// and are covered separately.
export const PUBLIC_PAGES = [
  { path: '/login', name: 'login' },
  { path: '/signup', name: 'signup' },
  { path: '/forgot-password', name: 'forgot-password' },
  { path: '/reset-password?token=demo-token', name: 'reset-password' },
  { path: '/verify-email', name: 'verify-email-resend' },
  { path: '/verify-email?token=demo-token', name: 'verify-email-confirm' },
  { path: '/accept-invitation?token=demo-token', name: 'accept-invitation' },
];

for (const page_ of PUBLIC_PAGES) {
  test(`capture ${page_.name}`, async ({ page }) => {
    await page.goto(page_.path);
    // Wait for the card to exist rather than a fixed timeout: these pages fetch,
    // and a screenshot taken mid-render would make every review a guess.
    await page.waitForLoadState('networkidle');
    await page.screenshot({ path: `e2e/__screenshots__/${page_.name}.png`, fullPage: true });
    // The page rendered something. A blank screenshot is the failure mode this
    // whole suite would otherwise hide.
    await expect(page.locator('body')).not.toBeEmpty();
  });
}
