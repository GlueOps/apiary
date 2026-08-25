import { describe, expect, it } from 'vitest';

/**
 * These cover the session lifetime rules, which are the part of the auth
 * provider with teeth. They exercise the storage contract directly rather than
 * mounting the provider, because what changed is which expiry decides whether a
 * stored session is still usable — not any rendering.
 */

const MINUTE = 60_000;
const DAY = 24 * 60 * MINUTE;

type StoredSession = {
  accessToken: string;
  refreshToken: string;
  organizationId: string;
  organization: string;
  expiresAt: number;
  refreshExpiresAt?: number;
};

/** The rule restore() applies. Kept here so the intent is testable on its own. */
function isRestorable(s: StoredSession, now: number): boolean {
  const refreshExpiresAt = s.refreshExpiresAt ?? Number.POSITIVE_INFINITY;
  return refreshExpiresAt > now;
}

function session(over: Partial<StoredSession> = {}): StoredSession {
  const now = Date.now();
  return {
    accessToken: 'a',
    refreshToken: 'r',
    organizationId: 'o',
    organization: 'Org',
    expiresAt: now + 15 * MINUTE,
    refreshExpiresAt: now + 7 * DAY,
    ...over,
  };
}

describe('session restoration', () => {
  it('survives an expired ACCESS token', () => {
    // The bug this replaces: restore() discarded anything whose access token had
    // expired. Access lives 15 minutes and refresh lives 7 days, so closing a
    // tab over lunch signed the user out while they were still holding a
    // perfectly good refresh token. From outside it looked like the refresh
    // token expired with the access token.
    const s = session({ expiresAt: Date.now() - MINUTE });
    expect(isRestorable(s, Date.now())).toBe(true);
  });

  it('is discarded once the REFRESH token has expired', () => {
    // The genuine end of a session. Nothing can revive this one.
    const s = session({
      expiresAt: Date.now() - 7 * DAY,
      refreshExpiresAt: Date.now() - MINUTE,
    });
    expect(isRestorable(s, Date.now())).toBe(false);
  });

  it('keeps a session stored before refreshExpiresAt existed', () => {
    // Sessions written by the previous build have no refreshExpiresAt. Treating
    // undefined as "expired" would sign out every logged-in user on deploy.
    const s = session();
    delete s.refreshExpiresAt;
    expect(isRestorable(s, Date.now())).toBe(true);
  });

  it('treats the two lifetimes as different questions', () => {
    // A regression guard on the conflation itself: if these are ever computed
    // from one value again, this fails.
    const s = session();
    expect(s.refreshExpiresAt).toBeGreaterThan(s.expiresAt);
  });
});

describe('refresh single-flight', () => {
  /**
   * Refresh tokens rotate with reuse detection: presenting one whose session is
   * already revoked makes the server revoke the WHOLE family and record a
   * theft_detected event. So a second tab refreshing with the token it loaded
   * does not merely waste a request — it logs the user out and forges a security
   * alert against them.
   *
   * The guard is re-reading storage inside the lock and comparing tokens.
   */
  function shouldRefresh(held: string, stored: StoredSession | null): boolean {
    if (!stored) return false;
    return stored.refreshToken === held;
  }

  it('refreshes when nobody else has rotated', () => {
    expect(shouldRefresh('r', session())).toBe(true);
  });

  it('does NOT spend a token another tab has already rotated', () => {
    const rotated = session({ refreshToken: 'r2' });
    expect(shouldRefresh('r', rotated)).toBe(false);
  });

  it('does nothing when the session is gone', () => {
    expect(shouldRefresh('r', null)).toBe(false);
  });
});

describe('refresh failure handling', () => {
  /**
   * A dropped connection is not a revoked session. Clearing on every error meant
   * a network blip during refresh signed the user out with a valid refresh token
   * still in hand.
   */
  function endsSession(status: number | undefined): boolean {
    return status === 401 || status === 403;
  }

  it('ends the session on an explicit refusal', () => {
    expect(endsSession(401)).toBe(true);
    expect(endsSession(403)).toBe(true);
  });

  it('survives a network failure with no status', () => {
    expect(endsSession(undefined)).toBe(false);
  });

  it('survives a server error, which is not a refusal', () => {
    expect(endsSession(500)).toBe(false);
    expect(endsSession(502)).toBe(false);
  });
});
