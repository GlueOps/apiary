import {
  type ReactNode,
  createContext,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

import { client } from '@/sdk/client.gen';
import { authLogout, authRefresh } from '@/sdk/sdk.gen';
import type {
  LoginResponseBody,
  MfaVerifyResponseBody,
  PasskeyFinishLoginResponseBody,
} from '@/sdk/types.gen';

export interface AuthSession {
  accessToken: string;
  refreshToken: string;
  organizationId: string;
  organization: string;
  /** When the ACCESS token expires — the trigger to refresh, not to give up. */
  expiresAt: number;
  /**
   * When the REFRESH token expires, and therefore when the session is genuinely
   * over.
   *
   * Tracked separately because conflating the two threw away working sessions:
   * restore() discarded anything whose ACCESS token had expired, so closing a
   * tab for sixteen minutes meant signing in again while holding a refresh token
   * good for another seven days.
   */
  refreshExpiresAt: number;
}

/** Pending MFA challenge returned by login when mfa_required is true. */
export interface MFAPending {
  mfaToken: string;
  organizationId: string;
  organization: string;
}

export interface AuthContextValue {
  authed: boolean;
  session: AuthSession | null;
  /** Save a full token response (from login, MFA verify, passkey, or social). */
  save: (data: TokenData) => void;
  /** Clear the session (local-only, call logout() for server-side revocation). */
  clear: () => void;
  /** Revoke the current session on the server and clear locally. */
  logout: () => Promise<void>;
}

/**
 * Union of all response shapes that carry tokens.
 * LoginResponseBody fields are optional (absent when mfa_required),
 * so callers must ensure they only pass it when tokens are present.
 */
export type TokenData = LoginResponseBody | MfaVerifyResponseBody | PasskeyFinishLoginResponseBody;

const STORAGE_KEY = 'apiary_session';

function persist(session: AuthSession) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
}

function restore(): AuthSession | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const s = JSON.parse(raw) as AuthSession;
    // The REFRESH expiry decides whether a session is over. An expired access
    // token is the normal state of a tab that has been closed for a quarter of
    // an hour, and it is exactly what the refresh token exists to fix.
    //
    // Sessions stored before refreshExpiresAt existed have it undefined; treat
    // those as still refreshable rather than logging everyone out on deploy.
    const refreshExpiresAt = s.refreshExpiresAt ?? Number.POSITIVE_INFINITY;
    if (refreshExpiresAt <= Date.now()) {
      localStorage.removeItem(STORAGE_KEY);
      return null;
    }
    return s;
  } catch {
    localStorage.removeItem(STORAGE_KEY);
    return null;
  }
}

export const AuthContext = createContext<AuthContextValue | null>(null);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [session, setSession] = useState<AuthSession | null>(restore);

  const save = useCallback((data: TokenData) => {
    // Normalize: LoginResponseBody has optional fields, others are required.
    const accessToken = data.access_token;
    const refreshToken = data.refresh_token;
    const expiresIn = data.expires_in;

    if (!accessToken || !refreshToken || expiresIn == null) {
      // This is an MFA-pending login response — nothing to persist.
      return;
    }

    const organizationId = data.organization_id;
    const organization = 'organization' in data ? (data.organization ?? '') : '';

    const s: AuthSession = {
      accessToken,
      refreshToken,
      organizationId,
      organization,
      expiresAt: Date.now() + expiresIn * 1000,
      // Falls back to the access expiry only if the server did not send it,
      // which keeps behaviour identical to before rather than inventing a
      // lifetime the server never promised.
      refreshExpiresAt:
        'refresh_expires_in' in data && typeof data.refresh_expires_in === 'number'
          ? Date.now() + data.refresh_expires_in * 1000
          : Date.now() + expiresIn * 1000,
    };
    persist(s);
    sessionRef.current = s;
    setSession(s);
  }, []);

  const clear = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY);
    setSession(null);
  }, []);

  const sessionRef = useRef(session);
  useEffect(() => {
    sessionRef.current = session;
  }, [session]);

  const logout = useCallback(async () => {
    try {
      await authLogout();
    } catch {
      // best-effort server revocation
    }
    clear();
  }, [clear]);

  // Attach Bearer token to every SDK request
  useEffect(() => {
    const eject = client.interceptors.request.use((req) => {
      const cur = sessionRef.current;
      if (cur?.accessToken) {
        req.headers.set('Authorization', `Bearer ${cur.accessToken}`);
      }
      if (cur?.organizationId) {
        req.headers.set('X-Org-ID', cur.organizationId);
      }
      return req;
    });
    return () => client.interceptors.request.eject(eject);
  }, []);

  // Adopt a session another tab rotated into.
  //
  // Refresh tokens ROTATE with reuse detection: the server revokes the old
  // session row and issues a new pair. So the moment one tab refreshes, every
  // other tab is holding an access token whose session is revoked, and its very
  // next request of ANY kind gets a 401 — not merely its own refresh. Without
  // this listener two tabs cannot coexist past the first rotation, which is
  // about fourteen minutes.
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key !== STORAGE_KEY) return;
      if (!e.newValue) {
        setSession(null);
        return;
      }
      try {
        const s = JSON.parse(e.newValue) as AuthSession;
        sessionRef.current = s;
        setSession(s);
      } catch {
        // A corrupt value is not a reason to destroy a working in-memory
        // session; the next refresh will overwrite it.
      }
    };
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, []);

  // Auto-refresh, single-flighted ACROSS TABS.
  //
  // Rotation plus reuse detection makes a concurrent refresh actively harmful,
  // not merely wasteful: the second tab presents a refresh token whose session
  // is already revoked, which the server correctly reads as theft and answers by
  // revoking the WHOLE family and writing a theft_detected audit event. Two open
  // tabs would do that to themselves on a timer — logging the user out and
  // poisoning the very signal that is supposed to detect real token theft.
  //
  // navigator.locks serialises the tabs, and re-reading storage INSIDE the lock
  // is what makes it correct: the tab that waited discovers the winner already
  // rotated and returns instead of spending the token it was holding.
  useEffect(() => {
    if (!session) return;

    // Fires on the ACCESS expiry; the session is only over when the REFRESH
    // token expires, which restore() and the guard below handle separately.
    const refreshIn = Math.max(session.expiresAt - Date.now() - 60_000, 0);

    let cancelled = false;

    const runRefresh = async () => {
      const current = restore() ?? sessionRef.current;
      if (!current) return;

      // Somebody else already rotated while we waited for the lock.
      if (current.refreshToken !== session.refreshToken) {
        sessionRef.current = current;
        setSession(current);
        return;
      }
      if ((current.refreshExpiresAt ?? Number.POSITIVE_INFINITY) <= Date.now()) {
        clear();
        return;
      }

      try {
        const { data } = await authRefresh({ body: { refresh_token: current.refreshToken } });
        if (data && !cancelled) save(data);
      } catch (err) {
        // A network blip is NOT a revoked session. Clearing on both meant a
        // dropped connection during refresh signed the user out, with a valid
        // refresh token still in hand. Only an explicit refusal ends the
        // session; anything else leaves it alone for the next attempt.
        const status =
          (err as { response?: { status?: number }; status?: number })?.response?.status ??
          (err as { status?: number })?.status;
        if (status === 401 || status === 403) clear();
      }
    };

    const timer = setTimeout(() => {
      if (typeof navigator !== 'undefined' && navigator.locks?.request) {
        void navigator.locks.request('apiary-auth-refresh', runRefresh);
      } else {
        // Older browsers get the previous behaviour rather than none. The race
        // is back for them, which is worse than the lock and better than not
        // refreshing at all.
        void runRefresh();
      }
    }, refreshIn);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [session, save, clear]);

  // Listen for social auth postMessage from popup
  useEffect(() => {
    const handler = (e: MessageEvent) => {
      const allowed = new Set([window.location.origin]);
      // Treat localhost and 127.0.0.1 as equivalent (popup callback
      // may arrive on a different loopback address than the opener).
      try {
        const u = new URL(window.location.origin);
        if (u.hostname === 'localhost') {
          allowed.add(u.origin.replace('localhost', '127.0.0.1'));
        } else if (u.hostname === '127.0.0.1') {
          allowed.add(u.origin.replace('127.0.0.1', 'localhost'));
        }
      } catch {
        // ignore parse errors
      }
      if (!allowed.has(e.origin)) return;
      // Must match the type posted by internal/handlers/auth/social.go. The two
      // halves live in different languages, so nothing but this comment ties
      // them together — change one and the OAuth popup silently never resolves.
      if (e.data?.type !== 'apiary:oauth:complete') return;
      const payload = e.data.payload;
      if (!payload) return;

      // Social auth may return mfa_required — callers handle that separately
      if (payload.mfa_required) return;

      save(payload as TokenData);
    };
    window.addEventListener('message', handler);
    return () => window.removeEventListener('message', handler);
  }, [save]);

  const value = useMemo<AuthContextValue>(
    () => ({ authed: session !== null, session, save, clear, logout }),
    [session, save, clear, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
