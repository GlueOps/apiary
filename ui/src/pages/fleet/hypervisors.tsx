import { useEffect, useRef, useState } from 'react';

import {
  IconLogout,
  IconPlayerPause,
  IconPlayerPlay,
  IconPlus,
  IconRefresh,
  IconTrash,
} from '@tabler/icons-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useSearchParams } from 'react-router-dom';
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
  fleetHostsBootstrapMutation,
  fleetHostsFleetlessOptions,
  fleetHostsFleetlessQueryKey,
  fleetHypervisorsCordonMutation,
  fleetHypervisorsDepartMutation,
  fleetHypervisorsListOptions,
  fleetHypervisorsListQueryKey,
  fleetHypervisorsReprovisionMutation,
} from '@/sdk/@tanstack/react-query.gen';

import { StatusBadge, relative } from './bee-table';
import { DrainBadge, Hint, LoadError, RefusalBadges, RemedyCard, capacityLabel } from './drain';

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
  // Departure asks a question before it runs, and starts with no answer chosen.
  // The two safe answers cost different things — rotating disrupts every API
  // client in the cluster, accepting leaves a departed host able to mint auth
  // tickets for it — so preselecting either would be this page deciding which
  // the operator can afford.
  const [departFor, setDepartFor] = useState<{ id: string; address?: string } | null>(null);
  // Typed to the API's own enum rather than string, so a third option cannot be
  // added here without the generated SDK agreeing it exists.
  const [departDisposition, setDepartDisposition] = useState<'' | 'rotate' | 'accept-exposure'>('');

  const [arriveFor, setArriveFor] = useState<FleetlessHostView | null>(null);
  // Arriving used to be offered here AND on the agents page, as two different
  // things wearing one name: this queues the modelled arrival — target cluster,
  // corosync rings, the refusal checks, and the membership recorded when it
  // lands — while that one queued a raw agent task with a hand-typed peer and
  // never told APIary which cluster the machine had joined. The agents page now
  // links here instead, and ?arrive=<id> is how it hands the machine over.
  const [params, setParams] = useSearchParams();
  const arriveParam = params.get('arrive');
  const handedOver = useRef<string | null>(null);
  const [arriveHive, setArriveHive] = useState('');
  const [arriveLinks, setArriveLinks] = useState<
    { number: number; address: string; priority: string }[]
  >([{ number: 0, address: '', priority: '' }]);

  // Depart, join and onboard are QUEUED, not done. Clicking one and seeing the
  // page unchanged is correct and useless: the work is happening in a worker,
  // and until this refetches there is nothing on screen to say so.
  //
  // Polled at a rate that depends on whether anything is actually in flight,
  // rather than a flat interval. A fleet page left open on a wall display would
  // otherwise re-run several joins every few seconds forever to learn nothing;
  // and a flat SLOW interval would leave an operator staring at a stale row for
  // half a minute after clicking Depart, which is the problem this exists to fix.
  //
  // react-query already pauses polling while the tab is hidden, so a backgrounded
  // page costs nothing.
  const [pollFast, setPollFast] = useState(false);
  const actedAt = useRef(0);
  const refetchInterval = pollFast ? 2000 : 15000;

  const hypervisors = useQuery({ ...fleetHypervisorsListOptions(), refetchInterval });
  const fleetless = useQuery({ ...fleetHostsFleetlessOptions(), refetchInterval });
  const hives = useQuery({ ...fleetHivesListOptions(), refetchInterval });

  const invalidate = () => {
    // Poll quickly from THIS moment, not from when the work becomes visible.
    //
    // A queued departure sets nothing on the machine until a worker picks it up,
    // so the data-derived signal below cannot see it yet — and that gap is
    // exactly the one an operator is staring at, having just clicked the button.
    actedAt.current = Date.now();
    setPollFast(true);
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

  // The other half of the same operator intent: put this machine in that
  // cluster. Which of the two runs is decided by the fleet, not by the operator,
  // because there is no choice to make — a cluster with no nodes has to be
  // created before anything can join it, and one with nodes cannot be created
  // again. Offering both as buttons would be asking someone to answer a question
  // the node count already answers, and getting it wrong costs a round trip and
  // a refusal that reads like a mistake they made.
  const bootstrap = useMutation({
    ...fleetHostsBootstrapMutation(),
    onSuccess: () => {
      toast.success('Cluster creation queued');
      setArriveFor(null);
      setArriveHive('');
      setArriveLinks([{ number: 0, address: '', priority: '' }]);
      invalidate();
    },
    onError: (e) => toast.error(refusalMessage(e, 'This hypervisor cannot create that cluster')),
  });

  const nodes = hypervisors.data?.hypervisors ?? [];

  // Whether the fleet is mid-operation, derived from the data rather than from
  // what this tab happens to have clicked.
  //
  // That distinction matters: a departure started in another browser, or by a
  // colleague, or by an API client, moves this page just as much as one started
  // here — and a flag set by our own mutation would leave those invisible until
  // somebody reloaded.
  //
  // Three signals, because the three operations report themselves differently.
  // Onboarding has a state column. Departures and arrivals have none, so they
  // are recognised by a recent last_operation_at — the window is generous
  // because a job can sit queued before it runs. And a machine that is neither
  // provisioned nor failed is somewhere in between.
  const busy = nodes.some((n) => {
    if (n.provisioning_state === 'pending' || n.provisioning_state === 'provisioning') return true;
    if (!n.last_operation_at) return false;
    return Date.now() - new Date(n.last_operation_at).getTime() < 3 * 60 * 1000;
  });

  useEffect(() => {
    if (busy) {
      setPollFast(true);
      return;
    }
    // Nothing in flight. Keep polling quickly for a grace period after the last
    // click, because a queued job is invisible in the data until it starts —
    // dropping to the slow interval the instant a mutation returns would put the
    // operator back to waiting fifteen seconds to see their own action.
    const remaining = 60_000 - (Date.now() - actedAt.current);
    if (remaining <= 0) {
      setPollFast(false);
      return;
    }
    const timer = setTimeout(() => setPollFast(false), remaining);
    return () => clearTimeout(timer);
  }, [busy]);

  // Every refusal on the page, for the remedy card at the bottom.
  //
  // Collected here rather than inside each row because the fix for a reason is
  // the same wherever it appears: rendering it per host repeats the same
  // paragraph down the page and makes every row wide enough to scroll sideways.
  const allRefusals = nodes.flatMap((n) => n.departure_refusals ?? []);

  // Refusals by machine id, so the fleetless table can show the same badges.
  //
  // That table is fed by a different endpoint whose view carries no refusals,
  // and without this a host listed there was counted in the card ("not a member
  // · 1 host") with nothing on the page saying WHICH host — the exact gap the
  // badges exist to close. The two lists describe the same machines, so the id
  // is enough and no second copy of the gate has to travel.
  const refusalsByMachine = new Map(nodes.map((n) => [n.id, n.departure_refusals ?? []]));

  // A cluster to offer the sync against, for the one refusal an operator
  // resolves by touching no host at all. The first configured one: a stale
  // observation is almost always fleet-wide, and offering a picker for it would
  // be more ceremony than the action deserves.
  const syncableHiveId = (hives.data?.hives ?? []).find((h) => h.configured)?.id;
  const strays = fleetless.data?.hosts ?? [];
  const hiveList = hives.data?.hives ?? [];

  // An empty cluster has to be CREATED on its first node; a cluster with nodes
  // is JOINED. Read off the chosen cluster rather than asked, because the node
  // count already decides it and there is no second answer available.
  const chosenHive = hiveList.find((h) => h.id === arriveHive);
  const creatingTheCluster = chosenHive !== undefined && chosenHive.node_count === 0;
  const submitting = arrive.isPending || bootstrap.isPending;

  // Open the arrival for a machine handed over from the agents page.
  //
  // Waits for the fleetless list, because the dialog is about a row and the row
  // arrives with the query. The ref makes it fire once: without it, closing the
  // dialog while the parameter is still in the URL reopens it, and the operator
  // cannot get out. Clearing the parameter is what makes a reload of this URL
  // land on the plain page rather than a dialog they already dismissed.
  useEffect(() => {
    if (!arriveParam || handedOver.current === arriveParam) return;
    if (fleetless.isLoading) return;

    handedOver.current = arriveParam;
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.delete('arrive');
        return next;
      },
      { replace: true },
    );

    const host = strays.find((h) => h.hypervisor_id === arriveParam);
    if (!host) {
      // Reached here from a row that is no longer fleetless — it joined
      // something in between, or was removed. Saying so beats a dialog that
      // never opens and no explanation for why.
      toast.error('That hypervisor is no longer waiting to join a cluster.');
      return;
    }
    setArriveFor(host);
    setArriveHive('');
    setArriveLinks([{ number: 0, address: '', priority: '' }]);
  }, [arriveParam, fleetless.isLoading, strays, setParams]);

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
                          {/* Which rules are blocking THIS host. The card at the
                              bottom explains each one and how to fix it; naming
                              them here is what stops that card having to list
                              hostnames in a table cell. */}
                          <RefusalBadges refusals={n.departure_refusals ?? []} />
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
                              onClick={() => {
                                setDepartDisposition('');
                                setDepartFor({ id: n.id, address: n.address ?? undefined });
                              }}
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
                          {/* The reasons live in one card at the bottom of the
                              page, not here. Repeated per host they put the same
                              two lines beside every node that shares a problem
                              and make every row scroll sideways. The disabled
                              Depart button is what marks this host as blocked;
                              the card says what is blocking it. */}
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
                    <TableCell className="font-medium">
                      {h.address}
                      <RefusalBadges refusals={refusalsByMachine.get(h.hypervisor_id) ?? []} />
                    </TableCell>
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
                        title={
                          hiveList.length === 0
                            ? 'Create a cluster on the Clusters page first — naming it is all that takes, and this is where its first node comes from.'
                            : undefined
                        }
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

      <Dialog open={departFor !== null} onOpenChange={(open) => !open && setDepartFor(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Leave the cluster</DialogTitle>
            <DialogDescription>
              {departFor?.address} will be removed from its cluster and left standalone. Its copy of
              the cluster's secrets is destroyed as it goes — but it has already read them.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="credential-disposition">Cluster secrets</Label>
            <Select
              value={departDisposition}
              onValueChange={(v) =>
                setDepartDisposition((v ?? '') as '' | 'rotate' | 'accept-exposure')
              }
            >
              <SelectTrigger id="credential-disposition">
                <SelectValue placeholder="Choose what happens to them" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="rotate">Rotate them — this host is not trusted</SelectItem>
                <SelectItem value="accept-exposure">
                  Accept the exposure — it is trusted and being redeployed
                </SelectItem>
              </SelectContent>
            </Select>
            <Hint>
              Every node shares one set of credentials, so this host holds the cluster's API tokens,
              its ticket-signing key and its root CA. Revoking its SSH access does not revoke those:
              with the ticket-signing key it can still authenticate as any user in the cluster it
              left. Rotate if that matters.
            </Hint>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDepartFor(null)}>
              Cancel
            </Button>
            <Button
              disabled={!departDisposition || depart.isPending}
              onClick={() => {
                if (!departFor || departDisposition === '') return;
                depart.mutate({
                  path: { id: departFor.id },
                  body: { credential_disposition: departDisposition },
                });
                setDepartFor(null);
              }}
            >
              Depart
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={arriveFor !== null} onOpenChange={(open) => !open && setArriveFor(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{creatingTheCluster ? 'Create a cluster' : 'Join a cluster'}</DialogTitle>
            <DialogDescription>
              {creatingTheCluster
                ? `${arriveFor?.address} will become the first node of this cluster, which is created on it. Nothing on the host is replaced, and guests already running here are left alone.`
                : `${arriveFor?.address} will join the cluster and become one of its nodes. Its own cluster state is replaced by the destination's.`}
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
            {creatingTheCluster ? (
              <Hint>
                This cluster has no nodes yet, so it is created here rather than joined. Not the
                same operation: `pvecm create` replaces nothing, so unlike a join it is safe on a
                host already running guests.
              </Hint>
            ) : (
              <Hint>
                The joining hypervisor must carry no guests of its own — a cluster join refuses a
                node with any, stopped ones included.
              </Hint>
            )}
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
              disabled={!arriveHive || arriveLinks.every((l) => !l.address.trim()) || submitting}
              onClick={() => {
                if (!arriveFor) return;
                const body = {
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
                      ...(l.priority.trim() === '' ? {} : { priority: Number(l.priority.trim()) }),
                    })),
                };
                const path = { hypervisor_id: arriveFor.hypervisor_id };
                if (creatingTheCluster) {
                  bootstrap.mutate({ path, body });
                } else {
                  arrive.mutate({ path, body });
                }
              }}
            >
              {submitting
                ? 'Queueing…'
                : creatingTheCluster
                  ? 'Create the cluster'
                  : 'Queue arrival'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <RemedyCard
        refusals={allRefusals}
        onSync={syncableHiveId ? () => sync.mutate({ path: { id: syncableHiveId } }) : undefined}
      />
    </div>
  );
};
