import { useCallback, useEffect, useState } from 'react';

import { Badge } from '@/components/ui/badge.tsx';
import { Button } from '@/components/ui/button.tsx';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card.tsx';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog.tsx';
import { Input } from '@/components/ui/input.tsx';
import { Label } from '@/components/ui/label.tsx';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select.tsx';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table.tsx';
import { estatesCreate, estatesDelete, estatesList, estatesRotateCredential } from '@/sdk';
import type { EstateView } from '@/sdk';
import { client } from '@/sdk/client.gen.ts';

/** The scope keys each provider accepts, mirroring repo.DefaultProviderScopes.
 *
 * The server is the authority — it refuses unknown keys and a scope missing a
 * required or derivable one. This exists so the form asks for the right fields
 * rather than offering a free-text blob the API will reject. */
const PROVIDER_SCOPES: Record<string, { key: string; label: string; hint?: string }[]> = {
  aws: [
    {
      key: 'account',
      label: 'Account ID',
      hint: '12 digits. Derivable from the key later; required for now.',
    },
    { key: 'region', label: 'Region', hint: 'e.g. eu-west-1' },
    { key: 'vpc', label: 'VPC', hint: 'optional' },
  ],
  proxmox: [{ key: 'cluster', label: 'Cluster', hint: 'e.g. dc1' }],
  hetzner: [{ key: 'project', label: 'Project' }],
};

/** The identifying component differs per provider, and only this is
 * fingerprinted — never the secret. */
const IDENTITY_LABEL: Record<string, string> = {
  aws: 'Access key ID',
  proxmox: 'Token ID',
  hetzner: 'Token name',
};

/** A worked example of the identifying half, per provider.
 *
 * The register form had none, and the rotate dialog hard-coded the Proxmox one —
 * so an AWS credential was offered `root@pam!apiary` as its example. Both read
 * from here now.
 *
 * This is not decoration. A Proxmox token id is `user@realm!tokenname`, a shape
 * nobody guesses, and a credential stored without it authenticates against
 * nothing — the capacity sync fails with "the API token id is unknown and the
 * secret alone cannot authenticate", which is a long way from the form that
 * accepted it. */
const IDENTITY_PLACEHOLDER: Record<string, string> = {
  aws: 'AKIAIOSFODNN7EXAMPLE',
  proxmox: 'root@pam!apiary',
  hetzner: 'apiary-readonly',
};

/** Why the identifying half is asked for and stored, said once.
 *
 * Both forms write the same value to the same column, so they say the same
 * thing. They did not: the register form said only "Not secret. This is what
 * gets fingerprinted", which explains what happens to the value and not why it
 * is required — and a field whose purpose reads as bookkeeping is one people
 * leave empty. */
const IDENTITY_HINT =
  'Not secret, and it is stored: the provider API needs both halves. A credential ' +
  'saved without it cannot authenticate at all, whatever the secret is.';

const SECRET_LABEL: Record<string, string> = {
  aws: 'Secret access key',
  proxmox: 'Token secret',
  hetzner: 'Token value',
};

function useBearerToken() {
  useEffect(() => {
    // The generated client points at its own dev origin and carries no auth.
    // Serve the UI from the API and both problems go away.
    client.setConfig({
      baseUrl: window.location.origin,
      // Every credential operation declares bearer security, so the generated
      // client asks for the token here rather than each call passing a header.
      // Read at call time, not at mount: a token refreshed mid-session must be
      // picked up without a reload.
      auth: () => localStorage.getItem('access_token') ?? undefined,
    });
  }, []);
}

export function EstatesPage() {
  useBearerToken();

  const [credentials, setCredentials] = useState<EstateView[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [rotating, setRotating] = useState<EstateView | null>(null);
  const [rotateIdentity, setRotateIdentity] = useState('');
  const [rotateSecret, setRotateSecret] = useState('');

  const [provider, setProvider] = useState<string>('aws');
  const [name, setName] = useState('');
  const [identity, setIdentity] = useState('');
  const [secret, setSecret] = useState('');
  const [scope, setScope] = useState<Record<string, string>>({});

  const refresh = useCallback(async () => {
    setLoading(true);
    const res = await estatesList();
    if (res.error) {
      setError(describe(res.error));
    } else {
      setError(null);
      setCredentials(res.data?.estates ?? []);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    const res = await estatesCreate({
      body: {
        name,
        provider: provider as 'aws' | 'proxmox' | 'hetzner',
        scope: Object.fromEntries(Object.entries(scope).filter(([, v]) => v.trim() !== '')),
        credential_name: name,
        credential_kind: 'static_secret',
        identity,
        secret,
      },
    });
    setBusy(false);

    // Clear the secret first and unconditionally — it must not linger in state
    // once it has been sent, whatever else happens next.
    //
    // BEFORE the error check, deliberately. This used to sit below the early
    // return, so any refusal — a 422 for a bad scope key, a 409, a 503 — left a
    // live provider credential in React state and in a live DOM input for the
    // rest of the session, which is exactly the case an operator hits most. The
    // secret has already been sent by this point, so keeping it buys nothing.
    setSecret('');

    if (res.error) {
      setError(describe(res.error));
      return;
    }
    setIdentity('');
    setName('');
    setScope({});
    setError(null);
    void refresh();
  }

  async function onRotate(e: React.FormEvent) {
    e.preventDefault();
    if (!rotating) return;
    setBusy(true);
    const res = await estatesRotateCredential({
      path: { id: rotating.id },
      body: { identity: rotateIdentity, secret: rotateSecret },
    });
    setBusy(false);

    // Cleared first and unconditionally, before the error branch — the same rule
    // the create form follows. A secret that survives a failed request sits in
    // component state until something else happens to clear it.
    setRotateSecret('');

    if (res.error) {
      setError(describe(res.error));
      return;
    }
    setRotating(null);
    setRotateIdentity('');
    setError(null);
    void refresh();
  }

  async function onDelete(id: string) {
    setBusy(true);
    const res = await estatesDelete({ path: { id } });
    setBusy(false);
    if (res.error) setError(describe(res.error));
    else void refresh();
  }

  const fields = PROVIDER_SCOPES[provider] ?? [];

  return (
    <div className="mx-auto max-w-5xl space-y-8 p-8">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Credentials</h1>
        <p className="text-sm text-muted-foreground">
          A credential and what it is FOR — an AWS account and region, a Proxmox cluster. The scope
          decides where anything created with it ends up, so the same secret against two accounts is
          two credentials here.
        </p>
      </header>

      {error && (
        <div
          role="alert"
          className="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
        >
          {error}
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Register a credential</CardTitle>
          <CardDescription>
            The credential is sealed before it is stored and is never returned by any endpoint. You
            will see its fingerprint instead — compare that against your provider console to confirm
            which credential is installed.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onCreate} className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="provider">Provider</Label>
              <Select
                value={provider}
                onValueChange={(v) => {
                  if (!v) return;
                  setProvider(v);
                  setScope({});
                }}
              >
                <SelectTrigger id="provider">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.keys(PROVIDER_SCOPES).map((p) => (
                    <SelectItem key={p} value={p}>
                      {p}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="prod-eu-west-1"
                required
              />
            </div>

            {fields.map((f) => (
              <div key={f.key} className="grid gap-2">
                <Label htmlFor={`scope-${f.key}`}>{f.label}</Label>
                <Input
                  id={`scope-${f.key}`}
                  value={scope[f.key] ?? ''}
                  onChange={(e) => setScope({ ...scope, [f.key]: e.target.value })}
                />
                {f.hint && <p className="text-xs text-muted-foreground">{f.hint}</p>}
              </div>
            ))}

            <div className="grid gap-2">
              <Label htmlFor="identity">{IDENTITY_LABEL[provider] ?? 'Identity'}</Label>
              <Input
                id="identity"
                value={identity}
                onChange={(e) => setIdentity(e.target.value)}
                placeholder={IDENTITY_PLACEHOLDER[provider]}
                required
              />
              <p className="text-xs text-muted-foreground">{IDENTITY_HINT}</p>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="secret">{SECRET_LABEL[provider] ?? 'Secret'}</Label>
              <Input
                id="secret"
                type="password"
                value={secret}
                onChange={(e) => setSecret(e.target.value)}
                autoComplete="off"
                required
              />
              <p className="text-xs text-muted-foreground">Sealed at rest. Never shown again.</p>
            </div>

            <div className="sm:col-span-2">
              <Button type="submit" disabled={busy}>
                {busy ? 'Registering…' : 'Register credential'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Registered credentials</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : credentials.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No credentials yet. Register one above to get started.
            </p>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Provider</TableHead>
                    <TableHead>Scope</TableHead>
                    <TableHead>Credential</TableHead>
                    <TableHead className="sr-only">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {credentials.map((e) => (
                    <TableRow key={e.id}>
                      <TableCell className="font-medium">{e.name}</TableCell>
                      <TableCell>
                        <Badge variant="secondary">{e.provider}</Badge>
                      </TableCell>
                      <TableCell className="font-mono text-xs">
                        {Object.entries(e.scope ?? {})
                          .map(([k, v]) => `${k}=${v}`)
                          .join(' ')}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col gap-1">
                          <Badge variant="outline" className="w-fit">
                            {e.credential_kind}
                          </Badge>
                          {/* Rendered whole, deliberately. A truncated fingerprint
                              makes every human comparison a weak one. */}
                          <code className="text-xs break-all text-muted-foreground">
                            {e.fingerprint}
                          </code>
                        </div>
                      </TableCell>
                      <TableCell className="space-x-1 text-right">
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={busy}
                          onClick={() => {
                            setRotating(e);
                            setRotateIdentity('');
                            setRotateSecret('');
                            setError(null);
                          }}
                        >
                          Rotate
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={busy}
                          onClick={() => void onDelete(e.id)}
                        >
                          Delete
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={rotating !== null} onOpenChange={(open) => !open && setRotating(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rotate credential</DialogTitle>
            <DialogDescription>
              Replaces the sealed material for {rotating?.name} in place, keeping the credential and
              everything bound to it. The fingerprint changes — that is how the rotation is
              confirmed.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={(e) => void onRotate(e)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="rotate-identity">
                {(rotating && IDENTITY_LABEL[rotating.provider]) ?? 'Identity'}
              </Label>
              <Input
                id="rotate-identity"
                value={rotateIdentity}
                required
                placeholder={rotating ? IDENTITY_PLACEHOLDER[rotating.provider] : undefined}
                onChange={(e) => setRotateIdentity(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                {IDENTITY_HINT} A rotation that changed only the secret would leave the two halves
                belonging to different credentials.
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="rotate-secret">Secret</Label>
              <Input
                id="rotate-secret"
                type="password"
                value={rotateSecret}
                required
                autoComplete="off"
                onChange={(e) => setRotateSecret(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Sealed at rest and never returned by any endpoint.
              </p>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setRotating(null)}>
                Cancel
              </Button>
              <Button type="submit" disabled={busy || !rotateIdentity || !rotateSecret}>
                {busy ? 'Rotating…' : 'Rotate'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/** Surface the server's own message where it has one — the API deliberately
 * names the offending scope key, which is what an operator needs to fix it. */
function describe(err: unknown): string {
  if (typeof err === 'object' && err !== null) {
    const e = err as { detail?: string; title?: string };
    if (e.detail) return e.detail;
    if (e.title) return e.title;
  }
  return 'Request failed';
}
