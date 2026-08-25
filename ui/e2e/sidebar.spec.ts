import { expect, test } from '@playwright/test';

// Same seeded session as protected.spec.ts. Kept local rather than shared so a
// change to that file's fixture cannot silently alter what this one tests.
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

/** Every sidebar link currently marked active, by href. */
async function activeHrefs(page: import('@playwright/test').Page) {
  return page.evaluate(() =>
    [...document.querySelectorAll('[data-slot="sidebar-content"] a[data-active]')].map((a) =>
      a.getAttribute('href'),
    ),
  );
}

test.describe('sidebar', () => {
  // The bug this guards: the nav marked an item active with
  // pathname.startsWith(href). /fleet is a prefix of /fleet/hypervisors, so
  // visiting the latter lit up BOTH — telling an operator they were in two
  // places at once. The fix picks the longest matching href, so exactly one
  // entry is ever active and it stays right as routes are added.
  for (const [path, expected] of [
    ['/fleet', '/fleet'],
    ['/fleet/hypervisors', '/fleet/hypervisors'],
    ['/fleet/hives', '/fleet/hives'],
    ['/estates', '/estates'],
    ['/', '/'],
  ] as const) {
    test(`marks exactly one entry active on ${path}`, async ({ page }) => {
      await page.goto(path);
      await page.waitForLoadState('networkidle');

      const active = await activeHrefs(page);
      expect(
        active,
        `${path} should light exactly one nav entry, got ${JSON.stringify(active)}`,
      ).toEqual([expected]);
    });
  }

  test('opens the group containing the current page', async ({ page }) => {
    // An operator who lands on /fleet/hives from a link or a reload should see
    // where they are without expanding anything.
    await page.goto('/fleet/hives');
    await page.waitForLoadState('networkidle');

    await expect(
      page.locator('[data-slot="sidebar-content"] a[href="/fleet/hives"]'),
    ).toBeVisible();
    await expect(
      page.locator('[data-slot="sidebar-content"] a[href="/fleet/hypervisors"]'),
    ).toBeVisible();
  });

  test('keeps an unrelated group collapsed', async ({ page }) => {
    await page.goto('/fleet/hives');
    await page.waitForLoadState('networkidle');

    // Firewall is a different group, so its children stay put away. Base UI
    // unmounts a closed panel, so "not visible" here means genuinely absent
    // rather than merely hidden.
    await expect(
      page.locator('[data-slot="sidebar-content"] a[href="/firewall/rulesets"]'),
    ).toHaveCount(0);
  });

  test('expands a collapsed group when its trigger is used', async ({ page }) => {
    await page.goto('/fleet/hives');
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: 'Firewall' }).click();

    await expect(
      page.locator('[data-slot="sidebar-content"] a[href="/firewall/rulesets"]'),
    ).toBeVisible();
    // Opening a group must not change WHERE the operator is.
    expect(await activeHrefs(page)).toEqual(['/fleet/hives']);
  });
});
