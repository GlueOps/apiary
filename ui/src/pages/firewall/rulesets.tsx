import { useState } from 'react';

import {
  IconAlertTriangle,
  IconArrowDown,
  IconArrowNarrowDown,
  IconArrowUp,
  IconArrowsSort,
  IconPencil,
  IconPlus,
  IconTrash,
} from '@tabler/icons-react';
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  firewallBindingsListOptions,
  firewallRulesCreateMutation,
  firewallRulesDeleteMutation,
  firewallRulesListOptions,
  firewallRulesListQueryKey,
  firewallRulesUpdateMutation,
  firewallRulesetsCreateMutation,
  firewallRulesetsDeleteMutation,
  firewallRulesetsListOptions,
  firewallRulesetsListQueryKey,
  firewallRulesetsUpdateMutation,
} from '@/sdk/@tanstack/react-query.gen';
import type { RuleBodyWritable, RuleView, RulesetView } from '@/sdk/types.gen';

import { RuleFormDialog, carriesPorts, toRuleBody } from './rule-form';

/** New rules land 10 past the last one, leaving room to slot one in between. */
const POSITION_STEP = 10;

const ACTION_TONE: Record<string, string> = {
  accept: 'bg-emerald-500/15 text-emerald-500',
  drop: 'bg-destructive/15 text-destructive',
  reject: 'bg-amber-500/15 text-amber-500',
};

const detail = (e: unknown, fallback: string) => (e as { detail?: string })?.detail ?? fallback;

/** ":22", ":8000-8100", or nothing at all when the rule matches any port. */
function portLabel(start: number, end: number, protocol: string) {
  if (!carriesPorts(protocol) || !start) return '';
  return !end || end === start ? `:${start}` : `:${start}-${end}`;
}

export const FirewallRulesets = () => {
  const queryClient = useQueryClient();

  const [chosenId, setChosenId] = useState<string | null>(null);
  const [rulesetDialog, setRulesetDialog] = useState<'new' | RulesetView | null>(null);
  const [deletingRuleset, setDeletingRuleset] = useState<RulesetView | null>(null);
  // Three states, not two. A clone is prefilled from an existing rule but
  // CREATES a new one, so it cannot be represented by "the rule being edited"
  // without the submit path guessing which the operator meant.
  const [ruleDialog, setRuleDialog] = useState<
    { mode: 'new' } | { mode: 'edit' | 'clone'; rule: RuleView } | null
  >(null);
  const [deletingRule, setDeletingRule] = useState<RuleView | null>(null);

  const rulesetsQuery = useQuery(firewallRulesetsListOptions());
  const rulesets = rulesetsQuery.data?.rulesets ?? [];
  // Derived rather than stored, so a ruleset deleted underneath the selection
  // falls back to the first one instead of leaving the rules card pointed at an
  // id the server no longer has.
  const selected = rulesets.find((r) => r.id === chosenId) ?? rulesets[0] ?? null;
  const selectedId = selected?.id ?? null;

  const rulesQuery = useQuery({
    ...firewallRulesListOptions({ path: { id: selectedId ?? '' } }),
    enabled: !!selectedId,
  });

  // Position is the policy, so the list is sorted by it rather than trusting the
  // order rows happen to come back in.
  const rules = [...(rulesQuery.data?.rules ?? [])].sort((a, b) => a.position - b.position);
  const nextPosition = rules.length ? rules[rules.length - 1].position + POSITION_STEP : 10;

  // Two rules a position apart (or on the same one) cannot be reordered by
  // swapping numbers, and equal positions mean the database decides the order.
  const spacingIsTight = rules.some((r, i) => i > 0 && r.position - rules[i - 1].position < 2);

  const refreshRulesets = () =>
    queryClient.invalidateQueries({ queryKey: firewallRulesetsListQueryKey() });
  const refreshRules = () => {
    if (!selectedId) return;
    void queryClient.invalidateQueries({
      queryKey: firewallRulesListQueryKey({ path: { id: selectedId } }),
    });
  };

  const createRuleset = useMutation({
    ...firewallRulesetsCreateMutation(),
    onError: (e) => toast.error(detail(e, 'Could not create that ruleset')),
  });
  const updateRuleset = useMutation({
    ...firewallRulesetsUpdateMutation(),
    onError: (e) => toast.error(detail(e, 'Could not rename that ruleset')),
  });
  const deleteRuleset = useMutation({
    ...firewallRulesetsDeleteMutation(),
    onError: (e) => toast.error(detail(e, 'Could not delete that ruleset')),
  });
  const createRule = useMutation({
    ...firewallRulesCreateMutation(),
    onError: (e) => toast.error(detail(e, 'Could not add that rule')),
  });
  // Shared by the form, the enable/disable toggle and reordering, because all
  // three are the same whole-rule PUT.
  const saveRule = useMutation({
    ...firewallRulesUpdateMutation(),
    onError: (e) => toast.error(detail(e, 'Could not save that rule')),
  });
  const deleteRule = useMutation({
    ...firewallRulesDeleteMutation(),
    onError: (e) => toast.error(detail(e, 'Could not delete that rule')),
  });

  // What a ruleset deletion takes with it. Fetched only while the confirmation
  // is open, so the dialog can name the damage instead of describing it vaguely.
  const cascadeRules = useQuery({
    ...firewallRulesListOptions({ path: { id: deletingRuleset?.id ?? '' } }),
    enabled: !!deletingRuleset,
  });
  const cascadeBindings = useQuery({
    ...firewallBindingsListOptions(),
    enabled: !!deletingRuleset,
  });
  const cascadeRuleCount = cascadeRules.data ? (cascadeRules.data.rules ?? []).length : null;
  const cascadeBindingCount = cascadeBindings.data
    ? (cascadeBindings.data.bindings ?? []).filter((b) => b.ruleset_id === deletingRuleset?.id)
        .length
    : null;

  const submitRule = (body: RuleBodyWritable) => {
    if (!selectedId || !ruleDialog) return;
    const done = (message: string) => () => {
      toast.success(message);
      setRuleDialog(null);
      refreshRules();
    };
    if (ruleDialog.mode === 'edit') {
      saveRule.mutate(
        { path: { id: ruleDialog.rule.id }, body },
        { onSuccess: done('Rule replaced') },
      );
    } else {
      // Both 'new' and 'clone' create. A clone that PUT to its source would
      // silently replace the rule it was copied from.
      createRule.mutate({ path: { id: selectedId }, body }, { onSuccess: done('Rule added') });
    }
  };

  const toggleRule = (rule: RuleView) =>
    saveRule.mutate(
      { path: { id: rule.id }, body: { ...toRuleBody(rule), enabled: !rule.enabled } },
      { onSuccess: refreshRules },
    );

  /** Swap one rule with its neighbour by exchanging their positions. */
  const move = (index: number, delta: number) => {
    const from = rules[index];
    const to = rules[index + delta];
    if (!from || !to || from.position === to.position) return;
    void (async () => {
      try {
        await saveRule.mutateAsync({
          path: { id: from.id },
          body: { ...toRuleBody(from), position: to.position },
        });
        await saveRule.mutateAsync({
          path: { id: to.id },
          body: { ...toRuleBody(to), position: from.position },
        });
      } catch {
        // onError has already reported it; the refresh below shows which of the
        // two writes landed rather than leaving a reordered list that is a lie.
      } finally {
        refreshRules();
      }
    })();
  };

  const respace = () => {
    void (async () => {
      try {
        // Sequential on purpose: these are writes to rows that are ordered
        // against each other, and the server may hold a uniqueness constraint
        // over (ruleset, position).
        for (let i = 0; i < rules.length; i += 1) {
          const position = (i + 1) * POSITION_STEP;
          if (rules[i].position === position) continue;
          await saveRule.mutateAsync({
            path: { id: rules[i].id },
            body: { ...toRuleBody(rules[i]), position },
          });
        }
        toast.success('Positions respaced to 10, 20, 30…');
      } catch {
        // Reported by onError; the refresh shows how far it got.
      } finally {
        refreshRules();
      }
    })();
  };

  return (
    <div className="w-full space-y-6 p-6">
      <div>
        <h1 className="font-heading text-2xl">Firewall rulesets</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          A ruleset is a named group of rules — “management access”, “ceph replication”. Fleets are
          bound to rulesets rather than owning rules, so a rule you want everywhere is written once
          instead of copied per fleet. A ruleset does nothing until something is bound to it.
        </p>
      </div>

      {rulesetsQuery.isError ? (
        <Card>
          <CardHeader>
            <CardTitle>Rulesets could not be loaded</CardTitle>
            <CardDescription>
              {detail(rulesetsQuery.error, 'The list did not load.')} Nothing is shown rather than
              an empty list, because an empty firewall page and a firewall with no rules look the
              same and mean very different things.
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Rulesets</CardTitle>
            <CardDescription>
              {rulesets.length === 0
                ? 'None yet.'
                : `${rulesets.length} ruleset${rulesets.length === 1 ? '' : 's'}.`}
            </CardDescription>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" onClick={() => setRulesetDialog('new')}>
                <IconPlus /> New ruleset
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {rulesetsQuery.isLoading ? (
              <p className="text-sm text-muted-foreground">Loading…</p>
            ) : rulesets.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                Nothing here yet. Start with one ruleset per intent — “management access” for the
                ports your operators need, “ceph replication” for what the storage network needs —
                then bind it to the fleets that should enforce it.
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rulesets.map((r) => (
                    <TableRow
                      key={r.id}
                      className={r.id === selectedId ? 'bg-muted/50' : undefined}
                    >
                      <TableCell className="font-medium">{r.name}</TableCell>
                      <TableCell className="text-muted-foreground">
                        {r.description || '—'}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2">
                          <Button
                            size="xs"
                            variant={r.id === selectedId ? 'secondary' : 'outline'}
                            onClick={() => setChosenId(r.id)}
                          >
                            {r.id === selectedId ? 'Editing rules' : 'Edit rules'}
                          </Button>
                          <Button
                            size="icon-xs"
                            variant="ghost"
                            aria-label={`Rename ${r.name}`}
                            onClick={() => setRulesetDialog(r)}
                          >
                            <IconPencil className="size-4" />
                          </Button>
                          <Button
                            size="icon-xs"
                            variant="ghost"
                            aria-label={`Delete ${r.name}`}
                            onClick={() => setDeletingRuleset(r)}
                          >
                            <IconTrash className="size-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      )}

      {selected ? (
        <Card>
          <CardHeader>
            <CardTitle>Rules in {selected.name}</CardTitle>
            <CardDescription>
              {rules.length === 0
                ? 'No rules yet.'
                : `${rules.length} rule${rules.length === 1 ? '' : 's'}, evaluated lowest position first.`}
            </CardDescription>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" onClick={() => setRuleDialog({ mode: 'new' })}>
                <IconPlus /> Add rule
              </Button>
              {spacingIsTight ? (
                <Button size="sm" variant="outline" disabled={saveRule.isPending} onClick={respace}>
                  <IconArrowsSort /> Respace positions
                </Button>
              ) : null}
            </div>
          </CardHeader>
          <CardContent>
            {rulesQuery.isError ? (
              <p className="text-sm text-destructive">
                {detail(rulesQuery.error, 'These rules could not be loaded.')}
              </p>
            ) : rulesQuery.isLoading ? (
              <p className="text-sm text-muted-foreground">Loading…</p>
            ) : rules.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No rules in this ruleset, so binding it to a fleet would change nothing. Add the
                first rule — the order you add them in is the policy, since the first match decides.
              </p>
            ) : (
              <div className="space-y-3">
                <div className="flex items-start gap-2 rounded-md border bg-muted/40 p-3 text-xs text-muted-foreground">
                  <IconArrowNarrowDown className="mt-px size-4 shrink-0" />
                  <p>
                    Read top to bottom: that is the order the host evaluates them in, and the first
                    rule that matches decides. An accept above a drop is a different policy from the
                    same two reversed.
                  </p>
                </div>

                {spacingIsTight ? (
                  <div className="flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-xs text-amber-600 dark:text-amber-400">
                    <IconAlertTriangle className="mt-px size-4 shrink-0" />
                    <p>
                      Two rules sit on the same position or one apart. Tied positions still
                      render in a fixed order — oldest first — so the ruleset is not
                      unpredictable, but the order is now creation history rather than
                      anything you chose, and inserting between them needs a respace.
                      Space them 10, 20, 30 to put the ordering back in your hands.
                    </p>
                  </div>
                ) : null}

                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-16">Pos</TableHead>
                      <TableHead>Rule</TableHead>
                      <TableHead>Match</TableHead>
                      <TableHead>Action</TableHead>
                      <TableHead />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rules.map((rule, index) => (
                      <TableRow key={rule.id} className={rule.enabled ? undefined : 'opacity-60'}>
                        <TableCell className="text-muted-foreground tabular-nums">
                          {rule.position}
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-col gap-1">
                            <span className="font-medium">{rule.name}</span>
                            {rule.comment ? (
                              <span className="text-xs text-muted-foreground">{rule.comment}</span>
                            ) : null}
                            {rule.enabled ? null : (
                              <span className="text-xs text-muted-foreground">
                                Disabled — holds its place in the order but is not rendered onto any
                                host.
                              </span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-col gap-0.5">
                            <span className="font-mono text-xs">
                              {rule.protocol} {rule.src_cidr || 'any'}
                              {portLabel(rule.src_port, rule.src_port_end, rule.protocol)} →{' '}
                              {rule.dst_cidr || 'any'}
                              {portLabel(rule.dst_port, rule.dst_port_end, rule.protocol)}
                            </span>
                            <span className="text-xs text-muted-foreground">
                              {rule.family} · {rule.table_name}/{rule.chain_name}
                              {rule.in_iface ? ` · in ${rule.in_iface}` : ''}
                              {rule.out_iface ? ` · out ${rule.out_iface}` : ''}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge className={ACTION_TONE[rule.action] ?? ''}>{rule.action}</Badge>
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-2">
                            <Button
                              size="icon-xs"
                              variant="ghost"
                              aria-label={`Move ${rule.name} earlier`}
                              disabled={index === 0 || saveRule.isPending}
                              onClick={() => move(index, -1)}
                            >
                              <IconArrowUp className="size-4" />
                            </Button>
                            <Button
                              size="icon-xs"
                              variant="ghost"
                              aria-label={`Move ${rule.name} later`}
                              disabled={index === rules.length - 1 || saveRule.isPending}
                              onClick={() => move(index, 1)}
                            >
                              <IconArrowDown className="size-4" />
                            </Button>
                            <Button
                              size="xs"
                              variant="outline"
                              disabled={saveRule.isPending}
                              onClick={() => {
                                // Confirmed only in the direction that OPENS
                                // something. Disabling a drop or a reject removes
                                // a block, on every host in every fleet bound to
                                // this ruleset, at their next poll — from one
                                // unconfirmed click on a row among many.
                                // Disabling an accept, or enabling anything, is
                                // either closing or additive, and a prompt there
                                // is the noise that teaches people to click
                                // through the prompt that matters.
                                if (rule.enabled && rule.action !== 'accept') {
                                  const ok = window.confirm(
                                    `Disable "${rule.name}"?\n\n` +
                                      `It currently ${rule.action}s matching traffic. Disabling it ` +
                                      `removes that block from every host in every fleet bound to ` +
                                      `this ruleset, at their next poll.`,
                                  );
                                  if (!ok) return;
                                }
                                toggleRule(rule);
                              }}
                            >
                              {rule.enabled ? 'Disable' : 'Enable'}
                            </Button>
                            <Button
                              size="xs"
                              variant="outline"
                              onClick={() => setRuleDialog({ mode: 'edit', rule })}
                            >
                              Edit
                            </Button>
                            {/*
                              Clone keeps the source's POSITION rather than
                              appending to the end, because position is meaning:
                              a rule cloned from above a drop must stay above it,
                              and appending would silently move it below and
                              invert what it does. The tie is benign — rendering
                              orders by (position, created_at, id), so the copy
                              lands immediately after its source, deterministically.
                            */}
                            <Button
                              size="xs"
                              variant="outline"
                              onClick={() => setRuleDialog({ mode: 'clone', rule })}
                            >
                              Clone
                            </Button>
                            <Button
                              size="icon-xs"
                              variant="ghost"
                              aria-label={`Delete ${rule.name}`}
                              onClick={() => setDeletingRule(rule)}
                            >
                              <IconTrash className="size-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>

                <p className="text-xs text-muted-foreground">
                  Anything not matched above falls through to whatever policy the chain already has
                  on the host. This ruleset adds rules; it does not replace the chain.
                </p>
              </div>
            )}
          </CardContent>
        </Card>
      ) : null}

      {rulesetDialog ? (
        <RulesetDialog
          key={rulesetDialog === 'new' ? 'new' : rulesetDialog.id}
          ruleset={rulesetDialog === 'new' ? null : rulesetDialog}
          isPending={createRuleset.isPending || updateRuleset.isPending}
          onClose={() => setRulesetDialog(null)}
          onSubmit={(body) => {
            const done = (message: string) => () => {
              toast.success(message);
              setRulesetDialog(null);
              void refreshRulesets();
            };
            if (rulesetDialog === 'new') {
              createRuleset.mutate(
                { body },
                {
                  onSuccess: (created) => {
                    // Select it immediately: a ruleset with no rules is the one
                    // thing an operator has just decided to fix.
                    setChosenId(created.id);
                    done('Ruleset created')();
                  },
                },
              );
            } else {
              updateRuleset.mutate(
                { path: { id: rulesetDialog.id }, body },
                { onSuccess: done('Ruleset updated') },
              );
            }
          }}
        />
      ) : null}

      {ruleDialog && selected ? (
        <RuleFormDialog
          key={ruleDialog.mode === 'new' ? 'new' : `${ruleDialog.mode}:${ruleDialog.rule.id}`}
          mode={ruleDialog.mode}
          rule={ruleDialog.mode === 'new' ? null : ruleDialog.rule}
          rulesetName={selected.name}
          nextPosition={nextPosition}
          isPending={createRule.isPending || saveRule.isPending}
          onSubmit={submitRule}
          onClose={() => setRuleDialog(null)}
        />
      ) : null}

      <Dialog
        open={deletingRuleset !== null}
        onOpenChange={(open) => !open && setDeletingRuleset(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete {deletingRuleset?.name}?</DialogTitle>
            <DialogDescription>
              This also deletes{' '}
              {cascadeRuleCount === null
                ? 'every rule in it'
                : `its ${cascadeRuleCount} rule${cascadeRuleCount === 1 ? '' : 's'}`}{' '}
              and{' '}
              {cascadeBindingCount === null
                ? 'every binding pointing at it'
                : `${cascadeBindingCount} binding${cascadeBindingCount === 1 ? '' : 's'} pointing at it`}
              . Hosts in the bound fleets stop enforcing these rules at their next poll, which will
              open whatever the rules were closing. It cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeletingRuleset(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={deleteRuleset.isPending}
              onClick={() => {
                if (!deletingRuleset) return;
                deleteRuleset.mutate(
                  { path: { id: deletingRuleset.id } },
                  {
                    onSuccess: () => {
                      toast.success(`Deleted ${deletingRuleset.name}`);
                      setChosenId(null);
                      setDeletingRuleset(null);
                      void refreshRulesets();
                    },
                  },
                );
              }}
            >
              {deleteRuleset.isPending ? 'Deleting…' : 'Delete ruleset and its rules'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deletingRule !== null} onOpenChange={(open) => !open && setDeletingRule(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete {deletingRule?.name}?</DialogTitle>
            <DialogDescription>
              Removes the {deletingRule?.action} at position {deletingRule?.position} from{' '}
              {selected?.name}. Every rule below it keeps its position, so the order does not shift
              — but whatever this rule was deciding is now decided by the next rule that matches. To
              keep it for later without enforcing it, disable it instead.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeletingRule(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={deleteRule.isPending}
              onClick={() => {
                if (!deletingRule) return;
                deleteRule.mutate(
                  { path: { id: deletingRule.id } },
                  {
                    onSuccess: () => {
                      toast.success(`Deleted ${deletingRule.name}`);
                      setDeletingRule(null);
                      refreshRules();
                    },
                  },
                );
              }}
            >
              {deleteRule.isPending ? 'Deleting…' : 'Delete rule'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

function RulesetDialog({
  ruleset,
  isPending,
  onSubmit,
  onClose,
}: {
  ruleset: RulesetView | null;
  isPending: boolean;
  onSubmit: (body: { name: string; description?: string }) => void;
  onClose: () => void;
}) {
  const [name, setName] = useState(ruleset?.name ?? '');
  const [description, setDescription] = useState(ruleset?.description ?? '');

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{ruleset ? `Rename ${ruleset.name}` : 'New ruleset'}</DialogTitle>
          <DialogDescription>
            Name it after the intent, not the ports. Whoever reads a binding sees only this name and
            has to know from it what they are switching on for a whole fleet.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="ruleset-name">Name</Label>
          <Input
            id="ruleset-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="management access"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="ruleset-description">Description</Label>
          <Input
            id="ruleset-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="ssh and the Proxmox UI, from the operator range only"
          />
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            disabled={name.trim() === '' || isPending}
            onClick={() => onSubmit({ name: name.trim(), description: description.trim() })}
          >
            {isPending ? 'Saving…' : ruleset ? 'Save' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
