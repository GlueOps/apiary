import { useState } from 'react';

import { IconKey, IconPlus, IconTerminal2 } from '@tabler/icons-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
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
import type { BeeView, CreateTaskRequestBodyWritable } from '@/sdk';
import {
  fleetBeesListOptions,
  fleetBeesListQueryKey,
  fleetHivesListOptions,
  fleetHypervisorsListOptions,
  fleetTasksCreateMutation,
} from '@/sdk/@tanstack/react-query.gen';

import { BeeTable, beeLabel, hiveIdOf } from './bee-table';
import { EnrolmentDialog } from './enrolment-dialog';
import { TaskLogs } from './task-logs';

type TaskKind = CreateTaskRequestBodyWritable['kind'];

/** One cluster and the enrolled hypervisors reporting membership of it. */
type ClusterGroup = { id: string; name: string; bees: BeeView[] };

export const Fleet = () => {
  const queryClient = useQueryClient();
  const [enrolOpen, setEnrolOpen] = useState(false);
  const [watching, setWatching] = useState<{
    taskId: string;
    hostname: string;
    kind: TaskKind;
  } | null>(null);
  const [upgradeFor, setUpgradeFor] = useState<BeeView | null>(null);
  const [allowDowntime, setAllowDowntime] = useState(false);
  const [joinFor, setJoinFor] = useState<BeeView | null>(null);
  const [joinPeer, setJoinPeer] = useState('');
  const [joinFingerprint, setJoinFingerprint] = useState('');
  const [joinForce, setJoinForce] = useState(false);
  const [evictFrom, setEvictFrom] = useState<BeeView | null>(null);
  const [evictNode, setEvictNode] = useState('');

  // There is no endpoint that lists hives, only one that creates them, so the
  // only names this page can know are the ones it watched being created. Every
  // other cluster is identified by the id its hypervisors report. Losing the name on
  // reload is ugly; inventing one would be worse.

  // Bees are polled rather than subscribed: during a rotation the interesting
  // change is a hypervisor going away and coming back, which is minutes apart, and a
  // socket per operator is a lot of machinery for that.
  const { data, isLoading, isError, error } = useQuery({
    ...fleetBeesListOptions(),
    refetchInterval: 5000,
  });

  const hivesQuery = useQuery(fleetHivesListOptions());

  // Evict names a CLUSTER NODE, and the only place a node name exists is the
  // machine record. BeeView carries the address the agent answers on, which
  // corosync has never heard of — `pvecm delnode <address>` is refused by the
  // agent as a non-member, so the name has to be fetched and joined on
  // hypervisor_id rather than guessed from the row in hand.
  const hypervisorsQuery = useQuery(fleetHypervisorsListOptions());

  const refresh = () => queryClient.invalidateQueries({ queryKey: fleetBeesListQueryKey() });

  const createTask = useMutation({
    ...fleetTasksCreateMutation(),
    onSuccess: () => {
      toast.success('Task queued');
      refresh();
    },
    onError: (e) => toast.error((e as { detail?: string })?.detail ?? 'Could not queue that task'),
  });

  // Queueing opens the log tail on the same task. A bee may hold only one
  // outstanding task, so "what did that do" has exactly one answer and there is
  // no reason to make someone go looking for it.
  const queueTask = (bee: BeeView, kind: TaskKind, spec?: unknown) =>
    createTask.mutate(
      { body: { bee_id: bee.id, kind, ...(spec === undefined ? {} : { spec }) } },
      { onSuccess: (res) => setWatching({ taskId: res.id, hostname: beeLabel(bee), kind }) },
    );

  const bees = data?.bees ?? [];
  const hiveless = bees.filter((b) => hiveIdOf(b) === null);

  const byHive = new Map<string, BeeView[]>();
  for (const b of bees) {
    const id = hiveIdOf(b);
    if (id === null) continue;
    byHive.set(id, [...(byHive.get(id) ?? []), b]);
  }
  // Names come from the server. This page used to remember the clusters it had
  // created in a local map and fall back to "Cluster 3f2a91b4" for the rest — which
  // is every cluster made by anyone else, and every cluster at all after a reload.
  const hiveNames = new Map((hivesQuery.data?.hives ?? []).map((h) => [h.id, h.name]));
  const clusterGroups: ClusterGroup[] = [...new Set([...byHive.keys(), ...hiveNames.keys()])]
    .map((id) => ({
      id,
      name: hiveNames.get(id) ?? `Cluster ${id.slice(0, 8)}`,
      bees: byHive.get(id) ?? [],
    }))
    .sort((a, b) => a.name.localeCompare(b.name));

  const nodeNames = new Map(
    (hypervisorsQuery.data?.hypervisors ?? [])
      .filter((h) => h.name)
      .map((h) => [h.id, h.name as string]),
  );

  // Eviction is run BY a node that stays, naming one that goes: a node cannot
  // remove itself from the cluster it is leaving. So the candidates are this
  // hypervisor's cluster-mates, never this hypervisor — and only those whose
  // node name is known, because a candidate with no name is one the cluster has
  // not reported, which the agent would refuse anyway.
  const evictCandidates = (
    evictFrom
      ? (byHive.get(hiveIdOf(evictFrom) ?? '') ?? []).filter((b) => b.id !== evictFrom.id)
      : []
  )
    .map((b) => ({ bee: b, node: b.hypervisor_id ? nodeNames.get(b.hypervisor_id) : undefined }))
    .filter((c): c is { bee: BeeView; node: string } => Boolean(c.node));

  const rowActions = {
    onPing: (b: BeeView) => queueTask(b, 'fleet.host.ping'),
    onUpgrade: (b: BeeView) => setUpgradeFor(b),
    onJoin: (b: BeeView) => setJoinFor(b),
    onEvict: (b: BeeView) => {
      setEvictNode('');
      setEvictFrom(b);
    },
  };

  return (
    <div className="w-full space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-heading text-2xl">Agents</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            An agent runs on each hypervisor; this page lists the ones enrolled, and is where you
            give them tasks.{' '}
            <Link to="/fleet/hypervisors" className="underline underline-offset-2">
              Hypervisors
            </Link>{' '}
            shows the same machines as their cluster reports them — capacity, guests, drain state. A
            hypervisor can be enrolled here before it joins a cluster, and a cluster can report one
            that has no agent on it yet, so the two lists rarely match exactly. Belonging to a
            cluster is separate and optional: a hypervisor joins one, leaves it, and is briefly in
            none, and that in-between is a normal step rather than a fault.
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            render={<Link to="/fleet/hives" />}
            nativeButton={false}
          >
            <IconPlus /> Manage clusters
          </Button>
          <Button size="sm" onClick={() => setEnrolOpen(true)}>
            <IconKey /> Enrol a hypervisor
          </Button>
        </div>
      </div>

      {isLoading ? (
        <Card>
          <CardHeader>
            <CardTitle>Hypervisors</CardTitle>
            <CardDescription>Loading…</CardDescription>
          </CardHeader>
        </Card>
      ) : isError ? (
        // A failure has to say so. An empty page and an unreachable server
        // page look identical otherwise, and one of them means every hypervisor is
        // unmanaged right now.
        <Card>
          <CardHeader>
            <CardTitle>Hypervisors could not be loaded</CardTitle>
            <CardDescription>
              {(error as { detail?: string })?.detail ??
                'The server did not answer. Nothing below reflects the state of your hives.'}
            </CardDescription>
          </CardHeader>
        </Card>
      ) : bees.length === 0 && clusterGroups.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>No hypervisors yet</CardTitle>
            <CardDescription>
              A hypervisor runs the agent, reports in, and takes work such as an upgrade or a
              cluster join.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Enrol a hypervisor to get a token, then run{' '}
              <code className="rounded bg-muted px-1 py-0.5">
                apiary-agent enrol --token &lt;token&gt;
              </code>{' '}
              on the hypervisor. It will appear below in no cluster, which is where every hypervisor
              starts.
            </p>
          </CardContent>
        </Card>
      ) : (
        <>
          {/*
            First on the page, before any cluster. These are the hypervisors a migration
            is in the middle of and the ones a failed join stranded, and a page
            that lists hypervisors only under the cluster they are in cannot show them at
            all. The section stays put when it is empty so an operator learns
            where to look before they need it.
          */}
          <Card className={hiveless.length > 0 ? 'border-amber-500/50' : undefined}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                In no cluster
                {hiveless.length > 0 ? <Badge variant="secondary">{hiveless.length}</Badge> : null}
              </CardTitle>
              <CardDescription>
                {hiveless.length === 0
                  ? 'Every hypervisor is in a cluster. One appears here while it is between hives, and stays here if a join fails.'
                  : `${hiveless.length} hypervisor${hiveless.length === 1 ? '' : 's'} belong${
                      hiveless.length === 1 ? 's' : ''
                    } to no cluster — newly enrolled, mid-migration, or left behind by a join that did not finish.`}
              </CardDescription>
            </CardHeader>
            {hiveless.length > 0 ? (
              <CardContent>
                <BeeTable bees={hiveless} hiveless {...rowActions} busy={createTask.isPending} />
              </CardContent>
            ) : null}
          </Card>

          {clusterGroups.map((f) => (
            <Card key={f.id}>
              <CardHeader>
                <CardTitle>{f.name}</CardTitle>
                <CardDescription>
                  {f.bees.length === 0
                    ? 'No hypervisor has joined this cluster yet.'
                    : `${f.bees.length} hypervisor${f.bees.length === 1 ? '' : 's'} in this cluster.`}
                </CardDescription>
              </CardHeader>
              <CardContent>
                {f.bees.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    Enrol a hypervisor, then use <span className="font-medium">Join a cluster</span>{' '}
                    on it above. Joining runs on the hypervisor itself and needs an existing member
                    of this cluster to join through.
                  </p>
                ) : (
                  <BeeTable bees={f.bees} {...rowActions} busy={createTask.isPending} />
                )}
              </CardContent>
            </Card>
          ))}
        </>
      )}

      {watching ? (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <IconTerminal2 className="size-4" /> {watching.kind} on {watching.hostname}
            </CardTitle>
            <CardDescription>
              Streamed by the agent while it works, so a rotation can be watched without opening a
              shell on the hypervisor.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {/*
              key is load-bearing, not a lint appeasement. TaskLogs accumulates
              `lines` and a `since` cursor in its own state and never resets them
              on a prop change, so without a key React reuses the instance when
              the watched task changes and the new task's pane opens showing the
              PREVIOUS task's output — with a cursor already advanced past the
              new task's early lines, which are then never fetched. During a
              rotation that is one hypervisor's output attributed to another.
            */}
            <TaskLogs
              key={watching.taskId}
              taskId={watching.taskId}
              onClose={() => setWatching(null)}
            />
          </CardContent>
        </Card>
      ) : null}

      {/*
        The whole draft resets on close, matching the join and evict dialogs.
        Clearing only upgradeFor carried allow_guest_downtime into the NEXT
        hypervisor's dialog pre-ticked, so an operator who consented to downtime on one
        hypervisor silently consented on the following one without seeing the
        box change.
      */}
      <Dialog
        open={upgradeFor !== null}
        onOpenChange={(o) => {
          if (!o) {
            setUpgradeFor(null);
            setAllowDowntime(false);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Upgrade {upgradeFor ? beeLabel(upgradeFor) : ''}</DialogTitle>
            <DialogDescription>
              Runs apt dist-upgrade and reboots if the distribution asks for one. The agent declares
              the expected downtime first, so the reboot is not recorded as an outage — and it
              refuses outright if guests are still running here or if losing this node would cost
              the cluster quorum.
            </DialogDescription>
          </DialogHeader>
          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={allowDowntime}
              onChange={(e) => setAllowDowntime(e.target.checked)}
            />
            <span>
              Allow guest downtime
              <span className="block text-xs text-muted-foreground">
                Without this the upgrade refuses while any guest is running on the hypervisor. Tick
                it only if those guests are expected to go down with it.
              </span>
            </span>
          </label>
          <DialogFooter>
            <Button
              disabled={createTask.isPending}
              onClick={() => {
                if (!upgradeFor) return;
                queueTask(upgradeFor, 'fleet.host.dist-upgrade', {
                  allow_guest_downtime: allowDowntime,
                  expected_down_seconds: 900,
                });
                setUpgradeFor(null);
                setAllowDowntime(false);
              }}
            >
              {createTask.isPending ? 'Queueing…' : 'Upgrade'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={joinFor !== null}
        onOpenChange={(o) => {
          if (!o) {
            setJoinFor(null);
            setJoinPeer('');
            setJoinFingerprint('');
            setJoinForce(false);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Join a cluster with {joinFor ? beeLabel(joinFor) : ''}</DialogTitle>
            <DialogDescription>
              The join runs on this hypervisor, against an existing member of the cluster it is
              joining. Joining discards this hypervisor's local guest configuration, so the agent
              refuses while any guest is registered here. The cluster password is never sent from
              here — the agent reads it from the hypervisor.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="join-peer">Peer</Label>
            <Input
              id="join-peer"
              value={joinPeer}
              onChange={(e) => setJoinPeer(e.target.value)}
              placeholder="hv1.dc1.example.net"
            />
            <p className="text-xs text-muted-foreground">
              A hypervisor that is already in the target cluster.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="join-fingerprint">Peer certificate fingerprint</Label>
            <Input
              id="join-fingerprint"
              value={joinFingerprint}
              onChange={(e) => setJoinFingerprint(e.target.value)}
              placeholder="AB:CD:EF:…"
            />
            <p className="text-xs text-muted-foreground">
              Without it the join cannot tell the intended cluster from anything else answering on
              that address.
            </p>
          </div>
          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={joinForce}
              onChange={(e) => setJoinForce(e.target.checked)}
            />
            <span>
              Force
              <span className="block text-xs text-muted-foreground">
                Joins even though this hypervisor still carries cluster configuration of its own.
                Only for a hypervisor that was removed from a cluster and not reinstalled.
              </span>
            </span>
          </label>
          <DialogFooter>
            <Button
              disabled={!joinPeer || !joinFingerprint || createTask.isPending}
              onClick={() => {
                if (!joinFor) return;
                queueTask(joinFor, 'fleet.datacenter.join', {
                  peer: joinPeer,
                  fingerprint: joinFingerprint,
                  force: joinForce,
                });
                setJoinFor(null);
                setJoinPeer('');
                setJoinFingerprint('');
                setJoinForce(false);
              }}
            >
              {createTask.isPending ? 'Queueing…' : 'Join'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={evictFrom !== null}
        onOpenChange={(o) => {
          if (!o) {
            setEvictFrom(null);
            setEvictNode('');
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              Remove {evictNode || 'a node'} from {evictFrom ? beeLabel(evictFrom) : ''}&apos;s
              cluster
            </DialogTitle>
            <DialogDescription>
              The task runs on {evictFrom ? beeLabel(evictFrom) : 'it'}, which stays: a node cannot
              remove itself from the cluster it is leaving. The removed hypervisor ends up in no
              cluster and must not be powered back on with its existing configuration — it would try
              to rejoin a cluster that no longer expects it. Reassign its guests first; the agent
              refuses while any are still registered to it, because removal orphans them with no
              supported way back.
            </DialogDescription>
          </DialogHeader>
          {evictCandidates.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No other node of this cluster can be named. Either{' '}
              {evictFrom ? beeLabel(evictFrom) : 'it'} is the only hypervisor in it and cannot
              remove itself, or its cluster-mates have not been reported by a capacity sync yet —
              eviction needs the node name the cluster knows them by.
            </p>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="evict-node">Node to remove</Label>
              <Select
                value={evictNode}
                onValueChange={(v) => {
                  if (typeof v === 'string') setEvictNode(v);
                }}
              >
                <SelectTrigger id="evict-node" className="w-full">
                  <SelectValue>{(v: string | null) => v || 'Select a hypervisor'}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {evictCandidates.map((c) => (
                    <SelectItem key={c.bee.id} value={c.node}>
                      {c.node}
                      {beeLabel(c.bee) === c.node ? '' : ` (${beeLabel(c.bee)})`}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}
          <DialogFooter>
            <Button
              variant="destructive"
              disabled={!evictNode || createTask.isPending}
              onClick={() => {
                if (!evictFrom || !evictNode) return;
                queueTask(evictFrom, 'fleet.datacenter.evict', { node: evictNode });
                setEvictFrom(null);
                setEvictNode('');
              }}
            >
              {createTask.isPending ? 'Queueing…' : `Remove ${evictNode}`}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <EnrolmentDialog open={enrolOpen} onOpenChange={setEnrolOpen} />
    </div>
  );
};
