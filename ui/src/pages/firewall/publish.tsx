import { useState } from 'react';

import { IconArrowBackUp, IconCloudUpload, IconEye } from '@tabler/icons-react';
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
import {
  firewallPublishMutation,
  firewallRevertMutation,
  firewallSnapshotBodyOptions,
  firewallSnapshotsListOptions,
  firewallSnapshotsListQueryKey,
  fleetHivesListOptions,
} from '@/sdk/@tanstack/react-query.gen';

const errorDetail = (err: unknown, fallback: string) =>
  (err as { detail?: string })?.detail ?? fallback;

const when = (iso: string) => new Date(iso).toLocaleString();

/**
 * Publishing is what puts authored rules onto metal.
 *
 * It is deliberately a separate act rather than a consequence of saving a rule.
 * Every edit reaching hosts the moment it is saved makes a half-finished policy
 * a live one, and the window between "allow the new port" and "and here is the
 * deny that makes it meaningful" is exactly when a mistake cannot be undone
 * remotely.
 *
 * Which is also why this page exists at all: until it did, rules could be
 * authored and bound and nothing ever asked a host to enforce them.
 */
export function FirewallPublish() {
  const qc = useQueryClient();
  const [fleetId, setFleetId] = useState('');
  const [reason, setReason] = useState('');
  const [viewing, setViewing] = useState<number | null>(null);
  const [reverting, setReverting] = useState<number | null>(null);

  const hives = useQuery(fleetHivesListOptions());
  const fleets = hives.data?.hives ?? [];

  // Derived during render rather than synchronised in an effect: the default is
  // a function of the loaded list, not a reaction to it, and setting state from
  // an effect here costs a second render on every load for a value that was
  // already knowable in the first.
  const selectedFleet = fleetId || (fleets[0]?.id ?? '');

  const snapshots = useQuery({
    ...firewallSnapshotsListOptions({ path: { hive_id: selectedFleet } }),
    enabled: selectedFleet !== '',
  });

  const body = useQuery({
    ...firewallSnapshotBodyOptions({
      path: { hive_id: selectedFleet, revision: viewing ?? 0 },
    }),
    enabled: selectedFleet !== '' && viewing !== null,
  });

  const invalidate = () =>
    qc.invalidateQueries({
      queryKey: firewallSnapshotsListQueryKey({ path: { hive_id: selectedFleet } }),
    });

  const publish = useMutation({
    ...firewallPublishMutation(),
    onSuccess: (res) => {
      // Unchanged is not a failure. It is what publishing twice does, and saying
      // "published revision 4" for a request that created nothing would make the
      // revision history unreadable against what people remember doing.
      if (res.unchanged) {
        toast.info('Nothing to publish — the rules already match what this fleet is enforcing');
      } else {
        toast.success(
          `Published revision ${res.revision}. Hosts converge on their next heartbeat.`,
        );
      }
      setReason('');
      void invalidate();
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not publish')),
  });

  const revert = useMutation({
    ...firewallRevertMutation(),
    onSuccess: (res) => {
      toast.success(`Republished as revision ${res.revision}`);
      setReverting(null);
      void invalidate();
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not revert')),
  });

  const rows = snapshots.data?.snapshots ?? [];
  const current = rows.find((s) => s.current);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-heading text-2xl">Publish</h1>
        <p className="text-sm text-muted-foreground">
          Rules and bindings are authoring. Publishing renders them into one ruleset and tells a
          fleet's hosts to converge on it — nothing reaches a machine until you do this.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Fleet</CardTitle>
          <CardDescription>
            Each fleet publishes its own revisions. Org-wide bindings are included in every fleet's
            ruleset, so an org-wide change has to be published once per fleet.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Select value={selectedFleet} onValueChange={(v) => setFleetId(String(v))}>
            <SelectTrigger className="w-full max-w-md">
              <SelectValue placeholder="Select a fleet" />
            </SelectTrigger>
            <SelectContent>
              {fleets.map((f) => (
                <SelectItem key={f.id} value={f.id}>
                  {f.name} · {f.node_count} {f.node_count === 1 ? 'host' : 'hosts'}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {selectedFleet ? (
            <div className="space-y-2">
              <Label htmlFor="publish-reason">Reason</Label>
              <Input
                id="publish-reason"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="opening 8006 for the new console"
              />
              <p className="text-xs text-muted-foreground">
                Read later by whoever is deciding whether to back this out, which is the moment it
                matters most.
              </p>
              <Button
                disabled={publish.isPending}
                onClick={() =>
                  publish.mutate({ path: { hive_id: selectedFleet }, body: { reason } })
                }
              >
                <IconCloudUpload /> {publish.isPending ? 'Publishing…' : 'Publish'}
              </Button>
            </div>
          ) : null}
        </CardContent>
      </Card>

      {selectedFleet ? (
        <Card>
          <CardHeader>
            <CardTitle>Revisions</CardTitle>
            <CardDescription>
              {current
                ? `This fleet is telling its hosts to enforce revision ${current.revision}.`
                : 'Nothing published yet — this fleet is not telling its hosts to enforce anything, which is not the same as telling them to enforce nothing.'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {snapshots.isPending ? (
              <p className="text-sm text-muted-foreground">Loading…</p>
            ) : rows.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No revisions. Hosts in this fleet keep whatever firewall they already have — an
                unpublished fleet never flushes a host's rules.
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Revision</TableHead>
                    <TableHead>Published</TableHead>
                    <TableHead>Reason</TableHead>
                    <TableHead>Size</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rows.map((s) => (
                    <TableRow key={s.revision}>
                      <TableCell className="font-medium">
                        {s.revision}
                        {s.current ? (
                          <Badge className="ml-2 bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">
                            current
                          </Badge>
                        ) : null}
                        {s.revert_of ? (
                          <Badge variant="outline" className="ml-2">
                            back to {s.revert_of}
                          </Badge>
                        ) : null}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {when(s.created_at)}
                      </TableCell>
                      <TableCell className="max-w-[20rem] truncate text-sm">
                        {s.reason || '—'}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">{s.bytes} B</TableCell>
                      <TableCell className="space-x-1 text-right">
                        <Button size="sm" variant="ghost" onClick={() => setViewing(s.revision)}>
                          <IconEye /> View
                        </Button>
                        {s.current ? null : (
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => setReverting(s.revision)}
                          >
                            <IconArrowBackUp /> Restore
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
      ) : null}

      <Dialog open={viewing !== null} onOpenChange={(o) => !o && setViewing(null)}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>Revision {viewing}</DialogTitle>
            <DialogDescription>
              The exact text a host loads. Compare it against <code>nft list ruleset</code> on the
              machine — that comparison is the only way to answer "is this host running what we
              think" from the host itself.
            </DialogDescription>
          </DialogHeader>
          <pre className="max-h-[60vh] overflow-auto rounded bg-muted p-3 text-xs">
            {body.isPending ? 'Loading…' : (body.data?.content ?? '')}
          </pre>
        </DialogContent>
      </Dialog>

      <Dialog open={reverting !== null} onOpenChange={(o) => !o && setReverting(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Restore revision {reverting}?</DialogTitle>
            <DialogDescription>
              This publishes revision {reverting}'s ruleset as a NEW revision. The number goes up,
              never down — a host decides whether to converge by comparing revisions, and a number
              that moved backwards would leave it pinned to the very ruleset you are backing out.
              <br />
              <br />
              It restores that revision's text verbatim, not a re-render of the rules as they stand
              now. Your current rules are untouched; publishing again returns to them.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setReverting(null)}>
              Cancel
            </Button>
            <Button
              disabled={revert.isPending}
              onClick={() =>
                reverting !== null &&
                revert.mutate({
                  path: { hive_id: selectedFleet },
                  body: { to_revision: reverting, reason: `restoring revision ${reverting}` },
                })
              }
            >
              {revert.isPending ? 'Restoring…' : 'Restore'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
