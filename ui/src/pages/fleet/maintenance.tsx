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
} from '@/sdk';
import {
  fleetHivesListOptions,
  fleetMaintenanceRunsListOptions,
  fleetMaintenanceRunsListQueryKey,
  fleetMaintenanceRunsStopMutation,
  fleetMaintenanceSchedulesCreateMutation,
  fleetMaintenanceSchedulesDeleteMutation,
  fleetMaintenanceSchedulesListOptions,
  fleetMaintenanceSchedulesListQueryKey,
  fleetMaintenanceSchedulesUpdateMutation,
} from '@/sdk/@tanstack/react-query.gen';

import { LoadError } from './drain';

/** The standalone pool: every hypervisor in no cluster. */
const STANDALONE = '__standalone__';

const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

const REBOOTS: { value: 'if-needed' | 'always' | 'never'; label: string }[] = [
  { value: 'if-needed', label: 'Reboot if the upgrade needs it' },
  { value: 'always', label: 'Always reboot' },
  { value: 'never', label: 'Never reboot (packages only)' },
];

function errorDetail(e: unknown, fallback: string): string {
  return (e as { detail?: string })?.detail ?? fallback;
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
            Guests on a hypervisor go down with it. The first one that fails or does not come back
            stops the run and stays cordoned.
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
      <TableCell className="text-sm text-muted-foreground tabular-nums">
        {s.kind === 'once' && s.last_window_at
          ? 'done'
          : s.enabled
            ? s.next_openings.length
              ? s.next_openings.map((t) => <div key={t}>{when(t)}</div>)
              : '—'
            : 'paused'}
      </TableCell>
      <TableCell className="text-right whitespace-nowrap">
        <Button size="sm" variant="outline" disabled={update.isPending} onClick={toggle}>
          {s.enabled ? 'Pause' : 'Resume'}
        </Button>{' '}
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
  const done = run.nodes.filter((n) => n.status === 'succeeded').length;
  const skipped = run.nodes.filter((n) => n.status === 'skipped').length;

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
              Window {when(run.window_start)} – {when(run.window_end)} · {done} of{' '}
              {run.nodes.length} upgraded
              {skipped ? ` · ${skipped} skipped` : ''}
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
        {run.nodes.length === 0 ? (
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
                {run.nodes.map((n) => (
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

  const schedules = useQuery(fleetMaintenanceSchedulesListOptions());
  const runs = useQuery({
    ...fleetMaintenanceRunsListOptions({ query: { limit: 50 } }),
    // Poll while anything is in flight: a run moves a node a minute, and an
    // operator watching one should not have to keep pressing refresh.
    refetchInterval: (q) =>
      q.state.data?.runs?.some((r) => r.status === 'running') ? 15_000 : false,
  });

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
            run; one that fails or does not come back does, and stays cordoned until someone looks.
            The next window retries it if its agent is back online.
          </p>
        </div>
        <Button
          size="sm"
          variant="outline"
          disabled={runs.isFetching}
          onClick={() => {
            void runs.refetch();
            void schedules.refetch();
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
          disabled={active}
          title={active ? 'A run is already in progress for this scope' : undefined}
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
