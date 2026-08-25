import { useState } from 'react';

import { IconPlus, IconRefresh } from '@tabler/icons-react';
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
            departures need at least {cluster.effective_minimum_nodes}
          </CardDescription>
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled={!cluster.configured || sync.isPending}
          onClick={() => sync.mutate({ path: { id: cluster.id } })}
        >
          <IconRefresh className="size-4" />
          Sync now
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
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
                <SelectValue
                  placeholder={estates.length ? 'Choose a credential' : 'No credentials yet'}
                />
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
            <Label htmlFor={`min-${cluster.id}`}>Minimum nodes for a departure</Label>
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
                Below {DEFAULT_MINIMUM_NODES}, removing a node loses quorum part-way through and
                leaves the remaining nodes read-only with their guests unmanageable. Only lower this
                for a test fleet.
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">
                Blank means the default of {DEFAULT_MINIMUM_NODES}.
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
            key={`${h.id}:${h.api_url ?? ''}:${h.credential_id ?? ''}:${h.minimum_nodes}:${h.insecure_skip_verify}:${h.cpu_overcommit_ratio}:${h.description ?? ''}`}
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
