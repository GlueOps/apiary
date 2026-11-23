import { useMemo, useState } from 'react';

import { IconPlayerPlay, IconPlus, IconRefresh, IconTrash } from '@tabler/icons-react';
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type {
  CreateMaintenanceScheduleRequestBody,
  MaintenanceRunView,
  MaintenanceScheduleView,
  MaintenanceStrikesView,
} from '@/sdk';
import {
  fleetHivesListOptions,
  fleetMaintenanceAcknowledgeMutation,
  fleetMaintenanceRunsListOptions,
  fleetMaintenanceRunsListQueryKey,
  fleetMaintenanceRunsStopMutation,
  fleetMaintenanceSchedulesCreateMutation,
  fleetMaintenanceSchedulesDeleteMutation,
  fleetMaintenanceSchedulesListOptions,
  fleetMaintenanceSchedulesListQueryKey,
  fleetMaintenanceSchedulesUpdateMutation,
  fleetMaintenanceStrikesListOptions,
  fleetMaintenanceStrikesListQueryKey,
} from '@/sdk/@tanstack/react-query.gen';

import { apiErrorMessage } from './api-error';
import { LoadError } from './drain';

/** The standalone pool: every hypervisor in no cluster. */
const STANDALONE = '__standalone__';

const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

const REBOOTS: { value: 'if-needed' | 'always' | 'never'; label: string }[] = [
  { value: 'if-needed', label: 'Reboot if the upgrade needs it' },
  { value: 'always', label: 'Always reboot' },
  { value: 'never', label: 'Never reboot (packages only)' },
];

const errorDetail = apiErrorMessage;

/**
 * A one-off window that has opened, or been switched off, is finished: it will
 * never open again, so "Resume" on it would do nothing or be refused.
 */
function finishedOnce(s: MaintenanceScheduleView): boolean {
  return s.kind === 'once' && (!!s.last_window_at || (!s.enabled && !s.next_openings?.length));
}

/**
 * A one-off window that is open and has not been picked up yet: a run is about
 * to start. "Run now" again would queue a second one behind it.
 */
function waitingToStart(s: MaintenanceScheduleView): boolean {
  return s.kind === 'once' && s.enabled && !s.last_window_at && s.open_now;
}

function when(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    weekday: 'short',
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function hours(minutes: number): string {
  if (minutes % 60 === 0) return `${minutes / 60}h`;
  if (minutes < 60) return `${minutes}m`;
  return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
}

/** The window in words, in the zone it was written in — not the viewer's. */
function describe(s: MaintenanceScheduleView): string {
  if (s.kind === 'once') return `Once, ${when(s.once_at)} for ${hours(s.duration_minutes)}`;
  const days = (s.weekdays ?? []).map((d) => DAYS[d]).join(', ');
  return `${days} at ${s.start_time} ${s.timezone} for ${hours(s.duration_minutes)}`;
}

function inScope(scope: string, row: { hive_id?: string; standalone: boolean }): boolean {
  return scope === STANDALONE ? row.standalone : row.hive_id === scope;
}

/** The budget in words: how much one run works, and how many may fail. */
function describeBudget(b: { max_nodes_per_run?: number; maintenance_pct?: number }): string {
  const size = b.max_nodes_per_run ? `up to ${b.max_nodes_per_run} per run` : 'every hypervisor';
  const pct = b.maintenance_pct ?? 0;
  return pct
    ? `${size}; carries on until more than ${pct}% have failed`
    : `${size}; stops at the first failure`;
}

function RunBadge({ status }: { status: string }) {
  if (status === 'failed') return <Badge variant="destructive">failed</Badge>;
  if (status === 'running') return <Badge>running</Badge>;
  if (status === 'succeeded') return <Badge variant="outline">succeeded</Badge>;
  return <Badge variant="secondary">{status}</Badge>;
}

function NodeBadge({ status }: { status: string }) {
  if (status === 'failed') return <Badge variant="destructive">failed</Badge>;
  if (status === 'upgrading') return <Badge>upgrading</Badge>;
  if (status === 'succeeded') return <Badge variant="outline">done</Badge>;
  if (status === 'skipped') return <Badge variant="secondary">skipped</Badge>;
  return <Badge variant="secondary">waiting</Badge>;
}

type Draft = {
  name: string;
  kind: 'recurring' | 'once';
  weekdays: number[];
  startTime: string;
  timezone: string;
  onceAt: string;
  duration: number;
  reboot: 'if-needed' | 'always' | 'never';
  maxNodes: number;
  pct: number;
};

function blankDraft(): Draft {
  return {
    name: '',
    kind: 'recurring',
    weekdays: [0],
    startTime: '02:00',
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
    onceAt: '',
    duration: 120,
    reboot: 'if-needed',
    maxNodes: 0,
    pct: 0,
  };
}

/**
 * Writing a window. The consequence is stated in the dialog itself, not in a
 * help page: a window reboots hypervisors with their guests on them.
 */
function WindowDialog({
  open,
  onOpenChange,
  scope,
  scopeName,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  scope: string;
  scopeName: string;
}) {
  const qc = useQueryClient();
  const [d, setD] = useState<Draft>(blankDraft);
  const create = useMutation({
    ...fleetMaintenanceSchedulesCreateMutation(),
    onSuccess: (res) => {
      const next = res.next_openings?.[0];
      toast.success(next ? `Scheduled. Next window opens ${when(next)}.` : 'Scheduled.');
      void qc.invalidateQueries({ queryKey: fleetMaintenanceSchedulesListQueryKey() });
      setD(blankDraft());
      onOpenChange(false);
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not schedule the window')),
  });

  const toggleDay = (day: number) =>
    setD((cur) => ({
      ...cur,
      weekdays: cur.weekdays.includes(day)
        ? cur.weekdays.filter((x) => x !== day)
        : [...cur.weekdays, day].sort(),
    }));

  const submit = () => {
    const body: CreateMaintenanceScheduleRequestBody = {
      name: d.name.trim(),
      kind: d.kind,
      duration_minutes: d.duration,
      reboot: d.reboot,
      enabled: true,
      max_nodes_per_run: d.maxNodes,
      maintenance_pct: d.pct,
      ...(scope === STANDALONE ? { standalone: true } : { hive_id: scope }),
      ...(d.kind === 'recurring'
        ? { weekdays: d.weekdays, start_time: d.startTime, timezone: d.timezone }
        : { once_at: d.onceAt ? new Date(d.onceAt).toISOString() : undefined }),
    };
    create.mutate({ body });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>New maintenance window — {scopeName}</DialogTitle>
          <DialogDescription>
            In the window each hypervisor is cordoned, upgraded and rebooted in turn, one at a time.
            Guests on a hypervisor go down with it. One that fails or does not come back stays
            cordoned, and by default stops the run.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="mw-name">Name</Label>
            <Input
              id="mw-name"
              value={d.name}
              maxLength={100}
              placeholder="Weekly patching"
              onChange={(e) => setD({ ...d, name: e.target.value })}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="mw-kind">Repeats</Label>
            <Select
              value={d.kind}
              onValueChange={(v) => setD({ ...d, kind: (v as Draft['kind']) ?? 'recurring' })}
            >
              <SelectTrigger id="mw-kind" className="w-full">
                <SelectValue>{d.kind === 'recurring' ? 'Every week' : 'Once'}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="recurring">Every week</SelectItem>
                <SelectItem value="once">Once</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {d.kind === 'recurring' ? (
            <>
              <div className="space-y-2">
                <Label>Days</Label>
                <div className="flex flex-wrap gap-1">
                  {DAYS.map((label, i) => (
                    <Button
                      key={label}
                      type="button"
                      size="sm"
                      variant={d.weekdays.includes(i) ? 'default' : 'outline'}
                      aria-pressed={d.weekdays.includes(i)}
                      onClick={() => toggleDay(i)}
                    >
                      {label}
                    </Button>
                  ))}
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label htmlFor="mw-start">Opens at</Label>
                  <Input
                    id="mw-start"
                    type="time"
                    value={d.startTime}
                    onChange={(e) => setD({ ...d, startTime: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="mw-tz">Time zone</Label>
                  <Input
                    id="mw-tz"
                    value={d.timezone}
                    placeholder="Europe/Helsinki"
                    onChange={(e) => setD({ ...d, timezone: e.target.value })}
                  />
                </div>
              </div>
            </>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="mw-once">Opens at (your local time)</Label>
              <Input
                id="mw-once"
                type="datetime-local"
                value={d.onceAt}
                onChange={(e) => setD({ ...d, onceAt: e.target.value })}
              />
            </div>
          )}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="mw-duration">Length (minutes)</Label>
              <Input
                id="mw-duration"
                type="number"
                min={15}
                max={720}
                step={15}
                value={d.duration}
                onChange={(e) => setD({ ...d, duration: Number(e.target.value) })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="mw-reboot">Reboot</Label>
              <Select
                value={d.reboot}
                onValueChange={(v) => setD({ ...d, reboot: (v as Draft['reboot']) ?? 'if-needed' })}
              >
                <SelectTrigger id="mw-reboot" className="w-full">
                  <SelectValue>{REBOOTS.find((r) => r.value === d.reboot)?.label}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {REBOOTS.map((r) => (
                    <SelectItem key={r.value} value={r.value}>
                      {r.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="mw-max-nodes">Hypervisors per run</Label>
              <Input
                id="mw-max-nodes"
                type="number"
                min={0}
                max={1000}
                value={d.maxNodes}
                onChange={(e) => setD({ ...d, maxNodes: Number(e.target.value) })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="mw-pct">Failure budget (%)</Label>
              <Input
                id="mw-pct"
                type="number"
                min={0}
                max={50}
                value={d.pct}
                onChange={(e) => setD({ ...d, pct: Number(e.target.value) })}
              />
            </div>
          </div>
          <p className="text-xs text-muted-foreground">
            {describeBudget({ max_nodes_per_run: d.maxNodes, maintenance_pct: d.pct })}. 0
            hypervisors per run means all of them; the rest are left for the next window, which
            starts with the least recently maintained. In a cluster a run carries on past a failure
            only while the cluster keeps a vote to spare. A hypervisor that fails in two runs in a
            row is skipped until someone acknowledges it.
          </p>
          <p className="text-xs text-muted-foreground">
            A hypervisor already rebooting when the window closes is allowed to finish; no new one
            starts.
          </p>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={create.isPending || !d.name.trim()} onClick={submit}>
            {create.isPending ? 'Scheduling…' : 'Schedule'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** "Run now" is a one-off window that opens immediately. */
function RunNowDialog({
  open,
  onOpenChange,
  scope,
  scopeName,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  scope: string;
  scopeName: string;
}) {
  const qc = useQueryClient();
  const [reboot, setReboot] = useState<Draft['reboot']>('if-needed');
  const create = useMutation({
    ...fleetMaintenanceSchedulesCreateMutation(),
    onSuccess: () => {
      toast.success('Maintenance starts within a minute.');
      void qc.invalidateQueries({ queryKey: fleetMaintenanceSchedulesListQueryKey() });
      void qc.invalidateQueries({ queryKey: fleetMaintenanceRunsListQueryKey() });
      onOpenChange(false);
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not start maintenance')),
  });
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Run maintenance now — {scopeName}</DialogTitle>
          <DialogDescription>
            Every hypervisor in scope is upgraded in turn, starting within a minute.{' '}
            {reboot === 'never'
              ? 'Nothing is rebooted.'
              : 'Guests on each hypervisor go down while it reboots.'}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="rn-reboot">Reboot</Label>
          <Select value={reboot} onValueChange={(v) => setReboot((v as Draft['reboot']) ?? reboot)}>
            <SelectTrigger id="rn-reboot" className="w-full">
              <SelectValue>{REBOOTS.find((r) => r.value === reboot)?.label}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {REBOOTS.map((r) => (
                <SelectItem key={r.value} value={r.value}>
                  {r.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            disabled={create.isPending}
            onClick={() =>
              create.mutate({
                body: {
                  name: `Run now ${new Date().toLocaleString()}`.slice(0, 100),
                  kind: 'once',
                  once_at: new Date().toISOString(),
                  duration_minutes: 240,
                  reboot,
                  enabled: true,
                  ...(scope === STANDALONE ? { standalone: true } : { hive_id: scope }),
                },
              })
            }
          >
            {create.isPending ? 'Starting…' : 'Start'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ScheduleRow({ s }: { s: MaintenanceScheduleView }) {
  const qc = useQueryClient();
  const refresh = () =>
    void qc.invalidateQueries({ queryKey: fleetMaintenanceSchedulesListQueryKey() });
  const update = useMutation({
    ...fleetMaintenanceSchedulesUpdateMutation(),
    onSuccess: refresh,
    onError: (e) => toast.error(errorDetail(e, 'Could not change the window')),
  });
  const remove = useMutation({
    ...fleetMaintenanceSchedulesDeleteMutation(),
    onSuccess: () => {
      toast.success('Window deleted. Past runs are kept.');
      refresh();
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not delete the window')),
  });

  const toggle = () =>
    update.mutate({
      path: { id: s.id },
      body: {
        name: s.name,
        kind: s.kind,
        weekdays: s.weekdays,
        start_time: s.start_time,
        timezone: s.timezone,
        once_at: s.once_at,
        duration_minutes: s.duration_minutes,
        reboot: s.reboot as Draft['reboot'],
        enabled: !s.enabled,
        // Carried over: the update replaces the window, so leaving these out
        // would quietly reset the budget to zero on Pause.
        max_nodes_per_run: s.max_nodes_per_run,
        maintenance_pct: s.maintenance_pct,
      },
    });

  return (
    <TableRow>
      <TableCell className="font-medium">
        <div className="flex items-center gap-2">
          {s.name}
          {s.open_now ? <Badge>open now</Badge> : null}
        </div>
      </TableCell>
      <TableCell className="text-sm">{describe(s)}</TableCell>
      <TableCell className="text-sm text-muted-foreground">
        {REBOOTS.find((r) => r.value === s.reboot)?.label ?? s.reboot}
      </TableCell>
      <TableCell className="text-sm text-muted-foreground">{describeBudget(s)}</TableCell>
      <TableCell className="text-sm text-muted-foreground tabular-nums">
        {s.kind === 'once' && s.last_window_at
          ? 'done'
          : s.enabled
            ? s.next_openings?.length
              ? s.next_openings.map((t) => <div key={t}>{when(t)}</div>)
              : '—'
            : 'paused'}
      </TableCell>
      <TableCell className="text-right whitespace-nowrap">
        {finishedOnce(s) ? null : (
          <>
            <Button size="sm" variant="outline" disabled={update.isPending} onClick={toggle}>
              {s.enabled ? 'Pause' : 'Resume'}
            </Button>{' '}
          </>
        )}
        <Button
          size="sm"
          variant="ghost"
          aria-label={`Delete ${s.name}`}
          disabled={remove.isPending}
          onClick={() => {
            if (window.confirm(`Delete the window "${s.name}"? A run in progress is not stopped.`))
              remove.mutate({ path: { id: s.id } });
          }}
        >
          <IconTrash className="size-4" />
        </Button>
      </TableCell>
    </TableRow>
  );
}

/**
 * Hypervisors with maintenance failures that still count. A flagged one is
 * skipped by every window until somebody acknowledges it — which lets
 * maintenance try it again, and does NOT uncordon it.
 */
function StrikeRow({ h }: { h: MaintenanceStrikesView }) {
  const qc = useQueryClient();
  const ack = useMutation({
    ...fleetMaintenanceAcknowledgeMutation(),
    onSuccess: () => {
      toast.success(`${h.name} will be tried again in its next window.`);
      void qc.invalidateQueries({ queryKey: fleetMaintenanceStrikesListQueryKey() });
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not acknowledge')),
  });
  return (
    <TableRow>
      <TableCell className="font-mono text-sm">{h.name}</TableCell>
      <TableCell>
        <div className="flex flex-wrap items-center gap-1">
          {h.flagged ? (
            <Badge variant="destructive">skipped until acknowledged</Badge>
          ) : (
            <Badge variant="secondary">
              {h.strikes} failed run{h.strikes === 1 ? '' : 's'}
            </Badge>
          )}
          {!h.schedulable ? <Badge variant="outline">cordoned</Badge> : null}
        </div>
      </TableCell>
      <TableCell className="max-w-md text-xs break-words text-muted-foreground">
        {when(h.latest_at)}: {h.latest_reason}
      </TableCell>
      <TableCell className="text-right whitespace-nowrap">
        <Button
          size="sm"
          variant="outline"
          disabled={ack.isPending}
          onClick={() => {
            if (
              window.confirm(
                `Acknowledge ${h.name}? Its next maintenance window will upgrade and reboot it ` +
                  'again. It stays cordoned until that succeeds or someone uncordons it.',
              )
            )
              ack.mutate({ path: { id: h.hypervisor_id } });
          }}
        >
          Acknowledge
        </Button>
      </TableCell>
    </TableRow>
  );
}

function RunCard({ run }: { run: MaintenanceRunView }) {
  const qc = useQueryClient();
  const stop = useMutation({
    ...fleetMaintenanceRunsStopMutation(),
    onSuccess: () => {
      toast.success('The run stops once the hypervisor in progress is back.');
      void qc.invalidateQueries({ queryKey: fleetMaintenanceRunsListQueryKey() });
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not stop the run')),
  });
  // Nullable on the wire: the API sends null, not [], for a run with no nodes.
  const nodes = run.nodes ?? [];
  const done = nodes.filter((n) => n.status === 'succeeded').length;
  const skipped = nodes.filter((n) => n.status === 'skipped').length;

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div>
            <CardTitle className="flex items-center gap-2 text-base">
              {run.schedule_name || 'Maintenance run'} <RunBadge status={run.status} />
              {run.stop_requested && run.status === 'running' ? (
                <Badge variant="secondary">stopping</Badge>
              ) : null}
            </CardTitle>
            <CardDescription>
              Window {when(run.window_start)} – {when(run.window_end)} · {done} of {nodes.length}{' '}
              upgraded
              {skipped ? ` · ${skipped} skipped` : ''} · {describeBudget(run)}
            </CardDescription>
          </div>
          {run.status === 'running' && !run.stop_requested ? (
            <Button
              size="sm"
              variant="outline"
              disabled={stop.isPending}
              onClick={() => stop.mutate({ path: { id: run.id } })}
            >
              Stop after current
            </Button>
          ) : null}
        </div>
        {run.error ? <p className="text-sm text-amber-500">{run.error}</p> : null}
      </CardHeader>
      <CardContent>
        {nodes.length === 0 ? (
          <p className="text-sm text-muted-foreground">No hypervisors were in scope.</p>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Hypervisor</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Detail</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {nodes.map((n) => (
                  <TableRow key={n.hypervisor_id}>
                    <TableCell className="font-mono text-sm">{n.name}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap items-center gap-1">
                        <NodeBadge status={n.status} />
                        {n.left_cordoned ? <Badge variant="outline">left cordoned</Badge> : null}
                      </div>
                    </TableCell>
                    <TableCell className="max-w-md text-xs break-words text-muted-foreground">
                      {n.status === 'upgrading' && n.deadline
                        ? `in progress; given up on at ${when(n.deadline)}`
                        : n.reason}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export const Maintenance = () => {
  const hives = useQuery(fleetHivesListOptions());
  const clusters = useMemo(() => hives.data?.hives ?? [], [hives.data]);
  const [picked, setPicked] = useState<string | null>(null);
  const scope = picked ?? clusters[0]?.id ?? STANDALONE;
  const scopeName =
    scope === STANDALONE
      ? 'Standalone hypervisors'
      : (clusters.find((c) => c.id === scope)?.name ?? 'Cluster');

  const schedules = useQuery({
    ...fleetMaintenanceSchedulesListOptions(),
    // Poll while a window is waiting to be picked up, so the page sees its run
    // start without a refresh.
    refetchInterval: (q) => (q.state.data?.schedules?.some(waitingToStart) ? 15_000 : false),
  });
  const starting = (schedules.data?.schedules ?? []).some(
    (s) => inScope(scope, s) && waitingToStart(s),
  );
  const runs = useQuery({
    ...fleetMaintenanceRunsListOptions({ query: { limit: 50 } }),
    // Poll while anything is in flight: a run moves a node a minute, and an
    // operator watching one should not have to keep pressing refresh.
    refetchInterval: (q) =>
      starting || q.state.data?.runs?.some((r) => r.status === 'running') ? 15_000 : false,
  });

  const strikes = useQuery(fleetMaintenanceStrikesListOptions());
  const myStrikes = (strikes.data?.hypervisors ?? []).filter((h) =>
    scope === STANDALONE ? !h.hive_id : h.hive_id === scope,
  );

  const [creating, setCreating] = useState(false);
  const [runningNow, setRunningNow] = useState(false);

  const mine = (schedules.data?.schedules ?? []).filter((s) => inScope(scope, s));
  const myRuns = (runs.data?.runs ?? []).filter((r) => inScope(scope, r));
  const active = myRuns.some((r) => r.status === 'running');

  return (
    <div className="w-full space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-heading text-2xl">Maintenance</h1>
          <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
            Scheduled upgrades and reboots, one hypervisor at a time. A hypervisor with no agent, an
            offline agent, or a departure under way is skipped with the reason and does not stop the
            run. One that fails or does not come back stays cordoned, and stops the run unless the
            window has a failure budget. The next window retries it, but after two failed runs in a
            row it is skipped until someone acknowledges it below.
          </p>
        </div>
        <Button
          size="sm"
          variant="outline"
          disabled={runs.isFetching}
          onClick={() => {
            void runs.refetch();
            void schedules.refetch();
            void strikes.refetch();
          }}
        >
          <IconRefresh className="size-4" />
          Refresh
        </Button>
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <div className="space-y-2">
          <Label htmlFor="mw-scope">Scope</Label>
          <Select value={scope} onValueChange={(v) => setPicked(v ?? null)}>
            <SelectTrigger id="mw-scope" className="w-72">
              <SelectValue>{scopeName}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {clusters.map((c) => (
                <SelectItem key={c.id} value={c.id}>
                  {c.name}
                </SelectItem>
              ))}
              <SelectItem value={STANDALONE}>Standalone hypervisors</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button size="sm" onClick={() => setCreating(true)}>
          <IconPlus className="size-4" />
          New window
        </Button>
        <Button
          size="sm"
          variant="outline"
          disabled={active || starting}
          title={
            active
              ? 'A run is already in progress for this scope'
              : starting
                ? 'A run for this scope is about to start'
                : undefined
          }
          onClick={() => setRunningNow(true)}
        >
          <IconPlayerPlay className="size-4" />
          Run now
        </Button>
      </div>
      {scope === STANDALONE ? (
        <p className="text-sm text-muted-foreground">
          Every hypervisor in no cluster, decided when each run starts — a hypervisor that joins a
          cluster moves onto that cluster's windows.
        </p>
      ) : null}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Windows</CardTitle>
        </CardHeader>
        <CardContent>
          {schedules.isError ? (
            <LoadError what="Maintenance windows" error={schedules.error} />
          ) : schedules.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : mine.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No windows for {scopeName}. Nothing here is upgraded on a schedule.
            </p>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>When</TableHead>
                    <TableHead>Reboot</TableHead>
                    <TableHead>Budget</TableHead>
                    <TableHead>Next</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {mine.map((s) => (
                    <ScheduleRow key={s.id} s={s} />
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {myStrikes.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Needs attention</CardTitle>
            <CardDescription>
              Hypervisors whose upgrade failed in a maintenance run with no success since.
              Acknowledging lets maintenance try one again; it does not uncordon it.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Hypervisor</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Latest failure</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {myStrikes.map((h) => (
                    <StrikeRow key={h.hypervisor_id} h={h} />
                  ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      ) : null}

      <div className="space-y-3">
        <h2 className="font-heading text-lg">Runs</h2>
        {runs.isError ? (
          <LoadError what="Maintenance runs" error={runs.error} />
        ) : runs.isLoading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : myRuns.length === 0 ? (
          <p className="text-sm text-muted-foreground">No runs yet for {scopeName}.</p>
        ) : (
          myRuns.map((r) => <RunCard key={r.id} run={r} />)
        )}
      </div>

      <WindowDialog
        open={creating}
        onOpenChange={setCreating}
        scope={scope}
        scopeName={scopeName}
      />
      <RunNowDialog
        open={runningNow}
        onOpenChange={setRunningNow}
        scope={scope}
        scopeName={scopeName}
      />
    </div>
  );
};
