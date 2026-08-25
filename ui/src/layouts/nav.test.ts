import { describe, expect, it } from 'vitest';

import { activeHref } from './nav';

describe('activeHref', () => {
  const hrefs = ['/', '/estates', '/fleet', '/fleet/hypervisors', '/fleet/hives'];

  it('picks the exact match', () => {
    expect(activeHref('/estates', hrefs)).toBe('/estates');
    expect(activeHref('/fleet', hrefs)).toBe('/fleet');
  });

  it('picks the longest match when one href is a prefix of another', () => {
    // The bug this replaced: a per-item `pathname.startsWith(href)` lit up both
    // /fleet and /fleet/hypervisors, telling an operator they were in two places
    // at once.
    expect(activeHref('/fleet/hypervisors', hrefs)).toBe('/fleet/hypervisors');
    expect(activeHref('/fleet/hives', hrefs)).toBe('/fleet/hives');
  });

  it('does not depend on the order the entries are declared in', () => {
    // The property an end-to-end test cannot show: it can only ever observe the
    // one order the nav happens to be written in. A last-match-wins rule is
    // correct only while the list stays sorted shortest-first, and would break
    // silently the first time someone reordered the menu.
    const reversed = [...hrefs].reverse();
    for (const path of ['/fleet', '/fleet/hypervisors', '/fleet/hives', '/estates', '/']) {
      expect(activeHref(path, reversed), `reversed order, path ${path}`).toBe(
        activeHref(path, hrefs),
      );
    }
  });

  it('treats the root as exact only', () => {
    // '/' is a prefix of every path, so a naive prefix rule marks Dashboard
    // active on every page in the application.
    expect(activeHref('/', hrefs)).toBe('/');
    expect(activeHref('/estates', hrefs)).not.toBe('/');
    expect(activeHref('/fleet/hives', hrefs)).not.toBe('/');
  });

  it('matches a subpath of an entry that has no child entries', () => {
    // A future /estates/:id must still light Estates.
    expect(activeHref('/estates/abc-123', hrefs)).toBe('/estates');
  });

  it('does not match a sibling that merely shares a prefix string', () => {
    // /fleetwide is not inside /fleet. Matching on the raw string prefix rather
    // than a path segment would say it is.
    expect(activeHref('/fleetwide', hrefs)).toBeNull();
  });

  it('returns null when nothing matches', () => {
    expect(activeHref('/nowhere', hrefs)).toBeNull();
  });
});
