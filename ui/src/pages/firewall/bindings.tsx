import { useMemo, useState } from 'react';

import { IconPlus, IconTrash, IconWorld } from '@tabler/icons-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
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
  firewallBindingsCreateMutation,
  firewallBindingsDeleteMutation,
  firewallBindingsListOptions,
  firewallBindingsListQueryKey,
  firewallBindingsUpdateMutation,
  firewallPreviewQueryKey,
  firewallRulesetsListOptions,
} from '@/sdk/@tanstack/react-query.gen';
import type { BindingView } from '@/sdk/types.gen';

import { BlastRadius } from './bindings-blast-radius';
import { ORG_WIDE, describeScope, useFleets } from './bindings-fleets';
import { FirewallPreview } from './preview';

const DEFAULT_PRIORITY = 100;

function conflictMessage(scopeLabel: string) {
  return `That ruleset is already bound ${scopeLabel === 'Org-wide' ? 'org-wide' : `to ${scopeLabel}`}. A ruleset binds once per scope — change the existing binding's priority instead of adding a second one.`;
}

export const FirewallBindings = () => {
  const queryClient = useQueryClient();

  const [previewScope, setPreviewScope] = useState<string>(ORG_WIDE);
  const [createOpen, setCreateOpen] = useState(false);
  const [newRulesetId, setNewRulesetId] = useState('');
  const [newScope, setNewScope] = useState<string>(ORG_WIDE);
  const [newPriority, setNewPriority] = useState(String(DEFAULT_PRIORITY));
  const [newEnabled, setNewEnabled] = useState(true);
  const [editing, setEditing] = useState<BindingView | null>(null);
  const [editPriority, setEditPriority] = useState(String(DEFAULT_PRIORITY));
  const [editEnabled, setEditEnabled] = useState(true);
  const [unbinding, setUnbinding] = useState<BindingView | null>(null);

  const bindingsQuery = useQuery(firewallBindingsListOptions());
  const rulesetsQuery = useQuery(firewallRulesetsListOptions());

  const bindings = useMemo(() => bindingsQuery.data?.bindings ?? [], [bindingsQuery.data]);
  const rulesets = useMemo(() => rulesetsQuery.data?.rulesets ?? [], [rulesetsQuery.data]);
  const { fleets } = useFleets(bindings);

  const rulesetNames = useMemo(() => new Map(rulesets.map((r) => [r.id, r.name])), [rulesets]);

  // Both the list and the preview move whenever a binding does: the preview is
  // computed server-side from exactly these rows, so refreshing one without the
  // other leaves the screen contradicting itself.
  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: firewallBindingsListQueryKey() });
    void queryClient.invalidateQueries({ queryKey: firewallPreviewQueryKey() });
  };

  const create = useMutation({
    ...firewallBindingsCreateMutation(),
    onSuccess: () => {
      toast.success('Ruleset bound');
      setCreateOpen(false);
      setNewRulesetId('');
      setNewPriority(String(DEFAULT_PRIORITY));
      refresh();
    },
    onError: (e) => {
      const err = e as { status?: number; detail?: string };
      // 409 is the uniqueness rule, not a fault. Reported as a generic failure
      // it reads as "try again", and the operator retries the one thing that
      // cannot work.
      if (err.status === 409) {
        toast.error(
          conflictMessage(describeScope(newScope === ORG_WIDE ? undefined : newScope, fleets)),
        );
        return;
      }
      toast.error(err.detail ?? 'Could not bind that ruleset');
    },
  });

  const update = useMutation({
    ...firewallBindingsUpdateMutation(),
    // The endpoint answers with {deleted}, not the updated binding, so there is
    // nothing here worth writing into the cache — only a refetch tells the truth.
    onSuccess: () => {
      toast.success('Binding updated');
      setEditing(null);
      refresh();
    },
    onError: (e) =>
      toast.error((e as { detail?: string })?.detail ?? 'Could not update that binding'),
  });

  const remove = useMutation({
    ...firewallBindingsDeleteMutation(),
    onSuccess: () => {
      toast.success('Ruleset unbound');
      setUnbinding(null);
      refresh();
    },
    onError: (e) =>
      toast.error((e as { detail?: string })?.detail ?? 'Could not unbind that ruleset'),
  });

  const ordered = useMemo(
    () =>
      [...bindings].sort(
        (a, b) => a.priority - b.priority || a.ruleset_name.localeCompare(b.ruleset_name),
      ),
    [bindings],
  );

  const previewHive = previewScope === ORG_WIDE ? undefined : previewScope;
  const appliesToPreview = (b: BindingView) => !b.hive_id || b.hive_id === previewHive;

  const newHiveId = newScope === ORG_WIDE ? undefined : newScope;
  const duplicate = bindings.some(
    (b) => b.ruleset_id === newRulesetId && (b.hive_id ?? undefined) === newHiveId,
  );

  return (
    <div className="w-full space-y-6 p-6">
      <div>
        <h1 className="font-heading text-2xl">Firewall bindings</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          A ruleset does nothing until it is bound. Bind it to a fleet, or org-wide so it also
          covers hosts that belong to no fleet — which is every host mid-migration, at its most
          exposed moment.
        </p>
      </div>

      <FirewallPreview
        scope={previewScope}
        onScopeChange={setPreviewScope}
        fleets={fleets}
        bindings={bindings}
        rulesetNames={rulesetNames}
      />

      <Card>
        <CardHeader>
          <CardTitle>Bindings</CardTitle>
          <CardDescription>
            {ordered.length === 0
              ? 'Nothing is bound.'
              : `${ordered.length} binding${ordered.length === 1 ? '' : 's'}. Lower priority renders first, and priority outranks the rule positions inside each ruleset.`}
          </CardDescription>
          <CardAction>
            <Button
              size="sm"
              variant="outline"
              disabled={rulesets.length === 0}
              onClick={() => {
                setNewRulesetId(rulesets[0]?.id ?? '');
                setCreateOpen(true);
              }}
            >
              <IconPlus /> Bind a ruleset
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          {bindingsQuery.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : bindingsQuery.isError ? (
            // Showing an empty table here would say "nothing is bound", which is
            // a statement about the firewall. This is a statement about the
            // request, and the difference decides whether someone goes looking.
            <div className="rounded-lg bg-destructive/10 p-4 text-sm">
              <p className="font-medium">Bindings could not be loaded.</p>
              <p className="mt-1 text-muted-foreground">
                {(bindingsQuery.error as { detail?: string })?.detail ??
                  'The server did not answer. This is not a claim that nothing is bound.'}
              </p>
            </div>
          ) : ordered.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              A binding attaches one ruleset to one fleet, or to the whole organization. Start with
              an org-wide binding carrying your baseline ruleset — a host between fleets has no
              fleet-scoped bindings at all, so that is the only thing covering it.
              {rulesets.length === 0
                ? ' There are no rulesets to bind yet; create one first.'
                : null}
            </p>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-20">Priority</TableHead>
                    <TableHead>Ruleset</TableHead>
                    <TableHead>Scope</TableHead>
                    <TableHead>State</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {ordered.map((b) => {
                    const here = appliesToPreview(b);
                    return (
                      <TableRow key={b.id} className={here ? undefined : 'opacity-50'}>
                        <TableCell className="tabular-nums">{b.priority}</TableCell>
                        <TableCell className="font-medium">
                          {b.ruleset_name}
                          {here && b.enabled ? (
                            <span className="block text-xs text-muted-foreground">
                              in the preview above
                            </span>
                          ) : null}
                        </TableCell>
                        <TableCell>
                          {b.hive_id ? (
                            <span className="text-sm">{describeScope(b.hive_id, fleets)}</span>
                          ) : (
                            <span className="flex items-center gap-1 text-sm">
                              <IconWorld className="size-4 text-muted-foreground" /> Org-wide
                            </span>
                          )}
                        </TableCell>
                        <TableCell>
                          {b.enabled ? (
                            <Badge className="bg-emerald-500/15 text-emerald-500">rendering</Badge>
                          ) : (
                            <Badge className="bg-muted text-muted-foreground">disabled</Badge>
                          )}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-2">
                            <Button
                              size="xs"
                              variant="outline"
                              onClick={() => {
                                setEditing(b);
                                setEditPriority(String(b.priority));
                                setEditEnabled(b.enabled);
                              }}
                            >
                              Edit
                            </Button>
                            <Button
                              size="xs"
                              variant="ghost"
                              onClick={() => setUnbinding(b)}
                              aria-label={`Unbind ${b.ruleset_name}`}
                            >
                              <IconTrash /> Unbind
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/*
        Suppressed while the bindings request is failing or still in flight.
        `bindings` falls back to [], so rendering regardless would state that
        every ruleset lands on 0 fleets — a confident, specific, wrong answer to
        the one question this card exists to answer, shown directly beneath a
        card that correctly says the data could not be loaded.
      */}
      {bindingsQuery.isError || bindingsQuery.isPending ? null : (
        <BlastRadius rulesets={rulesets} bindings={bindings} fleets={fleets} />
      )}

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Bind a ruleset</DialogTitle>
            <DialogDescription>
              The same ruleset can be bound org-wide and to a fleet at once — that is the intended
              base-plus-datacenter pattern, and both copies render.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <Label htmlFor="binding-ruleset">Ruleset</Label>
            <Select value={newRulesetId} onValueChange={(next) => setNewRulesetId(String(next))}>
              <SelectTrigger id="binding-ruleset" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {rulesets.map((r) => (
                  <SelectItem key={r.id} value={r.id}>
                    {r.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="binding-scope">Scope</Label>
            <Select value={newScope} onValueChange={(next) => setNewScope(String(next))}>
              <SelectTrigger id="binding-scope" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ORG_WIDE}>Org-wide — every host, fleet or not</SelectItem>
                {fleets.map((f) => (
                  <SelectItem key={f.id} value={f.id}>
                    {f.label}
                    {f.hostnames.length ? ` · ${f.hostnames.join(', ')}` : ' · no agents yet'}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Org-wide is the one that survives a host moving between fleets.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="binding-priority">Priority</Label>
            <Input
              id="binding-priority"
              type="number"
              className="max-w-32"
              value={newPriority}
              onChange={(e) => setNewPriority(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Lower renders first. This decides the order between rulesets and outranks the rule
              positions inside them.
            </p>
          </div>

          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={newEnabled}
              onChange={(e) => setNewEnabled(e.target.checked)}
            />
            <span>
              Render immediately
              <span className="block text-xs text-muted-foreground">
                Unticked, the binding is recorded but contributes no rules until you enable it.
              </span>
            </span>
          </label>

          {duplicate ? (
            <p className="text-sm text-amber-500">
              {conflictMessage(describeScope(newHiveId, fleets))}
            </p>
          ) : null}

          <DialogFooter>
            <Button variant="ghost" onClick={() => setCreateOpen(false)}>
              Cancel
            </Button>
            <Button
              disabled={!newRulesetId || duplicate || create.isPending}
              onClick={() =>
                create.mutate({
                  body: {
                    ruleset_id: newRulesetId,
                    enabled: newEnabled,
                    priority: Number(newPriority) || 0,
                    // Omitted, not null: an absent hive_id is how the API spells
                    // org-wide.
                    ...(newHiveId ? { hive_id: newHiveId } : {}),
                  },
                })
              }
            >
              {create.isPending ? 'Binding…' : 'Bind'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={editing !== null} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editing?.ruleset_name} on {describeScope(editing?.hive_id, fleets)}
            </DialogTitle>
            <DialogDescription>
              Priority and whether this binding renders are the only things a binding has to change.
              To point it at a different ruleset or fleet, unbind and bind again.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <Label htmlFor="edit-priority">Priority</Label>
            <Input
              id="edit-priority"
              type="number"
              className="max-w-32"
              value={editPriority}
              onChange={(e) => setEditPriority(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Reordering here moves an entire ruleset ahead of or behind another one, whatever the
              positions inside them say.
            </p>
          </div>

          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={editEnabled}
              onChange={(e) => setEditEnabled(e.target.checked)}
            />
            <span>
              Render this binding
              <span className="block text-xs text-muted-foreground">
                Turning it off removes its rules from every host in scope at the next render.
              </span>
            </span>
          </label>

          <DialogFooter>
            <Button variant="ghost" onClick={() => setEditing(null)}>
              Cancel
            </Button>
            <Button
              disabled={update.isPending}
              onClick={() =>
                editing &&
                update.mutate({
                  path: { id: editing.id },
                  body: { enabled: editEnabled, priority: Number(editPriority) || 0 },
                })
              }
            >
              {update.isPending ? 'Saving…' : 'Save'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={unbinding !== null} onOpenChange={(open) => !open && setUnbinding(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              Unbind {unbinding?.ruleset_name} from {describeScope(unbinding?.hive_id, fleets)}
            </DialogTitle>
            <DialogDescription>
              {unbinding && !unbinding.hive_id
                ? 'This is an org-wide binding. Removing it withdraws these rules from every host in the organization, including hosts that belong to no fleet and have nothing else covering them.'
                : `These rules stop being rendered on ${describeScope(unbinding?.hive_id, fleets)} at the next render. The ruleset itself and its rules are kept.`}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setUnbinding(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={remove.isPending}
              onClick={() => unbinding && remove.mutate({ path: { id: unbinding.id } })}
            >
              {remove.isPending ? 'Unbinding…' : 'Unbind'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};
