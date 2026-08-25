import type { Icon } from '@tabler/icons-react';

type NavLeaf = { title: string; href: string; icon: Icon };
type NavGroup = {
  title: string;
  icon: Icon;
  children: { title: string; href: string; icon: Icon }[];
};
type NavEntry = NavLeaf | NavGroup;

function isGroup(e: NavEntry): e is NavGroup {
  return 'children' in e;
}

/**
 * The single active entry, chosen as the LONGEST matching href.
 *
 * Not `pathname.startsWith(href)` per item, which is what this replaced: with
 * /fleet and /fleet/hypervisors both in the list, visiting the latter lit up
 * both, because one href is a prefix of the other. Picking the longest match
 * means exactly one entry is ever active, and it stays correct as routes are
 * added without anyone having to notice the prefix relationship.
 */
function activeHref(pathname: string, hrefs: string[]): string | null {
  let best: string | null = null;
  for (const href of hrefs) {
    const matches =
      href === '/' ? pathname === '/' : pathname === href || pathname.startsWith(href + '/');
    if (matches && (best === null || href.length > best.length)) best = href;
  }
  return best;
}

function allHrefs(entries: NavEntry[]): string[] {
  return entries.flatMap((e) => (isGroup(e) ? e.children.map((c) => c.href) : [e.href]));
}

export { activeHref, allHrefs, isGroup };
export type { NavEntry, NavGroup, NavLeaf };
