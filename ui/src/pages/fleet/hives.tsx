import { useState } from 'react';

import { IconPlus, IconRefresh, IconTrash } from '@tabler/icons-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { ConfirmDestructive } from '@/components/confirm-destructive';
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { EstateView, HiveDetailView } from '@/sdk';
import {
  estatesListOptions,
  fleetHivesCreateMutation,
  fleetHivesDeleteMutation,
  fleetHivesListOptions,
  fleetHivesListQueryKey,
  fleetHivesSyncMutation,
  fleetHivesUpdateMutation,
} from '@/sdk/@tanstack/react-query.gen';

import { Hint, LoadError } from './drain';

/** Mirrors the server's default. Shown, not enforced here — the server decides. */
const DEFAULT_MINIMUM_NODES = 3;

function HiveCard({ cluster, estates }: { cluster: HiveDetailView; estates: EstateView[] }) {
  const queryClient = useQueryClient();
  const [apiUrl, setApiUrl] = useState(cluster.api_url ?? '');
  const [credentialId, setCredentialId] = useState(cluster.credential_id ?? '');
  const [minimumNodes, setMinimumNodes] = useState(String(cluster.minimum_nodes || ''));
  const [insecure, setInsecure] = useState(cluster.insecure_skip_verify);
  const [overcommit, setOvercommit] = useState(String(cluster.cpu_overcommit_ratio ?? 1));
  const [description, setDescription] = useState(cluster.description ?? '');
  const [name, setName] = useState(cluster.name);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const update = useMutation({
    ...fleetHivesUpdateMutation(),
    onSuccess: () => {
      toast.success('Cluster updated');
      void queryClient.invalidateQueries({ queryKey: fleetHivesListQueryKey() });
    },
    onError: () => toast.error('Could not update this cluster'),
  });

  const sync = useMutation({
    ...fleetHivesSyncMutation(),
    onSuccess: () => toast.success('Capacity sync queued'),
    onError: () => toast.error('Could not queue a sync'),
  });

  const remove = useMutation({
    ...fleetHivesDeleteMutation(),
    onSuccess: () => {
      toast.success('Cluster deleted');
      setDeleteOpen(false);
      void queryClient.invalidateQueries({ queryKey: fleetHivesListQueryKey() });
    },
    onError: (e) =>
      toast.error((e as { detail?: string })?.detail ?? 'Could not delete this cluster'),
  });

  // The server refuses while the cluster still holds hypervisors. Disabling the
  // button here is not a second gate — it is saying why in advance, instead of
  // spending a round trip to be told.
  const holdsNodes = cluster.node_count > 0;

  // Radix resolves a trigger's label from the item that is MOUNTED with the
  // matching value, so before the credentials query resolves — which is exactly
  // the first render of a cluster that already has one bound — there is no item
  // to match and the raw uuid is shown. A uuid is not a slower way of saying the
  // name; it is a different fact, and not one anybody picked a credential by.
  const boundCredential = estates.find((e) => e.credential_id === credentialId) ?? null;
  const credentialLabel = () => {
    if (!credentialId) return estates.length ? 'Choose a credential' : 'No credentials yet';
    if (!boundCredential) return 'A credential that is no longer registered';
    return `${boundCredential.credential_name || boundCredential.name} — ${boundCredential.fingerprint.slice(0, 24)}…`;
  };

  // A cluster with no nodes has not been created in Proxmox yet, so its name is
  // still what `pvecm create` will be given. Once it has nodes the name is the
  // one corosync already uses, and changing it here only makes the two disagree.
  const nameIsTheClusterName = cluster.node_count === 0;
  const renamed = name.trim() !== cluster.name;

  const parsedMin = minimumNodes.trim() === '' ? undefined : Number(minimumNodes);
  const belowDefault = parsedMin !== undefined && parsedMin < DEFAULT_MINIMUM_NODES;

  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-4 space-y-0">
        <div>
          <CardTitle className="text-base">
            {cluster.name}{' '}
            {cluster.configured ? (
              <Badge className="bg-emerald-500/15 text-emerald-500">syncable</Badge>
            ) : (
              <Badge className="bg-amber-500/15 text-amber-500">not configured</Badge>
            )}
          </CardTitle>
          <CardDescription>
            {cluster.provider} · {cluster.node_count} node{cluster.node_count === 1 ? '' : 's'} ·
            departures keep at least {cluster.effective_minimum_nodes}
          </CardDescription>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={!cluster.configured || sync.isPending}
            onClick={() => sync.mutate({ path: { id: cluster.id } })}
          >
            <IconRefresh className="size-4" />
            Sync now
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="text-destructive hover:text-destructive"
            disabled={holdsNodes}
            aria-label={`Delete ${cluster.name}`}
            title={
              holdsNodes
                ? 'Depart its hypervisors first — each one keeps its identity and becomes fleetless.'
                : undefined
            }
            onClick={() => setDeleteOpen(true)}
          >
            <IconTrash className="size-4" />
            Delete
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {holdsNodes ? (
          <Hint>
            This cluster cannot be deleted while it holds {cluster.node_count} hypervisor
            {cluster.node_count === 1 ? '' : 's'}. Depart them first — each keeps its machine
            identity and onboarding history, and becomes a hypervisor in no cluster.
          </Hint>
        ) : null}

        {!cluster.configured ? (
          <Hint>
            Without an API URL and a credential this cluster is never synced, so every node in it
            reads as never-observed and every departure refuses.
          </Hint>
        ) : null}

        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor={`api-${cluster.id}`}>API URL</Label>
            <Input
              id={`api-${cluster.id}`}
              value={apiUrl}
              placeholder="https://10.0.0.1:8006/api2/json"
              onChange={(e) => setApiUrl(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor={`cred-${cluster.id}`}>Credential</Label>
            <Select value={credentialId} onValueChange={(v) => setCredentialId(v ?? '')}>
              <SelectTrigger id={`cred-${cluster.id}`}>
                <SelectValue>{credentialLabel()}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {estates.map((e) => (
                  <SelectItem key={e.credential_id} value={e.credential_id}>
                    {e.credential_name || e.name} — {e.fingerprint.slice(0, 24)}…
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {estates.length === 0 ? (
              <p className="text-xs text-muted-foreground">
                Register a Proxmox estate first; its credential is what reads this cluster's
                capacity.
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">
                Compare the fingerprint against your Proxmox console to confirm which token is bound
                — it identifies the credential without disclosing it.
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor={`min-${cluster.id}`}>Minimum nodes to keep</Label>
            <Input
              id={`min-${cluster.id}`}
              type="number"
              min={1}
              value={minimumNodes}
              placeholder={String(DEFAULT_MINIMUM_NODES)}
              onChange={(e) => setMinimumNodes(e.target.value)}
            />
            {belowDefault ? (
              <p className="text-xs text-amber-500">
                Below {DEFAULT_MINIMUM_NODES}, a departure can leave a cluster that loses quorum the
                moment one more node fails, and departing from two leaves the survivor read-only
                with its guests unmanageable. Only lower this for a test fleet.
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">
                A departure is refused if it would leave fewer nodes than this, counting departures
                already under way as gone. Blank means the default of {DEFAULT_MINIMUM_NODES}.
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor={`overcommit-${cluster.id}`}>CPU overcommit ratio</Label>
            <Input
              id={`overcommit-${cluster.id}`}
              type="number"
              min={1}
              step="0.1"
              value={overcommit}
              onChange={(e) => setOvercommit(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Multiplies each node's cores when deciding what fits. 1 means no overcommit. RAM and
              disk are never overcommitted: overcommitting CPU degrades guests, overcommitting
              memory kills them.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor={`name-${cluster.id}`}>Name</Label>
            <Input
              id={`name-${cluster.id}`}
              value={name}
              placeholder="lab-a"
              autoComplete="off"
              spellCheck={false}
              onChange={(e) => setName(e.target.value)}
            />
            {nameIsTheClusterName ? (
              <p className="text-xs text-muted-foreground">
                This cluster has no nodes yet, so this is also what the Proxmox cluster will be
                called when it is created. Letters, digits and hyphens, starting and ending
                alphanumeric — a space is refused, and renaming a Proxmox cluster afterwards is not
                something you want to do.
              </p>
            ) : (
              <p className="text-xs text-amber-500">
                The Proxmox cluster is already called {cluster.name} and renaming here does not
                change that. Only the label on this page moves.
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor={`desc-${cluster.id}`}>Description</Label>
            <Input
              id={`desc-${cluster.id}`}
              value={description}
              placeholder="What this cluster is for"
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor={`insecure-${cluster.id}`}>TLS</Label>
            <label className="flex items-start gap-2 text-sm" htmlFor={`insecure-${cluster.id}`}>
              <input
                id={`insecure-${cluster.id}`}
                type="checkbox"
                className="mt-1"
                checked={insecure}
                onChange={(e) => setInsecure(e.target.checked)}
              />
              <span>
                Accept a self-signed certificate
                <span className="block text-xs text-muted-foreground">
                  The sealed API token is handed to whatever answers on that address. Per cluster,
                  so turning it on for a lab does not weaken a production cluster.
                </span>
              </span>
            </label>
          </div>
        </div>

        <div className="flex justify-end">
          <Button
            size="sm"
            disabled={update.isPending}
            onClick={() =>
              update.mutate({
                path: { id: cluster.id },
                body: {
                  ...(renamed ? { name: name.trim() } : {}),
                  api_url: apiUrl,
                  credential_id: credentialId,
                  insecure_skip_verify: insecure,
                  description,
                  ...(Number.isFinite(Number(overcommit)) && Number(overcommit) >= 1
                    ? { cpu_overcommit_ratio: Number(overcommit) }
                    : {}),
                  ...(parsedMin !== undefined && Number.isFinite(parsedMin)
                    ? { minimum_nodes: parsedMin }
                    : {}),
                },
              })
            }
          >
            {update.isPending ? 'Saving…' : 'Save'}
          </Button>
        </div>
      </CardContent>

      <ConfirmDestructive
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={`Delete ${cluster.name}?`}
        description="Only the cluster record here is removed. Nothing is done to Proxmox."
        consequences={[
          'No hypervisor is touched — this cluster holds none, which is why the action is available at all.',
          'The API URL and the credential binding are forgotten. The credential itself is not revoked and stays registered as an estate.',
          'Nothing about the real Proxmox cluster changes; if one still exists at that address, creating a cluster here again and re-binding is how it comes back.',
        ]}
        confirmLabel="Delete this cluster"
        pendingLabel="Deleting…"
        pending={remove.isPending}
        onConfirm={() => remove.mutate({ path: { id: cluster.id } })}
      />
    </Card>
  );
}

export const Clusters = () => {
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [newName, setNewName] = useState('');

  const hives = useQuery(fleetHivesListOptions());
  // Proxmox estates only. The credential bound to a cluster is what reads that
  // cluster's API, so offering an AWS credential here would be offering
  // something that can only fail.
  const estatesQuery = useQuery(estatesListOptions());
  const estates = (estatesQuery.data?.estates ?? []).filter((e) => e.provider === 'proxmox');

  const create = useMutation({
    ...fleetHivesCreateMutation(),
    onSuccess: () => {
      toast.success('Cluster created');
      setCreateOpen(false);
      setNewName('');
      void queryClient.invalidateQueries({ queryKey: fleetHivesListQueryKey() });
    },
    onError: () => toast.error('Could not create the cluster'),
  });

  const list = hives.data?.hives ?? [];

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Clusters</h1>
          <p className="text-sm text-muted-foreground">
            One Proxmox cluster each. Point one at an API and bind a credential, and its capacity is
            read on a schedule — which is what the drain and departure checks decide against.
          </p>
        </div>
        <Button size="sm" onClick={() => setCreateOpen(true)}>
          <IconPlus className="size-4" />
          New cluster
        </Button>
      </div>

      {hives.isError ? (
        <LoadError what="Clusters" error={hives.error} />
      ) : hives.isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : list.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">No clusters yet</CardTitle>
            <CardDescription>
              Creating a cluster here only names it — hypervisors join one at a time, and a
              hypervisor belonging to none is a normal state rather than a fault.
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        list.map((h) => (
          /* Keyed on the SERVER's values, not just the id. When the stored cluster
             changes — this form saving, another operator editing, a sync — the
             key changes and the card remounts with fresh state. That resets the
             form from the server without an effect that writes state during
             render, and it leaves typing undisturbed while the values are
             unchanged. */
          <HiveCard
            key={`${h.id}:${h.name}:${h.api_url ?? ''}:${h.credential_id ?? ''}:${h.minimum_nodes}:${h.insecure_skip_verify}:${h.cpu_overcommit_ratio}:${h.description ?? ''}`}
            cluster={h}
            estates={estates}
          />
        ))
      )}

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New cluster</DialogTitle>
            <DialogDescription>
              Naming it is all this does — point it at an API and bind a credential afterwards,
              which is what makes its capacity readable.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="new-cluster-name">Name</Label>
            <Input
              id="new-cluster-name"
              value={newName}
              placeholder="dc1"
              onChange={(e) => setNewName(e.target.value)}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              Cancel
            </Button>
            <Button
              disabled={!newName.trim() || create.isPending}
              onClick={() => create.mutate({ body: { name: newName.trim() } })}
            >
              {create.isPending ? 'Creating…' : 'Create'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};
