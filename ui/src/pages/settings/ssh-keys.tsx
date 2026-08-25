import { useState } from 'react';

import { IconCopy, IconKey, IconPlus, IconTrash } from '@tabler/icons-react';
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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  sshKeysListOptions,
  sshKeysListQueryKey,
  sshKeysMintMutation,
  sshKeysRevokeMutation,
} from '@/sdk/@tanstack/react-query.gen';
import type { KeyView } from '@/sdk/types.gen';

const errorDetail = (err: unknown, fallback: string) =>
  (err as { detail?: string })?.detail ?? fallback;

const when = (iso: string | undefined, absent: string) =>
  iso ? new Date(iso).toLocaleString() : absent;

/** Usable keys first, newest first within each group: the key an operator came
 *  here to copy is one that still works. */
const forDisplay = (keys: KeyView[]) =>
  [...keys].sort((a, b) => {
    if (a.usable !== b.usable) return a.usable ? -1 : 1;
    return b.created_at.localeCompare(a.created_at);
  });

function KeyStatus({ k }: { k: KeyView }) {
  if (k.revoked) return <Badge variant="destructive">Revoked</Badge>;
  if (!k.usable) return <Badge variant="secondary">Unusable</Badge>;
  return <Badge variant="outline">Active</Badge>;
}

function copy(value: string, what: string) {
  void navigator.clipboard.writeText(value);
  toast.success(`${what} copied`);
}

/**
 * Fleet SSH keys: the control plane's own keys for reaching hypervisors.
 *
 * This page exists because the automated onboarding path has a prerequisite that
 * was invisible. The control plane connects to a host with the private half of
 * one of these keys, so the public half has to be in that host's authorized_keys
 * BEFORE anything can be installed on it — and until this page existed the API
 * held the value and nothing showed it, which left "whitelist our key" as a step
 * an operator could not perform.
 *
 * The fingerprint is shown next to it deliberately. It is OpenSSH's own SHA256
 * form, so it compares directly against `ssh-keygen -lf` on the hypervisor, and
 * that comparison is how an operator confirms the key they pasted is the key the
 * control plane actually holds rather than one that arrived some other way.
 */
export function SshKeys() {
  const qc = useQueryClient();
  const [mintOpen, setMintOpen] = useState(false);
  const [name, setName] = useState('');
  const [confirmRevoke, setConfirmRevoke] = useState<KeyView | null>(null);

  const keys = useQuery(sshKeysListOptions());

  const invalidate = () => qc.invalidateQueries({ queryKey: sshKeysListQueryKey() });

  const mint = useMutation({
    ...sshKeysMintMutation(),
    onSuccess: () => {
      toast.success('Key minted');
      setMintOpen(false);
      setName('');
      void invalidate();
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not mint a key')),
  });

  const revoke = useMutation({
    ...sshKeysRevokeMutation(),
    onSuccess: () => {
      toast.success('Key revoked');
      setConfirmRevoke(null);
      void invalidate();
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not revoke the key')),
  });

  const rows = forDisplay(keys.data?.keys ?? []);

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="font-heading text-2xl">Fleet SSH keys</h1>
          <p className="text-sm text-muted-foreground">
            How the control plane reaches your hypervisors. Paste a key's public half into a host's{' '}
            <code className="rounded bg-muted px-1 py-0.5 text-xs">authorized_keys</code> and it can
            be onboarded without anyone logging into it.
          </p>
        </div>
        <Button onClick={() => setMintOpen(true)}>
          <IconPlus /> Mint a key
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <IconKey className="size-4" /> Keys
          </CardTitle>
          <CardDescription>
            The private half never leaves the control plane and no endpoint returns it. Only the
            public half and the fingerprint are shown here, which is everything needed to install
            and verify one.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {keys.isPending ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : rows.length === 0 ? (
            <div className="space-y-2 py-6 text-center">
              <p className="text-sm font-medium">No fleet key yet</p>
              <p className="text-sm text-muted-foreground">
                Without one the control plane cannot connect to a hypervisor at all, so onboarding a
                host has to be done by hand on the host itself.
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Purpose</TableHead>
                  <TableHead>Fingerprint</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((k) => (
                  <TableRow key={k.id}>
                    <TableCell className="font-medium">{k.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{k.purpose}</Badge>
                    </TableCell>
                    <TableCell className="max-w-[22rem] truncate font-mono text-xs">
                      {k.fingerprint}
                    </TableCell>
                    <TableCell>
                      <KeyStatus k={k} />
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {when(k.created_at, '—')}
                    </TableCell>
                    <TableCell className="space-x-1 text-right">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => copy(k.public_key, 'Public key')}
                      >
                        <IconCopy /> Public key
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => copy(k.fingerprint, 'Fingerprint')}
                      >
                        Fingerprint
                      </Button>
                      {k.revoked ? null : (
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => setConfirmRevoke(k)}
                          aria-label={`Revoke ${k.name}`}
                        >
                          <IconTrash />
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {rows.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>Installing a key on a hypervisor</CardTitle>
            <CardDescription>
              Run this on the host, as root, before onboarding it. Then verify the fingerprint
              matches the one above — that is what tells you the key you installed is the key the
              control plane holds.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <code className="block overflow-x-auto rounded bg-muted p-3 text-xs whitespace-pre">
              {`mkdir -p /root/.ssh && chmod 700 /root/.ssh\n` +
                `echo '${rows[0].public_key}' >> /root/.ssh/authorized_keys\n` +
                `chmod 600 /root/.ssh/authorized_keys\n` +
                `ssh-keygen -lf /root/.ssh/authorized_keys | tail -1`}
            </code>
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                copy(
                  `mkdir -p /root/.ssh && chmod 700 /root/.ssh\n` +
                    `echo '${rows[0].public_key}' >> /root/.ssh/authorized_keys\n` +
                    `chmod 600 /root/.ssh/authorized_keys\n` +
                    `ssh-keygen -lf /root/.ssh/authorized_keys | tail -1`,
                  'Command',
                )
              }
            >
              <IconCopy /> Copy command
            </Button>
            <p className="text-xs text-muted-foreground">
              On a Proxmox host already in a cluster,{' '}
              <code className="rounded bg-muted px-1 py-0.5">/root/.ssh/authorized_keys</code> is a
              symlink onto the replicated filesystem, so this writes to every node in that cluster
              at once — and is discarded if the node later joins a different one.
            </p>
          </CardContent>
        </Card>
      ) : null}

      <Dialog open={mintOpen} onOpenChange={setMintOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Mint a fleet key</DialogTitle>
            <DialogDescription>
              A new keypair. The private half is stored encrypted and never returned; the public
              half is what you install on hypervisors.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="key-name">Name</Label>
            <Input
              id="key-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="control plane — production"
            />
            <p className="text-xs text-muted-foreground">
              What this key is for, so it can be recognised in an authorized_keys file a year from
              now.
            </p>
          </div>
          <DialogFooter>
            <Button
              disabled={mint.isPending || name.trim() === ''}
              onClick={() => mint.mutate({ body: { name: name.trim() } })}
            >
              {mint.isPending ? 'Minting…' : 'Mint'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={confirmRevoke !== null} onOpenChange={(o) => !o && setConfirmRevoke(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke {confirmRevoke?.name}?</DialogTitle>
            <DialogDescription>
              The control plane stops using this key immediately. It does NOT remove the public half
              from any host's authorized_keys — that is a separate act on each machine, and until it
              is done the key remains installed while no longer being one this control plane holds.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmRevoke(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={revoke.isPending}
              onClick={() => confirmRevoke && revoke.mutate({ path: { id: confirmRevoke.id } })}
            >
              {revoke.isPending ? 'Revoking…' : 'Revoke'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
