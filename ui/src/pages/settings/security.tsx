import React, { useEffect, useState } from 'react';

import {
  IconFingerprint,
  IconKey,
  IconPlus,
  IconShieldCheck,
  IconShieldOff,
  IconTrash,
} from '@tabler/icons-react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { meGetOptions, meGetQueryKey } from '@/sdk/@tanstack/react-query.gen';
import {
  authPasskeyRegisterBegin,
  authPasskeyRegisterFinish,
  authPasskeysDelete,
  authPasskeysList,
  authTotpConfirm,
  authTotpDisable,
  authTotpSetup,
} from '@/sdk/sdk.gen';
import type { PasskeyEntry } from '@/sdk/types.gen';

// --- WebAuthn helpers ---

const b64ToBuffer = (s: string) =>
  Uint8Array.from(atob(s.replace(/-/g, '+').replace(/_/g, '/')), (c) => c.charCodeAt(0)).buffer;

const toBase64Url = (buf: ArrayBuffer) =>
  btoa(String.fromCharCode(...new Uint8Array(buf)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');

// --- Component ---

export const Security = () => {
  const queryClient = useQueryClient();

  // MFA status from user profile
  const { data: me, isLoading: meLoading } = useQuery({
    ...meGetOptions(),
  });

  const mfaEnabled = me?.mfa?.enabled ?? false;
  const totpEnabled = me?.mfa?.totp_enabled ?? false;

  // TOTP setup flow state
  const [totpSetupUri, setTotpSetupUri] = useState<string | null>(null);
  const [totpSetupSecret, setTotpSetupSecret] = useState<string | null>(null);
  const [totpCode, setTotpCode] = useState('');
  const [totpSetupLoading, setTotpSetupLoading] = useState(false);
  const [totpConfirmLoading, setTotpConfirmLoading] = useState(false);

  // TOTP disable dialog
  const [disableDialogOpen, setDisableDialogOpen] = useState(false);
  const [disablePassword, setDisablePassword] = useState('');
  const [disableCode, setDisableCode] = useState('');
  const [disableLoading, setDisableLoading] = useState(false);

  // Passkeys
  //
  // Fetched with useQuery rather than useState + useEffect, matching how `me`
  // is loaded above. The previous version called an async fetcher from an
  // effect, which set state synchronously and cascaded a second render on
  // mount — flagged by react-hooks/set-state-in-effect as of
  // eslint-plugin-react-hooks 7.1. useQuery also gives us the refetch used
  // after register/delete for free.
  const {
    data: passkeys = [],
    isLoading: passkeysLoading,
    isError: passkeysError,
    refetch: refetchPasskeys,
  } = useQuery({
    queryKey: ['auth', 'passkeys'],
    queryFn: async (): Promise<PasskeyEntry[]> => {
      const { data, error } = await authPasskeysList();
      if (error) {
        throw new Error('Failed to load passkeys');
      }
      return data?.passkeys ?? [];
    },
  });

  // Surfacing the load failure as a toast keeps the previous behaviour. No
  // setState here, so this effect does not cascade a render.
  useEffect(() => {
    if (passkeysError) {
      toast.error('Failed to load passkeys');
    }
  }, [passkeysError]);

  const [passkeyRegisterLoading, setPasskeyRegisterLoading] = useState(false);
  const [passkeyDeleteId, setPasskeyDeleteId] = useState<string | null>(null);
  const [passkeyName, setPasskeyName] = useState('');
  const [passkeyNameDialogOpen, setPasskeyNameDialogOpen] = useState(false);
  const [pendingPasskeyCredential, setPendingPasskeyCredential] = useState<unknown>(null);
  const [pendingPasskeyChallenge, setPendingPasskeyChallenge] = useState<string | null>(null);

  const invalidateMe = () => {
    queryClient.invalidateQueries({ queryKey: meGetQueryKey() });
  };

  // --- TOTP Setup ---

  const handleTotpSetup = async () => {
    setTotpSetupLoading(true);
    try {
      const { data, error } = await authTotpSetup();
      if (error || !data) {
        toast.error('Failed to start TOTP setup');
        return;
      }
      setTotpSetupUri(data.uri);
      setTotpSetupSecret(data.secret);
      setTotpCode('');
    } catch {
      toast.error('Failed to start TOTP setup');
    } finally {
      setTotpSetupLoading(false);
    }
  };

  const handleTotpConfirm = async (e: React.FormEvent) => {
    e.preventDefault();
    setTotpConfirmLoading(true);
    try {
      const { error } = await authTotpConfirm({ body: { code: totpCode } });
      if (error) {
        toast.error('Invalid code. Please try again.');
        return;
      }
      toast.success('TOTP enabled successfully');
      setTotpSetupUri(null);
      setTotpSetupSecret(null);
      setTotpCode('');
      invalidateMe();
    } catch {
      toast.error('Failed to verify TOTP code');
    } finally {
      setTotpConfirmLoading(false);
    }
  };

  const handleTotpCancelSetup = () => {
    setTotpSetupUri(null);
    setTotpSetupSecret(null);
    setTotpCode('');
  };

  // --- TOTP Disable ---

  const handleTotpDisable = async (e: React.FormEvent) => {
    e.preventDefault();
    setDisableLoading(true);
    try {
      const { error } = await authTotpDisable({
        body: {
          password: disablePassword || undefined,
          code: disableCode || undefined,
        },
      });
      if (error) {
        toast.error('Failed to disable TOTP');
        return;
      }
      toast.success('TOTP disabled');
      setDisableDialogOpen(false);
      setDisablePassword('');
      setDisableCode('');
      invalidateMe();
    } catch {
      toast.error('Failed to disable TOTP');
    } finally {
      setDisableLoading(false);
    }
  };

  // --- Passkey Registration ---

  const handlePasskeyRegister = async () => {
    setPasskeyRegisterLoading(true);
    try {
      const { data: beginData, error: beginErr } = await authPasskeyRegisterBegin();
      if (beginErr || !beginData?.options) {
        toast.error('Failed to start passkey registration');
        return;
      }

      const rawPK = beginData.options.publicKey;
      if (!rawPK) {
        toast.error('Invalid registration options');
        return;
      }

      // Convert server options to browser-native PublicKeyCredentialCreationOptions
      const publicKey: PublicKeyCredentialCreationOptions = {
        ...rawPK,
        challenge: b64ToBuffer(rawPK.challenge as unknown as string),
        user: {
          ...rawPK.user,
          id: b64ToBuffer(rawPK.user.id as unknown as string),
        },
        rp: rawPK.rp,
        pubKeyCredParams: (rawPK.pubKeyCredParams ?? []).map((p) => ({
          alg: p.alg,
          type: 'public-key' as const,
        })),
        excludeCredentials: rawPK.excludeCredentials
          ? rawPK.excludeCredentials.map((c) => ({
              id: b64ToBuffer(c.id as unknown as string),
              type: 'public-key' as const,
              transports: (c.transports ?? undefined) as AuthenticatorTransport[] | undefined,
            }))
          : undefined,
        authenticatorSelection: rawPK.authenticatorSelection
          ? {
              ...rawPK.authenticatorSelection,
              authenticatorAttachment:
                (rawPK.authenticatorSelection.authenticatorAttachment as AuthenticatorAttachment) ??
                undefined,
              residentKey:
                (rawPK.authenticatorSelection.residentKey as ResidentKeyRequirement) ?? undefined,
              userVerification:
                (rawPK.authenticatorSelection.userVerification as UserVerificationRequirement) ??
                undefined,
            }
          : undefined,
        attestation: (rawPK.attestation as AttestationConveyancePreference) ?? undefined,
        timeout: rawPK.timeout,
      };

      const credential = (await navigator.credentials.create({
        publicKey,
      })) as PublicKeyCredential | null;
      if (!credential) {
        toast.error('Passkey registration cancelled');
        return;
      }

      const response = credential.response as AuthenticatorAttestationResponse;

      const credentialPayload = {
        id: credential.id,
        rawId: toBase64Url(credential.rawId),
        type: credential.type,
        response: {
          attestationObject: toBase64Url(response.attestationObject),
          clientDataJSON: toBase64Url(response.clientDataJSON),
        },
      };

      // Prompt for a friendly name before finishing
      setPendingPasskeyCredential(credentialPayload);
      setPendingPasskeyChallenge(beginData.challenge);
      setPasskeyName('');
      setPasskeyNameDialogOpen(true);
    } catch (err) {
      if (err instanceof DOMException && err.name === 'NotAllowedError') {
        return;
      }
      toast.error('Passkey registration failed');
    } finally {
      setPasskeyRegisterLoading(false);
    }
  };

  const handlePasskeyFinishRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!pendingPasskeyCredential || !pendingPasskeyChallenge) return;
    try {
      const { error } = await authPasskeyRegisterFinish({
        body: {
          challenge: pendingPasskeyChallenge,
          name: passkeyName || 'My Passkey',
          credential: pendingPasskeyCredential,
        },
      });
      if (error) {
        toast.error('Failed to register passkey');
        return;
      }
      toast.success('Passkey registered');
      setPasskeyNameDialogOpen(false);
      setPendingPasskeyCredential(null);
      setPendingPasskeyChallenge(null);
      setPasskeyName('');
      void refetchPasskeys();
    } catch {
      toast.error('Failed to register passkey');
    }
  };

  // --- Passkey Delete ---

  const handlePasskeyDelete = async (id: string) => {
    setPasskeyDeleteId(id);
    try {
      const { error } = await authPasskeysDelete({ path: { id } });
      if (error) {
        toast.error('Failed to delete passkey');
        return;
      }
      toast.success('Passkey deleted');
      void refetchPasskeys();
    } catch {
      toast.error('Failed to delete passkey');
    } finally {
      setPasskeyDeleteId(null);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text).then(
      () => toast.success('Copied to clipboard'),
      () => toast.error('Failed to copy'),
    );
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return 'Never';
    return new Date(dateStr).toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  return (
    <div className="mx-auto max-w-2xl space-y-6 p-6">
      {/* MFA Status */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <CardTitle className="flex items-center gap-2">
                {mfaEnabled ? (
                  <IconShieldCheck className="size-5 text-green-600" />
                ) : (
                  <IconShieldOff className="size-5 text-muted-foreground" />
                )}
                Multi-Factor Authentication
              </CardTitle>
              <CardDescription>
                Protect your account with additional verification methods.
              </CardDescription>
            </div>
            {!meLoading && (
              <Badge variant={mfaEnabled ? 'default' : 'secondary'}>
                {mfaEnabled ? 'Enabled' : 'Disabled'}
              </Badge>
            )}
          </div>
        </CardHeader>
      </Card>

      {/* TOTP Section */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <CardTitle className="flex items-center gap-2 text-base">
                <IconKey className="size-4" />
                Authenticator App (TOTP)
              </CardTitle>
              <CardDescription>
                Use an authenticator app like Google Authenticator or 1Password to generate
                verification codes.
              </CardDescription>
            </div>
            {totpEnabled && <Badge variant="default">Enabled</Badge>}
          </div>
        </CardHeader>
        <CardContent>
          {totpEnabled ? (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                Your authenticator app is configured and active.
              </p>
              <Button variant="destructive" size="sm" onClick={() => setDisableDialogOpen(true)}>
                Disable
              </Button>
            </div>
          ) : totpSetupUri ? (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Setup URI</Label>
                <p className="text-xs text-muted-foreground">
                  Copy this URI into your authenticator app, or use the secret key below.
                </p>
                <div className="flex gap-2">
                  <Input readOnly value={totpSetupUri} className="font-mono text-xs" />
                  <Button variant="outline" size="sm" onClick={() => copyToClipboard(totpSetupUri)}>
                    Copy
                  </Button>
                </div>
              </div>

              {totpSetupSecret && (
                <div className="space-y-2">
                  <Label>Secret Key</Label>
                  <div className="flex gap-2">
                    <Input readOnly value={totpSetupSecret} className="font-mono text-xs" />
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => copyToClipboard(totpSetupSecret)}
                    >
                      Copy
                    </Button>
                  </div>
                </div>
              )}

              <Separator />

              <form onSubmit={handleTotpConfirm} className="space-y-3">
                <div className="space-y-2">
                  <Label htmlFor="totp-verify-code">Verification Code</Label>
                  <p className="text-xs text-muted-foreground">
                    Enter the 6-digit code from your authenticator app to confirm setup.
                  </p>
                  <Input
                    id="totp-verify-code"
                    type="text"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    placeholder="000000"
                    maxLength={6}
                    value={totpCode}
                    onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, ''))}
                  />
                </div>
                <div className="flex gap-2">
                  <Button type="submit" disabled={totpConfirmLoading || totpCode.length !== 6}>
                    {totpConfirmLoading ? 'Verifying...' : 'Verify & Enable'}
                  </Button>
                  <Button type="button" variant="ghost" onClick={handleTotpCancelSetup}>
                    Cancel
                  </Button>
                </div>
              </form>
            </div>
          ) : (
            <Button onClick={handleTotpSetup} disabled={totpSetupLoading}>
              {totpSetupLoading ? 'Setting up...' : 'Set up authenticator'}
            </Button>
          )}
        </CardContent>
      </Card>

      {/* Passkeys Section */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <CardTitle className="flex items-center gap-2 text-base">
                <IconFingerprint className="size-4" />
                Passkeys
              </CardTitle>
              <CardDescription>
                Sign in with biometrics, security keys, or your device's screen lock.
              </CardDescription>
            </div>
            <Button size="sm" onClick={handlePasskeyRegister} disabled={passkeyRegisterLoading}>
              <IconPlus className="mr-1 size-4" />
              {passkeyRegisterLoading ? 'Registering...' : 'Add passkey'}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {passkeysLoading ? (
            <p className="text-sm text-muted-foreground">Loading passkeys...</p>
          ) : passkeys.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No passkeys registered. Add one to enable passwordless sign-in.
            </p>
          ) : (
            <div className="space-y-3">
              {passkeys.map((pk) => (
                <div
                  key={pk.id}
                  className="flex items-center justify-between rounded-md border p-3"
                >
                  <div className="space-y-1">
                    <p className="text-sm font-medium">{pk.name}</p>
                    <div className="flex gap-3 text-xs text-muted-foreground">
                      <span>Created {formatDate(pk.created_at)}</span>
                      <span>Last used {formatDate(pk.last_used_at)}</span>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handlePasskeyDelete(pk.id)}
                    disabled={passkeyDeleteId === pk.id}
                  >
                    <IconTrash className="size-4 text-destructive" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* TOTP Disable Dialog */}
      <Dialog open={disableDialogOpen} onOpenChange={setDisableDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Disable Authenticator App</DialogTitle>
            <DialogDescription>
              Enter your account password or a current TOTP code to disable two-factor
              authentication.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleTotpDisable} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="disable-password">Password</Label>
              <Input
                id="disable-password"
                type="password"
                autoComplete="current-password"
                placeholder="Enter your password"
                value={disablePassword}
                onChange={(e) => setDisablePassword(e.target.value)}
              />
            </div>
            <div className="relative flex items-center gap-4">
              <Separator className="flex-1" />
              <span className="text-xs text-muted-foreground">OR</span>
              <Separator className="flex-1" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="disable-code">TOTP Code</Label>
              <Input
                id="disable-code"
                type="text"
                inputMode="numeric"
                autoComplete="one-time-code"
                placeholder="000000"
                maxLength={6}
                value={disableCode}
                onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, ''))}
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="ghost" onClick={() => setDisableDialogOpen(false)}>
                Cancel
              </Button>
              <Button
                type="submit"
                variant="destructive"
                disabled={disableLoading || (!disablePassword && !disableCode)}
              >
                {disableLoading ? 'Disabling...' : 'Disable TOTP'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Passkey Name Dialog */}
      <Dialog
        open={passkeyNameDialogOpen}
        onOpenChange={(open) => {
          if (!open) {
            setPasskeyNameDialogOpen(false);
            setPendingPasskeyCredential(null);
            setPendingPasskeyChallenge(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Name Your Passkey</DialogTitle>
            <DialogDescription>
              Give this passkey a friendly name so you can identify it later.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handlePasskeyFinishRegister} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="passkey-name">Passkey Name</Label>
              <Input
                id="passkey-name"
                type="text"
                placeholder="e.g. MacBook Pro, YubiKey"
                value={passkeyName}
                onChange={(e) => setPasskeyName(e.target.value)}
                autoFocus
              />
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  setPasskeyNameDialogOpen(false);
                  setPendingPasskeyCredential(null);
                  setPendingPasskeyChallenge(null);
                }}
              >
                Cancel
              </Button>
              <Button type="submit">Save Passkey</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
};
