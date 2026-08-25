import { type Page, expect, test } from '@playwright/test';

import { PUBLIC_PAGES } from './pages';

// These tests exist because of a defect that no unit test could have caught and
// that looked, in the browser, like nothing more than "the cards seem cramped".
//
// #root carried Vite-scaffold styling: a fixed 1126px width, a border-inline that
// drew two rules down every page, text-align:center, and display:flex with
// flex-direction:column. That last one is the subtle part — in a COLUMN flex
// container the cross axis is horizontal, so a child with `mx-auto` has its auto
// margins absorb the horizontal free space and shrinks to its own content
// instead of stretching. Every auth card asked for the same `w-full max-w-md`,
// and they rendered at 233px, 394px, 412px and 447px depending only on how long
// their text was. `w-full` was resolving against a parent that had collapsed.
//
// So the assertion is not "the card looks right" but "the card reached the width
// it asked for", which is the property that was actually broken.

async function consoleErrors(page: Page) {
  const errors: string[] = [];
  page.on('console', (m) => {
    if (m.type() === 'error') errors.push(m.text());
  });
  page.on('pageerror', (e) => errors.push(String(e)));
  return errors;
}

for (const p of PUBLIC_PAGES) {
  test.describe(p.name, () => {
    test('card fills its max width rather than shrinking to its content', async ({ page }) => {
      await page.goto(p.path);
      await page.waitForLoadState('networkidle');

      const card = page.locator('[data-slot=card]').first();
      await expect(card).toBeVisible();

      const { width, maxWidth, parentWidth } = await card.evaluate((el) => {
        const cs = getComputedStyle(el);
        return {
          width: parseFloat(cs.width),
          maxWidth: parseFloat(cs.maxWidth),
          parentWidth: parseFloat(getComputedStyle(el.parentElement!).width),
        };
      });

      // The parent must span the viewport. When it collapses, everything below
      // is downstream of that and the card width alone is a confusing symptom.
      expect(
        parentWidth,
        `the card's parent is ${parentWidth}px wide on a 1280px viewport — it has ` +
          `collapsed to its content, so the card's w-full resolves against nothing`,
      ).toBeGreaterThan(1000);

      expect(
        width,
        `card rendered ${width}px but asked for max-width ${maxWidth}px; it is being ` +
          `sized by its text rather than by its layout`,
      ).toBeCloseTo(maxWidth, 0);
    });

    test('does not scroll sideways', async ({ page }) => {
      await page.goto(p.path);
      await page.waitForLoadState('networkidle');
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      );
      expect(overflow, 'the page scrolls horizontally').toBeLessThanOrEqual(0);
    });

    test('renders without console errors', async ({ page }) => {
      const errors = await consoleErrors(page);
      await page.goto(p.path);
      await page.waitForLoadState('networkidle');

      // Failed API calls are expected here: these pages are rendered without a
      // backend, and a 401/404 from fetch is not a rendering defect. Everything
      // else is — this is what caught Base UI warning that a Button rendering a
      // <Link> silently loses its native button semantics.
      const real = errors.filter(
        (e) => !/Failed to load resource|net::ERR|status of 4\d\d|status of 5\d\d/i.test(e),
      );
      expect(real, `console errors on ${p.path}:\n${real.join('\n')}`).toEqual([]);
    });

    test('is legible at a narrow viewport', async ({ page }) => {
      await page.setViewportSize({ width: 380, height: 800 });
      await page.goto(p.path);
      await page.waitForLoadState('networkidle');
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      );
      expect(overflow, 'the page scrolls sideways on a phone-width viewport').toBeLessThanOrEqual(
        0,
      );
    });
  });
}
