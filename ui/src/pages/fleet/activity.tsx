import { useState } from 'react';

import { IconRefresh } from '@tabler/icons-react';
import { useQuery } from '@tanstack/react-query';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
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
import type { FleetActivityView } from '@/sdk';
import { fleetActivityListOptions } from '@/sdk/@tanstack/react-query.gen';

import { LoadError } from './drain';

const ANY = 'any';

/**
 * The kinds an operator filters by, in the order they matter when something has
 * gone wrong: the credential work first, because a disarm that did not finish
 * leaves a host authenticating against a cluster it has left.
 */
const KINDS: { value: string; label: string }[] = [
  { value: ANY, label: 'Any task' },
  { value: 'fleet.datacenter.disarm', label: 'Disarm' },
  { value: 'fleet.host.agent-upgrade', label: 'Agent upgrade' },
  { value: 'fleet.host.dist-upgrade', label: 'System upgrade' },
];

const STATUSES: { value: string; label: string }[] = [
  { value: ANY, label: 'Any outcome' },
  { value: 'failed', label: 'Failed' },
  { value: 'succeeded', label: 'Succeeded' },
  { value: 'running', label: 'Running' },
  { value: 'pending', label: 'Pending' },
  { value: 'cancelled', label: 'Cancelled' },
];

/**
 * The task as an operator reads it.
 *
 * Falls back to the raw kind rather than to "Other". A task this page has not
 * been taught about is still a real thing an agent did, and filing it under a
 * placeholder is how a trail quietly stops covering what it claims to.
 */
function kindLabel(row: FleetActivityView): string {
  return KINDS.find((k) => k.value === row.kind)?.label ?? row.operation ?? row.kind;
}

/**
 * The machine, named the way somebody who has to go and look at it would.
 *
 * The address leads, because that is what an operator types into a terminal. A
 * task outlives the machine record it ran on — an agent can be removed, a host
 * can be rebuilt — so a row with no address falls back to the agent id rather
 * than rendering blank: the history of work on a host that is no longer where it
 * was is exactly what this page is for.
 */
function HostCell({ row }: { row: FleetActivityView }) {
  if (!row.address) {
    return (
      <span className="font-mono text-xs text-muted-foreground" title={row.bee_id}>
        {row.bee_id.slice(0, 8)}
      </span>
    );
  }
  return (
    <div className="flex flex-col gap-0.5">
      <span className="font-mono text-sm">{row.address}</span>
      {row.node_name ? (
        <span className="text-xs text-muted-foreground">{row.node_name}</span>
      ) : null}
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  if (status === 'failed') return <Badge variant="destructive">failed</Badge>;
  if (status === 'succeeded') return <Badge variant="outline">succeeded</Badge>;
  if (status === 'running') return <Badge>running</Badge>;
  return <Badge variant="secondary">{status}</Badge>;
}

function when(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

/**
 * How long the work took, for a task that finished.
 *
 * Rendered because the duration is often the whole diagnosis: a disarm that
 * failed in under a second never reached the host, and one that failed after two
 * minutes hit the step timeout — two very different things behind the same
 * message.
 *
 * Measured from claimed_at, not started_at. started_at is written only when a
 * bee reports status "running", and no bee does — the agent keeps that state in
 * its own local store and reports the terminal outcome. So started_at is null on
 * every row ever written, and reading it here made this column show a dash on
 * every task in the fleet. claimed_at is when an agent actually picked the work
 * up, which is the number an operator wants anyway.
 */
function took(row: FleetActivityView): string | null {
  const start = row.started_at ?? row.claimed_at;
  if (!start || !row.finished_at) return null;
  const ms = new Date(row.finished_at).getTime() - new Date(start).getTime();
  if (!Number.isFinite(ms) || ms < 0) return null;
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60_000)}m ${Math.round((ms % 60_000) / 1000)}s`;
}

export const FleetActivity = () => {
  const [kind, setKind] = useState(ANY);
  const [status, setStatus] = useState(ANY);

  const query = useQuery(
    fleetActivityListOptions({
      query: {
        ...(kind === ANY ? {} : { kind }),
        ...(status === ANY ? {} : { status }),
        limit: 100,
      },
    }),
  );

  const rows = query.data?.activity ?? [];

  return (
    <div className="w-full space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-heading text-2xl">Fleet activity</h1>
          <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
            What the fleet's agents have been asked to do, newest first, and what came of it.
            Distinct from the audit trail, which records what people did: asking for a departure is
            one action there, and the work that follows on the host is several here — so a request
            that succeeded and then failed on the machine appears in both, differently.
          </p>
        </div>
        <Button
          size="sm"
          variant="outline"
          disabled={query.isFetching}
          onClick={() => void query.refetch()}
        >
          <IconRefresh className="size-4" />
          {query.isFetching ? 'Refreshing…' : 'Refresh'}
        </Button>
      </div>

      <div className="flex flex-wrap gap-4">
        <div className="space-y-2">
          <Label htmlFor="activity-kind">Task</Label>
          <Select value={kind} onValueChange={(v) => setKind(v ?? ANY)}>
            <SelectTrigger id="activity-kind" className="w-64">
              {/* Explicit rather than a placeholder: Radix resolves a trigger's
                  label from the mounted item, and shows the raw value on the
                  render before the list has opened. */}
              <SelectValue>{KINDS.find((k) => k.value === kind)?.label}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {KINDS.map((k) => (
                <SelectItem key={k.value} value={k.value}>
                  {k.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="activity-status">Outcome</Label>
          <Select value={status} onValueChange={(v) => setStatus(v ?? ANY)}>
            <SelectTrigger id="activity-status" className="w-48">
              <SelectValue>{STATUSES.find((s) => s.value === status)?.label}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {STATUSES.map((s) => (
                <SelectItem key={s.value} value={s.value}>
                  {s.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {query.isError ? (
        <LoadError what="The fleet activity" error={query.error} />
      ) : query.isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : rows.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Nothing recorded yet</CardTitle>
            <CardDescription>
              {kind === ANY && status === ANY
                ? 'No agent has been asked to do anything yet. Work queued before this record existed is not here, so an empty list is not evidence that nothing ran.'
                : 'No tasks match this filter. Widen it before concluding nothing ran.'}
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>When</TableHead>
                <TableHead>Host</TableHead>
                <TableHead>Task</TableHead>
                <TableHead>Outcome</TableHead>
                <TableHead>Took</TableHead>
                <TableHead>Detail</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="whitespace-nowrap text-muted-foreground tabular-nums">
                    {when(r.created_at)}
                  </TableCell>
                  <TableCell>
                    <HostCell row={r} />
                  </TableCell>
                  <TableCell className="font-medium">
                    <div className="flex flex-col gap-0.5">
                      <span>{kindLabel(r)}</span>
                      {r.hive_name ? (
                        <span className="text-xs text-muted-foreground">{r.hive_name}</span>
                      ) : null}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col items-start gap-1">
                      <StatusBadge status={r.status} />
                      {/* CLAIMS, not executions. The counter increments every
                          time an agent picks the task up, including a re-claim
                          after a lease expiry — so "attempt 9" on a task that
                          ran once and succeeded was a straightforwardly false
                          reading of a real signal. Shown only past the first,
                          where being picked up repeatedly is worth noticing. */}
                      {r.attempt > 1 ? (
                        <span
                          className="text-xs text-muted-foreground tabular-nums"
                          title="How many times an agent picked this task up. A lease that expires and is re-claimed counts, so this is not a count of retries."
                        >
                          picked up {r.attempt}×
                        </span>
                      ) : null}
                    </div>
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-muted-foreground tabular-nums">
                    {took(r) ?? '—'}
                  </TableCell>
                  {/* The agent's own words. These are written to be acted on,
                      and paraphrasing one loses the half that says what to do. */}
                  <TableCell className="max-w-md">
                    {r.error ? (
                      <span className="text-xs break-words text-amber-500">{r.error}</span>
                    ) : r.phase ? (
                      <span className="text-xs text-muted-foreground">{r.phase}</span>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
};
