import { useState } from 'react';

import {
  IconLogout,
  IconPlayerPause,
  IconPlayerPlay,
  IconPlus,
  IconRefresh,
  IconTrash,
} from '@tabler/icons-react';
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { FleetlessHostView, HypervisorView } from '@/sdk';
import {
  fleetHivesListOptions,
  fleetHivesSyncMutation,
  fleetHostsArriveMutation,
  fleetHostsFleetlessOptions,
  fleetHostsFleetlessQueryKey,
  fleetHypervisorsCordonMutation,
  fleetHypervisorsDepartMutation,
  fleetHypervisorsListOptions,
  fleetHypervisorsListQueryKey,
  fleetHypervisorsReprovisionMutation,
} from '@/sdk/@tanstack/react-query.gen';

import { StatusBadge, relative } from './bee-table';
import { DrainBadge, Hint, LoadError, RefusalList, capacityLabel } from './drain';

/**
 * Pulls the message out of a 409 refusal.
 *
 * The server answers a gated operation with 409 and one detail per reason code.
 * Surfacing only the top-level message would drop the part that says what to do,
 * which is the whole reason the gate returns every reason rather than the first.
 */
function refusalMessage(error: unknown, fallback: string): string {
  const e = error as { errors?: { message?: string }[]; detail?: string } | undefined;
  const details = e?.errors?.map((d) => d.message).filter(Boolean) as string[] | undefined;
  if (details?.length) return details.join(' — ');
  return e?.detail ?? fallback;
}

/**
 * The onboarding state of one machine, which is a DIFFERENT axis from whether
 * its agent is reachable and from whether it is cordoned.
 *
 * Rendered only when there is something to say. `provisioned` is the ordinary
 * resting state of every working machine, and a badge on every row saying so
 * would bury the two rows that actually need attention.
 *
 * The attempt count appears on a failure because it is what distinguishes "this
 * failed once and will be retried" from "this has hit the cap and no further
 * tick will touch it" — and the second needs an operator, not patience.
 */
function ProvisioningBadge({ hv }: { hv: HypervisorView }) {
  switch (hv.provisioning_state) {
    case 'pending':
      return (
        <Badge className="bg-sky-500/15 text-sky-600 dark:text-sky-400">queued to onboard</Badge>
      );
    case 'provisioning':
      return <Badge className="bg-sky-500/15 text-sky-600 dark:text-sky-400">onboarding…</Badge>;
    case 'failed':
      return (
        <span className="block space-y-1">
          <Badge variant="destructive">
            onboarding failed
            {hv.provisioning_attempt ? ` (attempt ${hv.provisioning_attempt})` : ''}
          </Badge>
          {hv.provisioning_error ? (
            <span
              className="block max-w-[18rem] truncate text-xs text-muted-foreground"
              title={hv.provisioning_error}
            >
              {hv.provisioning_error}
            </span>
          ) : null}
        </span>
      );
    default:
      return null;
  }
}

export const Hypervisors = () => {
  const queryClient = useQueryClient();
  const [arriveFor, setArriveFor] = useState<FleetlessHostView | null>(null);
  const [arriveHive, setArriveHive] = useState('');
  const [arriveLinks, setArriveLinks] = useState<
    { number: number; address: string; priority: string }[]
  >([{ number: 0, address: '', priority: '' }]);

  const hypervisors = useQuery(fleetHypervisorsListOptions());
  const fleetless = useQuery(fleetHostsFleetlessOptions());
  const hives = useQuery(fleetHivesListOptions());

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: fleetHypervisorsListQueryKey() });
    void queryClient.invalidateQueries({ queryKey: fleetHostsFleetlessQueryKey() });
  };

  const cordon = useMutation({
    ...fleetHypervisorsCordonMutation(),
    onSuccess: (_data, vars) => {
      const schedulable = (vars as { body?: { schedulable?: boolean } })?.body?.schedulable;
      toast.success(schedulable ? 'Node uncordoned' : 'Node cordoned');
      invalidate();
    },
    onError: (e) => toast.error(refusalMessage(e, 'Could not change scheduling')),
  });

  const reprovision = useMutation({
    ...fleetHypervisorsReprovisionMutation(),
    onSuccess: () => {
      toast.success('Queued for onboarding — a worker claims it on the next tick');
      invalidate();
    },
    // The endpoint refuses with reason codes rather than a bare conflict: no
    // address to connect to, already queued, or a worker inside it right now.
    // Those are the three things an operator can act on, so they are shown.
    onError: (e) => toast.error(refusalMessage(e, 'This machine cannot be reprovisioned yet')),
  });

  const depart = useMutation({
    ...fleetHypervisorsDepartMutation(),
    onSuccess: () => {
      toast.success('Departure queued — the hypervisor will be left standalone and fleetless');
      invalidate();
    },
    // A refused departure is not a failure of the system; it is the gate saying
    // the hypervisor is not ready. The reasons are what the operator needs.
    onError: (e) => toast.error(refusalMessage(e, 'This hypervisor cannot leave its cluster yet')),
  });

  const sync = useMutation({
    ...fleetHivesSyncMutation(),
    onSuccess: () => toast.success('Capacity sync queued'),
    onError: (e) => toast.error(refusalMessage(e, 'Could not queue a sync')),
  });

  const arrive = useMutation({
    ...fleetHostsArriveMutation(),
    onSuccess: () => {
      toast.success('Arrival queued');
      setArriveFor(null);
      setArriveHive('');
      setArriveLinks([{ number: 0, address: '', priority: '' }]);
      invalidate();
    },
    onError: (e) => toast.error(refusalMessage(e, 'This hypervisor cannot join that cluster yet')),
  });

  const nodes = hypervisors.data?.hypervisors ?? [];
  const strays = fleetless.data?.hosts ?? [];
  const hiveList = hives.data?.hives ?? [];

  // /fleet/hypervisors returns every MACHINE, clustered or not, since the machine
  // record outlived cluster membership. Unclustered ones are not dropped from the
  // page — they are the "in no cluster" card below, fed by /fleet/fleetless — but
  // they must not be grouped, or an undefined hive_id becomes a cluster card with
  // no cluster behind it.
  const byHive = new Map<string, HypervisorView[]>();
  for (const n of nodes) {
    if (!n.hive_id) continue;
    const list = byHive.get(n.hive_id) ?? [];
    list.push(n);
    byHive.set(n.hive_id, list);
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Hypervisors</h1>
        <p className="text-sm text-muted-foreground">
          Hypervisors as their cluster reports them. Cordon one to stop new guests landing on it,
          move what is already there, then let it leave its cluster.{' '}
          <Link to="/fleet" className="underline underline-offset-2">
            Agents
          </Link>{' '}
          is the same machines seen through the agent installed on each — that view can show a
          hypervisor before it joins a cluster, and this one can show a hypervisor with no agent on
          it yet.
        </p>
      </div>

      {hypervisors.isError ? (
        <LoadError what="Hypervisors" error={hypervisors.error} />
      ) : hypervisors.isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : byHive.size === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">No hypervisors in a cluster</CardTitle>
            <CardDescription>
              Nodes appear here once a capacity sync has read them from a cluster's API. A cluster
              with no API URL or credential is never synced.
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        [...byHive.entries()].map(([hiveId, hiveNodes]) => {
          const cluster = hiveList.find((h) => h.id === hiveId);
          const min = cluster?.effective_minimum_nodes ?? 3;
          return (
            <Card key={hiveId}>
              <CardHeader className="flex-row items-start justify-between gap-4 space-y-0">
                <div>
                  <CardTitle className="text-base">
                    {cluster?.name ?? hiveNodes[0].hive_name}
                  </CardTitle>
                  <CardDescription>
                    {hiveNodes.length} node{hiveNodes.length === 1 ? '' : 's'} · departures need at
                    least {min}
                  </CardDescription>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={sync.isPending || !cluster?.configured}
                  title={
                    cluster?.configured
                      ? 'Refresh capacity observations for this cluster now'
                      : 'This cluster has no API URL or credential, so it cannot be synced'
                  }
                  onClick={() => sync.mutate({ path: { id: hiveId } })}
                >
                  <IconRefresh className="size-4" />
                  Sync now
                </Button>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Node</TableHead>
                      <TableHead>Scheduling</TableHead>
                      <TableHead>Drain</TableHead>
                      <TableHead>Guests</TableHead>
                      <TableHead>Allocatable</TableHead>
                      <TableHead>Observed</TableHead>
                      <TableHead>Agent</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {hiveNodes.map((n) => (
                      <TableRow key={n.id}>
                        <TableCell className="font-medium">
                          {n.name}
                          {n.address ? (
                            <span className="block text-xs text-muted-foreground">{n.address}</span>
                          ) : null}
                        </TableCell>
                        <TableCell>
                          {n.schedulable ? (
                            <Badge className="bg-emerald-500/15 text-emerald-500">
                              schedulable
                            </Badge>
                          ) : (
                            <Badge className="bg-amber-500/15 text-amber-500">cordoned</Badge>
                          )}
                        </TableCell>
                        <TableCell>
                          <DrainBadge state={n.drain_state} />
                        </TableCell>
                        <TableCell>
                          <span title="Guests the provider reports, stopped ones included">
                            {n.observed_guests}
                          </span>
                          {n.held_placements > 0 ? (
                            <span
                              className="block text-xs text-muted-foreground"
                              title="Placements this system still holds capacity for"
                            >
                              {n.held_placements} placed
                            </span>
                          ) : null}
                        </TableCell>
                        <TableCell className="text-xs">{capacityLabel(n.allocatable)}</TableCell>
                        <TableCell className="text-xs">
                          {capacityLabel(n.observed)}
                          <span
                            className={
                              n.observation_stale
                                ? 'block text-amber-500'
                                : 'block text-muted-foreground'
                            }
                          >
                            synced {relative(n.last_synced_at)}
                          </span>
                        </TableCell>
                        <TableCell className="space-y-1">
                          {n.bee_status ? (
                            <StatusBadge status={n.bee_status} />
                          ) : (
                            <span className="block text-xs text-muted-foreground">
                              not enrolled
                            </span>
                          )}
                          <ProvisioningBadge hv={n} />
                        </TableCell>
                        <TableCell className="space-y-2 text-right">
                          <div className="flex justify-end gap-2">
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={cordon.isPending}
                              onClick={() =>
                                cordon.mutate({
                                  path: { id: n.id },
                                  body: { schedulable: !n.schedulable },
                                })
                              }
                            >
                              {n.schedulable ? (
                                <>
                                  <IconPlayerPause className="size-4" />
                                  Cordon
                                </>
                              ) : (
                                <>
                                  <IconPlayerPlay className="size-4" />
                                  Uncordon
                                </>
                              )}
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={reprovision.isPending || n.address === ''}
                              onClick={() => reprovision.mutate({ path: { id: n.id } })}
                              title={
                                n.address === ''
                                  ? 'No address recorded, so there is nothing to connect to'
                                  : 'Install or reinstall the agent over SSH'
                              }
                            >
                              <IconRefresh className="size-4" />
                              {n.provisioning_state === 'provisioned' ? 'Reinstall' : 'Onboard'}
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              /* Disabled from the SERVER's refusal list, never
                                 from a rule re-derived here — a second copy of
                                 the gate is how a button ends up enabled for
                                 something the server will refuse. */
                              disabled={depart.isPending || (n.departure_refusals?.length ?? 0) > 0}
                              onClick={() => depart.mutate({ path: { id: n.id } })}
                            >
                              <IconLogout className="size-4" />
                              Depart
                            </Button>
                          </div>
                          {n.last_operation_error ? (
                            <div className="text-left">
                              <p className="text-xs font-medium text-destructive">
                                Last {n.last_operation || 'operation'} failed
                                {n.last_operation_at ? ` ${relative(n.last_operation_at)}` : ''}
                              </p>
                              <p
                                className="max-w-md text-xs break-words text-muted-foreground"
                                title={n.last_operation_error}
                              >
                                {n.last_operation_error}
                              </p>
                            </div>
                          ) : null}
                          {(n.departure_refusals?.length ?? 0) > 0 ? (
                            <div className="text-left">
                              <RefusalList
                                refusals={n.departure_refusals ?? []}
                                onSync={
                                  cluster?.configured
                                    ? () => sync.mutate({ path: { id: hiveId } })
                                    : undefined
                                }
                              />
                            </div>
                          ) : null}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          );
        })
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Hypervisors in no cluster</CardTitle>
          <CardDescription>
            A migration is a departure and then an arrival. Hypervisors sit here in between — and a
            join that failed leaves one here rather than half-migrated.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {strays.length === 0 ? (
            <Hint>Nothing is between clusters right now.</Hint>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Hypervisor</TableHead>
                  <TableHead>Agent</TableHead>
                  <TableHead>Last seen</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {strays.map((h) => (
                  <TableRow key={h.hypervisor_id}>
                    <TableCell className="font-medium">{h.address}</TableCell>
                    <TableCell>
                      {/* Absent status is not offline: it means no agent is enrolled on
                          this machine at all. Rendering the offline badge would report
                          an outage on a machine nobody has installed anything on. */}
                      {h.status ? (
                        <StatusBadge status={h.status} />
                      ) : (
                        <Badge variant="outline">no agent</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {relative(h.last_seen_at)}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={hiveList.length === 0}
                        onClick={() => {
                          setArriveFor(h);
                          setArriveHive('');
                          setArriveLinks([{ number: 0, address: '', priority: '' }]);
                        }}
                      >
                        Join a cluster
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={arriveFor !== null} onOpenChange={(open) => !open && setArriveFor(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Join a cluster</DialogTitle>
            <DialogDescription>
              {arriveFor?.address} will join the cluster and become one of its nodes. Its own
              cluster state is replaced by the destination's.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="target-cluster">Destination cluster</Label>
            <Select value={arriveHive} onValueChange={(v) => setArriveHive(v ?? '')}>
              <SelectTrigger id="target-cluster">
                <SelectValue placeholder="Choose a cluster" />
              </SelectTrigger>
              <SelectContent>
                {hiveList.map((h) => (
                  <SelectItem key={h.id} value={h.id}>
                    {h.name} ({h.node_count} node{h.node_count === 1 ? '' : 's'})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Hint>
              The joining hypervisor must carry no guests of its own — a cluster join refuses a node
              with any, stopped ones included.
            </Hint>
          </div>
          <div className="space-y-2">
            <Label>Corosync links</Label>
            {arriveLinks.map((link, i) => (
              <div key={link.number} className="flex items-center gap-2">
                <span className="w-14 shrink-0 text-xs text-muted-foreground">
                  link{link.number}
                </span>
                <Input
                  value={link.address}
                  placeholder="10.1.0.9"
                  onChange={(e) =>
                    setArriveLinks((ls) =>
                      ls.map((l, j) => (j === i ? { ...l, address: e.target.value } : l)),
                    )
                  }
                />
                <Input
                  className="w-24"
                  type="number"
                  min={0}
                  value={link.priority}
                  placeholder="prio"
                  title="knet link preference; higher wins. Leave blank for corosync's default, which is not the same as 0."
                  onChange={(e) =>
                    setArriveLinks((ls) =>
                      ls.map((l, j) => (j === i ? { ...l, priority: e.target.value } : l)),
                    )
                  }
                />
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={arriveLinks.length === 1}
                  onClick={() => setArriveLinks((ls) => ls.filter((_, j) => j !== i))}
                  aria-label={`Remove link${link.number}`}
                >
                  <IconTrash className="size-4" />
                </Button>
              </div>
            ))}
            {arriveLinks.length < 8 ? (
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  setArriveLinks((ls) => {
                    /* Next FREE ring number, not ls.length. Removing link1 from
                       [0,1,2] would otherwise offer 2, which already exists —
                       and two values for one ring is not a merge: pvecm takes
                       one --linkN and the second silently replaces the first. */
                    const used = new Set(ls.map((l) => l.number));
                    let next = 0;
                    while (used.has(next) && next < 8) next += 1;
                    return [...ls, { number: next, address: '', priority: '' }];
                  })
                }
              >
                <IconPlus className="size-4" />
                Add a link
              </Button>
            ) : null}
            <Hint>
              This hypervisor's addresses on the cluster's corosync rings. Not assumed to be the
              address the control plane manages the hypervisor over — those are routinely different
              networks, and putting cluster traffic on the wrong one works until that network is
              busy. A second ring is what stops the cluster network being a single point of failure
              for quorum. Priority is optional: higher wins, and leaving it blank keeps corosync's
              default rather than setting zero.
            </Hint>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setArriveFor(null)}>
              Cancel
            </Button>
            <Button
              disabled={
                !arriveHive || arriveLinks.every((l) => !l.address.trim()) || arrive.isPending
              }
              onClick={() =>
                arriveFor &&
                arrive.mutate({
                  path: { hypervisor_id: arriveFor.hypervisor_id },
                  body: {
                    target_hive_id: arriveHive,
                    // Blank rows are dropped rather than sent: an empty address
                    // is a row the operator added and did not fill in, and the
                    // server refuses the whole set for one of them.
                    links: arriveLinks
                      .filter((l) => l.address.trim() !== '')
                      .map((l) => ({
                        number: l.number,
                        address: l.address.trim(),
                        // Omitted when blank, never sent as 0. Absent leaves
                        // corosync's own default; an explicit 0 would silently
                        // deprioritise a link the operator said nothing about.
                        ...(l.priority.trim() === ''
                          ? {}
                          : { priority: Number(l.priority.trim()) }),
                      })),
                  },
                })
              }
            >
              {arrive.isPending ? 'Queueing…' : 'Queue arrival'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};
