import { useState } from 'react';

import { IconAlertTriangle, IconCopy } from '@tabler/icons-react';
import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiKeysCreateMutation } from '@/sdk/@tanstack/react-query.gen';

// The two scopes the endpoint will mint. 'agent' is deliberately absent and must
// stay absent: an agent credential is issued only by fleet enrolment, which is
// what stops a person forging a hypervisor's identity. Offering it here would
// advertise a scope the API refuses outright.
const SCOPES = [
  {
    value: 'user',
    label: 'Personal — acts as you',
    blurb:
      'Identifies you, not a tenant. You may belong to several organizations, so every request has to name which one it means.',
  },
  {
    value: 'org',
    label: 'Organization — acts as the tenant',
    blurb:
      'Already bound to one organization and carries admin authority inside it. It is not tied to your account, so it keeps working after you leave.',
  },
] as const;

type Scope = (typeof SCOPES)[number]['value'];

const TTLS = [
  { value: 'never', label: 'Never expires', hours: undefined },
  { value: '24', label: '24 hours', hours: 24 },
  { value: '720', label: '30 days', hours: 720 },
  { value: '2160', label: '90 days', hours: 2160 },
  { value: '8760', label: '1 year', hours: 8760 },
] as const;

type Minted = { scope: Scope; name: string; key: string; secret?: string };

/** Copyable line of credential material. Read-only text, never an input, so
 *  nothing here can be edited into a value that was never issued. */
function CopyRow({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="space-y-1">
      <p className="text-xs font-medium">{label}</p>
      <div className="flex items-start gap-2">
        <code className="min-w-0 flex-1 overflow-x-auto rounded bg-muted p-2 text-xs break-all">
          {value}
        </code>
        <Button
          size="xs"
          variant="outline"
          aria-label={`Copy ${label}`}
          onClick={() => {
            void navigator.clipboard.writeText(value);
            toast.success(`${label} copied`);
          }}
        >
          <IconCopy />
        </Button>
      </div>
      {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

export type MintApiKeyDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The organization this session is scoped to, for the worked example. */
  org?: { id: string; name: string };
  /** The caller's role in that organization, or undefined while it is unknown. */
  orgRole?: string;
  onMinted: () => void;
};

export function MintApiKeyDialog({
  open,
  onOpenChange,
  org,
  orgRole,
  onMinted,
}: MintApiKeyDialogProps) {
  const [name, setName] = useState('');
  const [scope, setScope] = useState<Scope>('user');
  const [ttl, setTtl] = useState<string>('never');
  const [minted, setMinted] = useState<Minted | null>(null);

  // Undefined role means the membership has not loaded yet; offering the option
  // and letting the server refuse beats hiding a control from an admin because
  // a second request was slow.
  const orgMintAllowed = orgRole === undefined || orgRole === 'owner' || orgRole === 'admin';
  const scopes = SCOPES.filter((s) => s.value !== 'org' || orgMintAllowed);
  const chosenScope = SCOPES.find((s) => s.value === scope);

  const mint = useMutation({
    ...apiKeysCreateMutation(),
    // Held in state rather than toasted. This is the only time the material
    // exists outside the client's hands: only a hash is stored, and no endpoint
    // reissues it, so a banner that fades after four seconds means minting a
    // second key and revoking the first.
    onSuccess: (res) => {
      setMinted({
        scope: res.credential.scope,
        name: res.credential.name,
        key: res.key,
        secret: res.secret,
      });
      onMinted();
    },
    onError: (e) => toast.error((e as { detail?: string })?.detail ?? 'Could not mint that key'),
  });

  const close = (next: boolean) => {
    if (!next) {
      // React Query keeps the mutation result — including the plaintext — in its
      // cache for gcTime (5 minutes by default) after the dialog is gone. So
      // dropping local state is necessary and NOT sufficient: reset() is what
      // makes the claim in the comment below actually true.
      mint.reset();
      // Dropped on close, not kept for a second look: a closed dialog still
      // holding the plaintext leaves a live credential in the DOM of a tab
      // nobody is watching, and the list it returns to can never show it again.
      setMinted(null);
      setName('');
      setScope('user');
      setTtl('never');
    }
    onOpenChange(next);
  };

  const origin = typeof window === 'undefined' ? '' : window.location.origin;

  const example = minted
    ? minted.scope === 'org'
      ? `curl -H "X-ORG-Key: ${minted.key}" \\\n     -H "X-ORG-Secret: ${minted.secret ?? ''}" \\\n     ${origin}/api/v1/me`
      : `curl -H "X-API-Key: ${minted.key}" \\\n     -H "X-Org-Id: ${org?.id ?? '<organization id>'}" \\\n     ${origin}/api/v1/me`
    : '';

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{minted ? `${minted.name} is ready` : 'Mint an API key'}</DialogTitle>
          <DialogDescription>
            {minted
              ? 'This is the only time the credential is shown. Only a hash of it is stored, so nothing here can reissue or reveal it later — a key that is lost is replaced by minting a new one and revoking this one.'
              : 'A key authenticates a program rather than a person at a browser. Which scope it carries decides whose authority it acts with, and that cannot be changed afterwards.'}
          </DialogDescription>
        </DialogHeader>

        {minted ? (
          <div className="space-y-4">
            <div className="space-y-3 rounded-lg border border-amber-500/40 bg-amber-500/5 p-3">
              <p className="flex items-center gap-2 text-sm font-medium text-amber-600 dark:text-amber-500">
                <IconAlertTriangle className="size-4" /> Copy this now — it is never shown again
              </p>
              {minted.scope === 'org' ? (
                <>
                  <CopyRow
                    label="Key"
                    value={minted.key}
                    hint="Sent as the X-ORG-Key header. Identifying half — not a credential on its own."
                  />
                  <CopyRow
                    label="Secret"
                    value={minted.secret ?? ''}
                    hint="Sent as the X-ORG-Secret header. Treat it like a root password for this organization."
                  />
                </>
              ) : (
                <CopyRow
                  label="Key"
                  value={minted.key}
                  hint="Sent as the X-API-Key header. This single value is the whole credential."
                />
              )}
            </div>

            <div className="space-y-2">
              <p className="text-sm font-medium">A request that uses it</p>
              <code className="block overflow-x-auto rounded bg-muted p-3 text-xs whitespace-pre">
                {example}
              </code>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  void navigator.clipboard.writeText(example);
                  toast.success('Example copied');
                }}
              >
                <IconCopy /> Copy example
              </Button>
              {minted.scope === 'org' ? (
                <p className="text-sm text-muted-foreground">
                  This key is already bound to {org?.name ?? 'this organization'} and ignores
                  X-Org-Id entirely — a tenant-bound credential that could be pointed at another
                  tenant by a header would not be bound to anything.
                </p>
              ) : (
                <p className="text-sm text-muted-foreground">
                  <span className="font-medium">X-Org-Id is required.</span> This key says who you
                  are, not which organization you mean, and you may belong to several. Without that
                  header anything organization-scoped is refused — it is the first request everyone
                  gets wrong. The id above is {org?.name ?? 'the organization this session is in'}.
                </p>
              )}
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="key-name">Name</Label>
              <Input
                id="key-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="ci-deploy-runner"
              />
              <p className="text-xs text-muted-foreground">
                The list shows no key material, so this name is the only thing telling two keys
                apart when one of them has to be revoked in a hurry.
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="key-scope">Scope</Label>
              <Select
                value={scope}
                onValueChange={(v) => {
                  if (v === 'user' || v === 'org') setScope(v);
                }}
              >
                <SelectTrigger id="key-scope" className="w-full">
                  <SelectValue>
                    {(v: string | null) =>
                      SCOPES.find((s) => s.value === v)?.label ?? 'Select a scope'
                    }
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {scopes.map((s) => (
                    <SelectItem key={s.value} value={s.value}>
                      {s.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">{chosenScope?.blurb}</p>
              {orgMintAllowed ? null : (
                <p className="text-xs text-muted-foreground">
                  An organization key is standing admin access to the whole tenant, so only an owner
                  or admin may mint one. You are {orgRole} here.
                </p>
              )}
              <p className="text-xs text-muted-foreground">
                Agent credentials are not minted here. A hypervisor's key comes from fleet
                enrolment, on the host itself, so that nobody can hand themselves a machine's
                identity from a browser.
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="key-ttl">Expires</Label>
              <Select
                value={ttl}
                onValueChange={(v) => {
                  if (typeof v === 'string') setTtl(v);
                }}
              >
                <SelectTrigger id="key-ttl" className="w-full">
                  <SelectValue>
                    {(v: string | null) =>
                      TTLS.find((t) => t.value === v)?.label ?? 'Never expires'
                    }
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {TTLS.map((t) => (
                    <SelectItem key={t.value} value={t.value}>
                      {t.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                An expiry is the one thing that limits a key nobody remembers issuing. Revocation is
                always available and takes effect immediately.
              </p>
            </div>
          </div>
        )}

        <DialogFooter>
          {minted ? (
            <Button onClick={() => close(false)}>I have stored it</Button>
          ) : (
            <Button
              disabled={!name.trim() || mint.isPending}
              onClick={() => {
                const hours = TTLS.find((t) => t.value === ttl)?.hours;
                mint.mutate({
                  body: {
                    name: name.trim(),
                    scope,
                    ...(hours === undefined ? {} : { ttl_hours: hours }),
                  },
                });
              }}
            >
              {mint.isPending ? 'Minting…' : 'Mint key'}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
