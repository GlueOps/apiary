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
import type { AuditEventView } from '@/sdk';
import { auditEventsListOptions } from '@/sdk/@tanstack/react-query.gen';

import { LoadError } from '../fleet/drain';

const ANY = 'any';

/**
 * Actions, in the order an operator would look for them: the destructive ones
 * first, because those are what someone opens this page to account for.
 */
const ACTIONS: { value: string; label: string }[] = [
  { value: ANY, label: 'Any action' },
  { value: 'hypervisor.depart', label: 'Hypervisor departed' },
  { value: 'hypervisor.arrive', label: 'Hypervisor joined a cluster' },
  { value: 'hypervisor.bootstrap', label: 'Cluster created on a hypervisor' },
  { value: 'hypervisor.cordon', label: 'Hypervisor cordoned' },
  { value: 'hypervisor.uncordon', label: 'Hypervisor uncordoned' },
  { value: 'agent.remove', label: 'Agent removed' },
  { value: 'cluster.create', label: 'Cluster created' },
  { value: 'cluster.delete', label: 'Cluster deleted' },
];

const TARGETS: { value: string; label: string }[] = [
  { value: ANY, label: 'Anything' },
  { value: 'hypervisor', label: 'Hypervisors' },
  { value: 'hive', label: 'Clusters' },
  { value: 'agent', label: 'Agents' },
  { value: 'firewall', label: 'Firewall' },
  { value: 'estate', label: 'Credentials' },
];

/**
 * How an action reads on screen.
 *
 * Falls back to the raw dotted string rather than to a friendly placeholder: an
 * action this page has not been taught about is still a real thing that
 * happened, and hiding it behind "Other" is how a trail quietly stops covering
 * what it claims to.
 */
function actionLabel(action: string): string {
  return ACTIONS.find((a) => a.value === action)?.label ?? action;
}

/**
 * The PRINCIPAL is the whole point of this page, so it gets the treatment.
 *
 * An organization API key is shown as a key, not as a person, because it is not
 * one — it authenticates a tenant. "The automation did this" and "nobody knows
 * who did this" must never look the same here, which is exactly what recording
 * an org key against a nil user would have produced.
 */
function PrincipalCell({ event }: { event: AuditEventView }) {
  const kind = event.principal_kind;
  const short = event.principal_id.slice(0, 8);
  const tone =
    kind === 'user'
      ? 'bg-sky-500/15 text-sky-500'
      : kind === 'org_key'
        ? 'bg-violet-500/15 text-violet-500'
        : kind === 'agent'
          ? 'bg-emerald-500/15 text-emerald-500'
          : 'bg-muted text-muted-foreground';
  const label =
    kind === 'user'
      ? 'user'
      : kind === 'org_key'
        ? 'API key'
        : kind === 'agent'
          ? 'agent'
          : 'system';

  return (
    <div className="flex flex-col gap-1">
      <Badge className={`w-fit ${tone}`}>{label}</Badge>
      <span className="font-mono text-xs text-muted-foreground">{short}</span>
      {event.actor_id ? (
        // Impersonation is BOTH facts at once. Showing only the principal would
        // report a customer's own action; showing only the actor would hide
        // whose account it ran against.
        <span className="text-xs text-amber-500">
          via platform staff {event.actor_id.slice(0, 8)}
        </span>
      ) : null}
    </div>
  );
}

function OutcomeBadge({ outcome }: { outcome: string }) {
  if (outcome === 'refused') {
    return <Badge className="bg-amber-500/15 text-amber-500">refused</Badge>;
  }
  if (outcome === 'failed') {
    return <Badge className="bg-destructive/15 text-destructive">failed</Badge>;
  }
  return <Badge className="bg-emerald-500/15 text-emerald-500">done</Badge>;
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

/** Renders the metadata a row carries, without pretending to know its shape. */
function MetadataCell({ metadata }: { metadata?: Record<string, unknown> }) {
  const entries = Object.entries(metadata ?? {});
  if (entries.length === 0) return <span className="text-muted-foreground">—</span>;
  return (
    <div className="flex flex-col gap-0.5">
      {entries.map(([k, v]) => (
        <span key={k} className="text-xs">
          <span className="text-muted-foreground">{k}: </span>
          <span className="font-mono">{typeof v === 'string' ? v : JSON.stringify(v)}</span>
        </span>
      ))}
    </div>
  );
}

export const AuditTrail = () => {
  const [action, setAction] = useState(ANY);
  const [targetKind, setTargetKind] = useState(ANY);

  const query = useQuery(
    auditEventsListOptions({
      query: {
        ...(action === ANY ? {} : { action }),
        ...(targetKind === ANY ? {} : { target_kind: targetKind }),
        limit: 100,
      },
    }),
  );

  const events = query.data?.events ?? [];

  return (
    <div className="w-full space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-heading text-2xl">Audit trail</h1>
          <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
            What has been done in this organization, newest first, and by whom. Refusals are
            recorded too — someone who tried to depart a node and was told no is a different fact
            from someone who never tried, and only one of them leaves any other trace.
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
          <Label htmlFor="audit-action">Action</Label>
          <Select value={action} onValueChange={(v) => setAction(v ?? ANY)}>
            <SelectTrigger id="audit-action" className="w-72">
              {/* Explicit, not a placeholder: Radix resolves a trigger's label
                  from the mounted item with the matching value, and shows the
                  raw value on the render before the list is open. */}
              <SelectValue>{ACTIONS.find((a) => a.value === action)?.label}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {ACTIONS.map((a) => (
                <SelectItem key={a.value} value={a.value}>
                  {a.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="audit-target">Acted on</Label>
          <Select value={targetKind} onValueChange={(v) => setTargetKind(v ?? ANY)}>
            <SelectTrigger id="audit-target" className="w-56">
              <SelectValue>{TARGETS.find((t) => t.value === targetKind)?.label}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {TARGETS.map((t) => (
                <SelectItem key={t.value} value={t.value}>
                  {t.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {query.isError ? (
        <LoadError what="The audit trail" error={query.error} />
      ) : query.isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : events.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Nothing recorded yet</CardTitle>
            <CardDescription>
              {action === ANY && targetKind === ANY
                ? 'Actions are recorded from the moment the trail was added, so anything done before that is not here — an empty trail is not evidence that nothing happened.'
                : 'No actions match this filter. Widen it before concluding nothing was done.'}
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>When</TableHead>
                <TableHead>Who</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Acted on</TableHead>
                <TableHead>Outcome</TableHead>
                <TableHead>Detail</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {events.map((e) => (
                <TableRow key={e.id}>
                  <TableCell className="whitespace-nowrap text-muted-foreground tabular-nums">
                    {when(e.created_at)}
                  </TableCell>
                  <TableCell>
                    <PrincipalCell event={e} />
                  </TableCell>
                  <TableCell className="font-medium">{actionLabel(e.action)}</TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      <span className="text-sm">{e.target_kind}</span>
                      {e.target_id ? (
                        <span className="font-mono text-xs text-muted-foreground">
                          {e.target_id.slice(0, 8)}
                        </span>
                      ) : null}
                    </div>
                  </TableCell>
                  <TableCell>
                    <OutcomeBadge outcome={e.outcome} />
                  </TableCell>
                  <TableCell className="max-w-md">
                    {e.detail ? (
                      <span className="text-xs text-amber-500">{e.detail}</span>
                    ) : (
                      <MetadataCell metadata={e.metadata} />
                    )}
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
