/**
 * The set page's job is to make a list editable. Two things it must not get
 * wrong, both of which the API enforces and which are better caught before a
 * request than after a 4xx.
 */
import { describe, expect, it } from 'vitest';

import type { SetView } from '@/sdk/types.gen';

/** Mirrors the guard in rule-form.tsx. */
const emptySetError = (set: SetView | null) =>
  set && (set.elements ?? []).length === 0
    ? `${set.name} has no addresses in it, so this rule would match nothing. Add one on the Address sets page.`
    : null;

const set = (over: Partial<SetView>): SetView => ({
  id: 'a',
  name: 'allowed_ips',
  family: 'ip',
  purpose: 'general',
  managed: false,
  elements: [],
  ...over,
});

describe('choosing a set for a rule', () => {
  it('refuses an empty set, which would match nothing', () => {
    // An accept rule against an empty set is a silent no-op — it sits in the
    // chain looking like access control and grants none.
    expect(emptySetError(set({ elements: [] }))).toContain('match nothing');
  });

  it('accepts a set that has addresses', () => {
    expect(emptySetError(set({ elements: [{ id: 'e', value: '192.0.2.10' }] }))).toBeNull();
  });

  it('has nothing to say when no set is chosen', () => {
    expect(emptySetError(null)).toBeNull();
  });
});

describe('a set family', () => {
  // A set declares its family; a rule stays inet. Matching an ip6 set against
  // an ip4 literal is legal to write and matches nothing at runtime.
  const familyOfSet = (s: SetView | null) => (s ? (s.family === 'ip6' ? 'ip6' : 'ip') : null);

  it('is taken from the set rather than parsed from an address', () => {
    expect(familyOfSet(set({ family: 'ip6' }))).toBe('ip6');
    expect(familyOfSet(set({ family: 'ip' }))).toBe('ip');
    expect(familyOfSet(null)).toBeNull();
  });
});
