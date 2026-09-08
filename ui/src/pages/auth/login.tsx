import React, { useState } from 'react';

import { useMutation } from '@tanstack/react-query';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Link } from 'react-router-dom';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import { useAuth } from '@/hooks/use-auth';
import { usePublicFlags } from '@/hooks/use-public-flags';
import type { MFAPending, TokenData } from '@/providers/auth-provider';
import { authLoginMutation } from '@/sdk/@tanstack/react-query.gen';
import { client } from '@/sdk/client.gen';
import { authMfaVerify, authPasskeyLoginBegin, authPasskeyLoginFinish } from '@/sdk/sdk.gen';
import type { LoginResponseBody } from '@/sdk/types.gen';

function GoogleIcon() {
  return (
    <svg viewBox="0 0 24 24" className="size-4">
      <path
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"
        fill="#4285F4"
      />
      <path
        d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
        fill="#34A853"
      />
      <path
        d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
        fill="#FBBC05"
      />
      <path
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
        fill="#EA4335"
      />
    </svg>
  );
}

function GitHubIcon() {
  return (
    <svg viewBox="0 0 24 24" className="size-4 fill-current">
      <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
    </svg>
  );
}

function PasskeyIcon() {
  return (
    <svg viewBox="0 0 24 24" className="size-4 fill-current">
      <path d="M12 2a5 5 0 0 0-5 5 5 5 0 0 0 3 4.58V22h2v-4h2v-2h-2v-2h2v-2h-2v-1.42A5 5 0 0 0 15 7a5 5 0 0 0-3-4.58A5 5 0 0 0 12 2m0 2a3 3 0 0 1 3 3 3 3 0 0 1-3 3 3 3 0 0 1-3-3 3 3 0 0 1 3-3Z" />
    </svg>
  );
}

function extractMFAPending(data: LoginResponseBody): MFAPending | null {
  if (!data.mfa_required || !data.mfa_token) return null;
  return {
    mfaToken: data.mfa_token,
    organizationId: data.organization_id,
    organization: data.organization,
  };
}

async function startSocialAuth(provider: string) {
  const baseUrl = client.getConfig().baseUrl;
  const origin = window.location.origin;
  const res = await fetch(
    `${baseUrl}/api/v1/auth/${provider}/start?mode=spa&origin=${encodeURIComponent(origin)}`,
    { method: 'POST' },
  );
  if (!res.ok) throw new Error(`Failed to start ${provider} auth`);
  const { auth_url } = (await res.json()) as { auth_url: string };

  const w = 500;
  const h = 600;
  const left = window.screenX + (window.innerWidth - w) / 2;
  const top = window.screenY + (window.innerHeight - h) / 2;
  window.open(auth_url, `${provider}_auth`, `width=${w},height=${h},left=${left},top=${top}`);
}

export const Login = () => {
  const { save } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [mfaPending, setMfaPending] = useState<MFAPending | null>(null);
  const [totpCode, setTotpCode] = useState('');
  const [mfaSubmitting, setMfaSubmitting] = useState(false);
  const [passkeyLoading, setPasskeyLoading] = useState(false);
  const { enabled } = usePublicFlags();
  const passwordSignIn = enabled('legacy_login');

  // Mirrors ErrTypeEmailNotVerified in internal/handlers/auth/login.go. The
  // generated SDK does not surface error type URIs, so this is hand-carried; a Go
  // test pins the server side so the two cannot drift silently in that direction.
  const ERR_EMAIL_NOT_VERIFIED = 'https://apiary.glueops.dev/errors/email-not-verified';

  const login = useMutation({
    ...authLoginMutation(),
    onSuccess: (data: LoginResponseBody) => {
      const pending = extractMFAPending(data);
      if (pending) {
        setMfaPending(pending);
        return;
      }
      save(data);
      navigate(searchParams.get('to') || '/', { replace: true });
    },
    onError: (err) => {
      const problem = err as { detail?: string; type?: string };

      // The server refuses an unconfirmed address with its own RFC 7807 type,
      // reached only AFTER the password has been checked — so anyone who sees it
      // has already proved the account is theirs, and sending them somewhere
      // useful discloses nothing. Without this branch the whole point of making
      // that refusal distinguishable is wasted: the user gets a toast telling
      // them to confirm, and no way to ask for another link.
      //
      // Branching on `type`, never on `detail`: RFC 7807 says the type is the
      // stable identifier and the detail is prose that may be reworded.
      if (problem?.type === ERR_EMAIL_NOT_VERIFIED) {
        navigate(`/verify-email?email=${encodeURIComponent(email)}`, { replace: true });
        return;
      }

      toast.error(problem?.detail ?? 'Login failed');
    },
  });

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault();
    login.mutate({
      body: { email, password },
    });
  };

  const handleMfaSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!mfaPending) return;
    setMfaSubmitting(true);
    try {
      const { data, error } = await authMfaVerify({
        body: { mfa_token: mfaPending.mfaToken, code: totpCode },
      });
      if (error || !data) {
        toast.error('Invalid TOTP code');
        return;
      }
      save(data as TokenData);
    } catch {
      toast.error('MFA verification failed');
    } finally {
      setMfaSubmitting(false);
    }
  };

  const handlePasskeyLogin = async () => {
    setPasskeyLoading(true);
    try {
      const { data: beginData, error: beginErr } = await authPasskeyLoginBegin();
      if (beginErr || !beginData?.options) {
        toast.error('Passkey login unavailable');
        return;
      }

      const rawPK = beginData.options.publicKey;
      if (!rawPK) {
        toast.error('Invalid passkey options');
        return;
      }

      const b64ToBuffer = (s: string) =>
        Uint8Array.from(atob(s.replace(/-/g, '+').replace(/_/g, '/')), (c) => c.charCodeAt(0))
          .buffer;

      // Build a native PublicKeyCredentialRequestOptions from the SDK shape.
      // The SDK types use `string` for WebAuthn enum fields; cast via unknown.
      const publicKey = {
        ...rawPK,
        challenge: b64ToBuffer(rawPK.challenge as unknown as string),
        allowCredentials: rawPK.allowCredentials
          ? rawPK.allowCredentials.map((c) => ({
              id: b64ToBuffer(c.id as unknown as string),
              type: 'public-key' as const,
              transports: (c.transports ?? undefined) as AuthenticatorTransport[] | undefined,
            }))
          : undefined,
        userVerification: rawPK.userVerification as UserVerificationRequirement | undefined,
      } satisfies PublicKeyCredentialRequestOptions;

      const assertion = (await navigator.credentials.get({
        publicKey,
      })) as PublicKeyCredential;
      if (!assertion) {
        toast.error('Passkey assertion cancelled');
        return;
      }

      const response = assertion.response as AuthenticatorAssertionResponse;

      const toBase64Url = (buf: ArrayBuffer) =>
        btoa(String.fromCharCode(...new Uint8Array(buf)))
          .replace(/\+/g, '-')
          .replace(/\//g, '_')
          .replace(/=+$/, '');

      const credential = {
        id: assertion.id,
        rawId: toBase64Url(assertion.rawId),
        type: assertion.type,
        response: {
          authenticatorData: toBase64Url(response.authenticatorData),
          clientDataJSON: toBase64Url(response.clientDataJSON),
          signature: toBase64Url(response.signature),
          userHandle: response.userHandle ? toBase64Url(response.userHandle) : undefined,
        },
      };

      const { data: finishData, error: finishErr } = await authPasskeyLoginFinish({
        body: {
          challenge: beginData.challenge,
          credential,
        },
      });
      if (finishErr || !finishData) {
        toast.error('Passkey login failed');
        return;
      }
      save(finishData as TokenData);
    } catch (err) {
      // NotAllowedError = user cancelled the browser prompt
      if (err instanceof DOMException && err.name === 'NotAllowedError') {
        return;
      }
      toast.error('Passkey login failed');
    } finally {
      setPasskeyLoading(false);
    }
  };

  const handleSocial = (provider: string) => {
    startSocialAuth(provider).catch(() => toast.error(`Could not start ${provider} sign-in`));
  };

  const handleBackToLogin = () => {
    setMfaPending(null);
    setTotpCode('');
  };

  // MFA verification step
  if (mfaPending) {
    return (
      <div className="flex min-h-screen w-full items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="text-lg md:text-xl">Two-factor authentication</CardTitle>
            <CardDescription className="text-xs md:text-sm">
              Enter the 6-digit code from your authenticator app.
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-4">
            <form onSubmit={handleMfaSubmit} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="totp-code" className="text-sm leading-none font-medium">
                  Code
                </label>
                <Input
                  id="totp-code"
                  type="text"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  placeholder="000000"
                  maxLength={6}
                  required
                  autoFocus
                  value={totpCode}
                  onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, ''))}
                />
              </div>

              <Button
                type="submit"
                className="w-full"
                disabled={mfaSubmitting || totpCode.length !== 6}
              >
                {mfaSubmitting ? 'Verifying...' : 'Verify'}
              </Button>
            </form>

            <Button variant="ghost" className="w-full" onClick={handleBackToLogin}>
              Back to sign in
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen w-full items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-lg md:text-xl">Sign in</CardTitle>
          <CardDescription className="text-xs md:text-sm">
            {passwordSignIn
              ? 'Continue with email/password or a provider below to access your account.'
              : 'Continue with a provider below to access your account.'}
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* Hidden, not disabled. A disabled form still says "you could sign in
              this way if only you were allowed", which is untrue — the server
              refuses the request outright. */}
          {passwordSignIn ? (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="email" className="text-sm leading-none font-medium">
                  Email
                </label>
                <Input
                  id="email"
                  type="email"
                  placeholder="you@example.com"
                  autoComplete="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>

              <div className="space-y-2">
                <label htmlFor="password" className="text-sm leading-none font-medium">
                  Password
                </label>
                <Input
                  id="password"
                  type="password"
                  placeholder="Enter your password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              </div>

              <div className="flex items-center justify-between">
                <Link
                  to="/forgot-password"
                  className="text-xs text-muted-foreground hover:underline"
                >
                  Forgot password?
                </Link>
              </div>

              <Button type="submit" className="w-full" disabled={login.isPending}>
                {login.isPending ? 'Signing in...' : 'Sign in'}
              </Button>
            </form>
          ) : null}

          {/* The separator only separates something from something. With the
              password form gone the providers are the whole of the page, and an
              "OR" above them reads as a missing half. */}
          {passwordSignIn ? (
            <div className="flex items-center gap-4">
              <Separator className="flex-1" />
              <span className="text-xs text-muted-foreground">OR</span>
              <Separator className="flex-1" />
            </div>
          ) : null}

          <div className="grid grid-cols-3 gap-3">
            <Button variant="outline" type="button" onClick={() => handleSocial('google')}>
              <GoogleIcon />
              Google
            </Button>
            <Button variant="outline" type="button" onClick={() => handleSocial('github')}>
              <GitHubIcon />
              GitHub
            </Button>
            <Button
              variant="outline"
              type="button"
              disabled={passkeyLoading}
              onClick={handlePasskeyLogin}
            >
              <PasskeyIcon />
              {passkeyLoading ? '...' : 'Passkey'}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};
