import { useState } from 'react';

import { IconBuilding, IconPlus, IconUser } from '@tabler/icons-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  apiKeysListOptions,
  apiKeysListQueryKey,
  apiKeysRevokeMutation,
  meGetOptions,
  meOrgsListOptions,
} from '@/sdk/@tanstack/react-query.gen';
import type { ApiKeyView } from '@/sdk/types.gen';

import { MintApiKeyDialog } from './api-keys-mint-dialog';

const errorDetail = (err: unknown, fallback: string) =>
  (err as { detail?: string })?.detail ?? fallback;

const when = (iso: string | undefined, absent: string) =>
  iso ? new Date(iso).toLocaleString() : absent;

const isExpired = (k: ApiKeyView) =>
  k.expires_at !== undefined && new Date(k.expires_at) <= new Date();

/** Live keys first, newest first within each group: the ones that can still
 *  authenticate a request are the ones an operator came here to act on. */
const forDisplay = (keys: ApiKeyView[]) =>
  [...keys].sort((a, b) => {
    const aDead = a.revoked || isExpired(a);
    const bDead = b.revoked || isExpired(b);
    if (aDead !== bDead) return aDead ? 1 : -1;
    return b.created_at.localeCompare(a.created_at);
  });

function KeyStatus({ k }: { k: ApiKeyView }) {
  if (k.revoked) return <Badge variant="destructive">Revoked</Badge>;
  if (isExpired(k)) return <Badge variant="secondary">Expired</Badge>;
  return <Badge variant="outline">Active</Badge>;
}

function KeyTable({
  keys,
  canRevoke,
  onRevoke,
  busy,
}: {
  keys: ApiKeyView[];
  canRevoke: boolean;
  onRevoke: (k: ApiKeyView) => void;
  busy: boolean;
}) {
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Prefix</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Created</TableHead>
            <TableHead>Expires</TableHead>
            <TableHead>Last used</TableHead>
            <TableHead className="sr-only">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {forDisplay(keys).map((k) => (
            <TableRow key={k.id}>
              <TableCell className="font-medium">{k.name}</TableCell>
              <TableCell className="text-muted-foreground">
                {k.prefix ? (
                  <code className="rounded bg-muted px-1 py-0.5 text-xs">{k.prefix}…</code>
                ) : (
                  '—'
                )}
              </TableCell>
              <TableCell>
                <KeyStatus k={k} />
              </TableCell>
              <TableCell className="text-muted-foreground">{when(k.created_at, '—')}</TableCell>
              <TableCell className="text-muted-foreground">{when(k.expires_at, 'Never')}</TableCell>
              <TableCell className="text-muted-foreground">
                {when(k.last_used_at, 'Never used')}
              </TableCell>
              <TableCell>
                <div className="flex justify-end gap-2">
                  {k.revoked ? null : (
                    <Button
                      size="xs"
                      variant="outline"
                      disabled={!canRevoke || busy}
                      onClick={() => onRevoke(k)}
                    >
                      Revoke
                    </Button>
                  )}
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

export const ApiKeys = () => {
  const queryClient = useQueryClient();
  const [mintOpen, setMintOpen] = useState(false);
  // The key itself, not a boolean: the confirmation has to name what it destroys,
  // and a prefix is the only other thing distinguishing two keys called "ci".
  const [revoking, setRevoking] = useState<ApiKeyView | null>(null);

  const { data, isLoading, isError, error } = useQuery({ ...apiKeysListOptions() });
  const { data: me } = useQuery({ ...meGetOptions() });
  const { data: myOrgs } = useQuery({ ...meOrgsListOptions() });

  // The ORG role lives on the membership; me.role is the PLATFORM role and says
  // nothing about authority inside this tenant. Minting and revoking an org key
  // are admin-only, so both gate on the membership for the org this session is
  // scoped to.
  const membership = (myOrgs?.organizations ?? []).find((o) => o.id === me?.organization?.id);
  const orgRole = membership?.role;
  const canAdminOrgKeys = orgRole === undefined || orgRole === 'owner' || orgRole === 'admin';

  const refresh = () => queryClient.invalidateQueries({ queryKey: apiKeysListQueryKey() });

  const revoke = useMutation({
    ...apiKeysRevokeMutation(),
    onSuccess: () => {
      toast.success('Key revoked');
      setRevoking(null);
      refresh();
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not revoke that key')),
  });

  const keys = data?.keys ?? [];
  const userKeys = keys.filter((k) => k.scope === 'user');
  const orgKeys = keys.filter((k) => k.scope === 'org');
  const orgName = me?.organization?.name ?? 'this organization';

  return (
    <div className="w-full space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-heading text-2xl">API keys</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Credentials for programs rather than people. A key is shown once when it is minted and
            only its hash is kept, so nothing on this page can recover one — a lost key is revoked
            and replaced. Agent credentials are absent by design: a hypervisor's key is issued only
            by fleet enrolment, on the host itself.
          </p>
        </div>
        <Button size="sm" onClick={() => setMintOpen(true)}>
          <IconPlus /> Mint a key
        </Button>
      </div>

      {isError ? (
        // An empty list and an unreachable server look identical otherwise, and
        // one of them means an operator concludes there is no standing access to
        // this tenant when there may be plenty.
        <Card>
          <CardHeader>
            <CardTitle>Keys could not be loaded</CardTitle>
            <CardDescription>
              {errorDetail(
                error,
                'The server did not answer. Nothing below reflects the credentials that can reach this organization.',
              )}
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <IconUser className="size-4" /> Your keys
              </CardTitle>
              <CardDescription>
                {isLoading
                  ? 'Loading…'
                  : `${userKeys.length} personal key${userKeys.length === 1 ? '' : 's'}. A personal key acts as you and can do nothing you cannot. It carries no tenant, so every request has to name one in an X-Org-Id header — you may belong to several organizations, and the key does not choose between them.`}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <p className="text-sm text-muted-foreground">Loading…</p>
              ) : userKeys.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  Nothing is authenticating as you outside the browser. Mint a personal key for a
                  script or a CI job that should act with your identity and lose its access when
                  yours is removed. Only you can see or revoke your keys.
                </p>
              ) : (
                <KeyTable
                  keys={userKeys}
                  canRevoke
                  onRevoke={setRevoking}
                  busy={revoke.isPending}
                />
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <IconBuilding className="size-4" /> Organization keys
              </CardTitle>
              <CardDescription>
                {isLoading
                  ? 'Loading…'
                  : `${orgKeys.length} key${orgKeys.length === 1 ? '' : 's'} for ${orgName}. These act as the organization itself, with admin authority inside it, and are already bound to it — they ignore X-Org-Id and survive the departure of whoever minted them. Every member can see them; ${
                      canAdminOrgKeys
                        ? 'only an owner or admin can mint or revoke one.'
                        : `only an owner or admin can mint or revoke one, and you are ${orgRole} here.`
                    }`}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <p className="text-sm text-muted-foreground">Loading…</p>
              ) : orgKeys.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  {orgName} has no standing credential of its own. Mint one for a service that must
                  keep working after any particular person leaves — a deploy pipeline or a
                  provisioning job. It is admin access to the whole tenant, so treat it accordingly
                  and give it an expiry.
                </p>
              ) : (
                <KeyTable
                  keys={orgKeys}
                  canRevoke={canAdminOrgKeys}
                  onRevoke={setRevoking}
                  busy={revoke.isPending}
                />
              )}
            </CardContent>
          </Card>
        </>
      )}

      <MintApiKeyDialog
        open={mintOpen}
        onOpenChange={setMintOpen}
        org={me?.organization}
        orgRole={orgRole}
        onMinted={refresh}
      />

      <Dialog open={revoking !== null} onOpenChange={(o) => !o && setRevoking(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              Revoke {revoking?.name}
              {revoking?.prefix ? ` (${revoking.prefix}…)` : ''}?
            </DialogTitle>
            <DialogDescription>
              {revoking?.scope === 'org'
                ? `This key acts as ${orgName} itself. Revocation is permanent and takes effect on the next request it makes, so anything still using it — a pipeline, a provisioning job — starts failing immediately. It cannot be restored; a replacement is a new key.`
                : 'This key acts as you. Revocation is permanent and takes effect on the next request it makes, so anything still using it starts failing immediately. It cannot be restored; a replacement is a new key.'}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="destructive"
              disabled={revoke.isPending}
              onClick={() => {
                if (!revoking) return;
                revoke.mutate({ path: { id: revoking.id } });
              }}
            >
              {revoke.isPending ? 'Revoking…' : `Revoke ${revoking?.name ?? 'this key'}`}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};
