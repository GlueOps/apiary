import { useMemo } from 'react';

import { IconRefresh, IconWorld } from '@tabler/icons-react';
import { useQuery } from '@tanstack/react-query';

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
import { firewallPreviewOptions } from '@/sdk/@tanstack/react-query.gen';
import type { BindingView, RuleView } from '@/sdk/types.gen';

import { type Fleet, ORG_WIDE } from './bindings-fleets';

const ACTION_TONE: Record<string, string> = {
  accept: 'bg-emerald-500/15 text-emerald-500',
  drop: 'bg-destructive/15 text-destructive',
  reject: 'bg-amber-500/15 text-amber-500',
};

function port(from: number, to: number) {
  if (!from) return '';
  return to && to !== from ? `:${from}-${to}` : `:${from}`;
}

/**
 * The match condition in one line, in the order nft reads it.
 *
 * Empty strings and zeroes are how the API says "unconstrained", and rendering
 * those as blanks would make a rule that matches EVERYTHING look like a rule
 * with missing data. 'any' is stated so the breadth is visible.
 */
function describeMatch(rule: RuleView) {
  const src = `${rule.src_cidr || 'any'}${port(rule.src_port, rule.src_port_end)}`;
  const dst = `${rule.dst_cidr || 'any'}${port(rule.dst_port, rule.dst_port_end)}`;
  const ifaces = [rule.in_iface && `in ${rule.in_iface}`, rule.out_iface && `out ${rule.out_iface}`]
    .filter(Boolean)
    .join(' ');
  return `${rule.protocol || 'any'} ${src} → ${dst}${ifaces ? ` (${ifaces})` : ''}`;
}

type Segment = { rulesetId: string; binding?: BindingView; rules: RuleView[]; firstIndex: number };

/**
 * Cuts the flattened rule list back into the rulesets it was assembled from,
 * and pairs each run with the binding that put it there.
 *
 * The pairing is positional on purpose: the same ruleset may be bound twice to
 * one host (org-wide AND to its fleet — the intended base-plus-datacenter
 * pattern), so it appears twice in the output at two different priorities.
 * Matching by ruleset_id alone would attribute both runs to whichever binding
 * was found first and report the wrong priority for one of them. Walking the
 * applicable bindings in priority order and consuming each as its run appears
 * keeps the two apart.
 */
function segment(rules: RuleView[], applicable: BindingView[]): Segment[] {
  const queue = [...applicable].sort((a, b) => a.priority - b.priority);
  const segments: Segment[] = [];

  rules.forEach((rule, index) => {
    const open = segments[segments.length - 1];
    // Same ruleset AND still inside the same run. The second condition is what
    // makes the positional pairing above actually work: a ruleset bound twice
    // emits its rules TWICE, as two adjacent identical runs, so matching on
    // ruleset_id alone swallows the second run into the first — the second
    // binding is then never consumed from the queue and its priority is never
    // reported, while the segment shows every rule twice.
    //
    // A run restarts exactly when its first rule reappears, which is a fact
    // about the data rather than an assumption about ordering.
    if (open && open.rulesetId === rule.ruleset_id && open.rules[0]?.id !== rule.id) {
      open.rules.push(rule);
      return;
    }
    const at = queue.findIndex((b) => b.ruleset_id === rule.ruleset_id);
    segments.push({
      rulesetId: rule.ruleset_id,
      binding: at === -1 ? undefined : queue.splice(at, 1)[0],
      rules: [rule],
      firstIndex: index,
    });
  });

  return segments;
}

export function FirewallPreview({
  scope,
  onScopeChange,
  fleets,
  bindings,
  rulesetNames,
}: {
  scope: string;
  onScopeChange: (scope: string) => void;
  fleets: Fleet[];
  bindings: BindingView[];
  rulesetNames: Map<string, string>;
}) {
  const hiveId = scope === ORG_WIDE ? undefined : scope;

  const { data, isLoading, isError, error, isFetching, refetch } = useQuery(
    firewallPreviewOptions(hiveId ? { query: { hive_id: hiveId } } : {}),
  );

  const rules = useMemo(() => data?.rules ?? [], [data]);

  // A host receives its fleet's bindings AND every org-wide one — that union is
  // the base-plus-datacenter pattern, and dropping the org-wide half here would
  // leave the rules it contributed unattributed. Disabled bindings are excluded
  // because they render nothing, so offering one as the explanation for a run
  // of rules would point the reader at the wrong binding.
  const applicable = useMemo(
    () => bindings.filter((b) => b.enabled && (!b.hive_id || b.hive_id === hiveId)),
    [bindings, hiveId],
  );

  const segments = useMemo(() => segment(rules, applicable), [rules, applicable]);

  const previewing = fleets.find((f) => f.id === hiveId);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          Rendered ruleset
          {isFetching ? <span className="text-xs text-muted-foreground">refreshing…</span> : null}
        </CardTitle>
        <CardDescription>
          {hiveId
            ? `Exactly what a host in ${previewing?.label ?? 'this fleet'} receives, in evaluation order.`
            : 'Exactly what a host that belongs to NO fleet receives. Every migration passes through this state, and only org-wide bindings reach it.'}
        </CardDescription>
        <CardAction>
          <div className="flex items-center gap-2">
            <Select value={scope} onValueChange={(next) => onScopeChange(String(next))}>
              <SelectTrigger className="w-56" aria-label="Preview scope">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ORG_WIDE}>No fleet (org-wide only)</SelectItem>
                {fleets.map((f) => (
                  <SelectItem key={f.id} value={f.id}>
                    {f.label}
                    {f.hostnames.length
                      ? ` · ${f.hostnames.length} host${f.hostnames.length === 1 ? '' : 's'}`
                      : ' · no agents'}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              size="icon-sm"
              variant="ghost"
              aria-label="Refresh preview"
              onClick={() => void refetch()}
            >
              <IconRefresh />
            </Button>
          </div>
        </CardAction>
      </CardHeader>

      <CardContent className="space-y-4">
        {previewing && previewing.hostnames.length ? (
          <p className="text-xs text-muted-foreground">
            Hosts here: {previewing.hostnames.join(', ')}
          </p>
        ) : null}

        {isLoading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : isError ? (
          // An empty table on a failed request reads as "this host receives
          // nothing", which is a claim about the firewall rather than about the
          // request. Say which one it is.
          <div className="rounded-lg bg-destructive/10 p-4 text-sm">
            <p className="font-medium">The preview could not be rendered.</p>
            <p className="mt-1 text-muted-foreground">
              {(error as { detail?: string })?.detail ??
                'The server did not answer. Nothing below reflects what these hosts are running.'}
            </p>
          </div>
        ) : rules.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {hiveId
              ? 'Nothing renders here. This fleet has no enabled bindings, and no org-wide binding covers it either — the agent applies no rules at all.'
              : 'Nothing renders for a host outside every fleet. Bind a ruleset org-wide if hosts should stay covered while they move between fleets.'}
          </p>
        ) : (
          <>
            <p className="text-xs text-muted-foreground">
              {data?.count ?? rules.length} rule{(data?.count ?? rules.length) === 1 ? '' : 's'},
              top to bottom. Binding priority decides which ruleset goes first and outranks the
              positions inside it — a rule at position 1 is still evaluated after every rule of a
              lower-priority binding.
            </p>

            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-12">#</TableHead>
                    <TableHead>Rule</TableHead>
                    <TableHead>Action</TableHead>
                    <TableHead>Match</TableHead>
                    <TableHead>Family</TableHead>
                    <TableHead className="text-right">Position</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {segments.map((seg) => [
                    <TableRow
                      key={`${seg.rulesetId}-${seg.firstIndex}-head`}
                      className="bg-muted/50"
                    >
                      <TableCell colSpan={6} className="py-2">
                        <div className="flex flex-wrap items-center gap-2 text-xs">
                          <span className="font-medium">
                            {rulesetNames.get(seg.rulesetId) ??
                              `Ruleset ${seg.rulesetId.slice(0, 8)}`}
                          </span>
                          {seg.binding ? (
                            <>
                              <Badge className="bg-muted text-muted-foreground">
                                priority {seg.binding.priority}
                              </Badge>
                              {seg.binding.hive_id ? (
                                <span className="text-muted-foreground">bound to this fleet</span>
                              ) : (
                                <span className="flex items-center gap-1 text-muted-foreground">
                                  <IconWorld className="size-3" /> bound org-wide
                                </span>
                              )}
                            </>
                          ) : (
                            // Reachable when the bindings list and the preview
                            // were fetched either side of someone else's change.
                            <span className="text-amber-500">
                              no matching binding in the list — reload to resync
                            </span>
                          )}
                          <span className="text-muted-foreground">
                            · {seg.rules.length} rule{seg.rules.length === 1 ? '' : 's'}
                          </span>
                        </div>
                      </TableCell>
                    </TableRow>,
                    ...seg.rules.map((rule, offset) => (
                      <TableRow key={`${seg.firstIndex}-${rule.id}-${offset}`}>
                        <TableCell className="text-muted-foreground tabular-nums">
                          {seg.firstIndex + offset + 1}
                        </TableCell>
                        <TableCell className="font-medium">
                          {rule.name}
                          {rule.comment ? (
                            <span className="block text-xs text-muted-foreground">
                              {rule.comment}
                            </span>
                          ) : null}
                        </TableCell>
                        <TableCell>
                          <Badge
                            className={ACTION_TONE[rule.action] ?? 'bg-muted text-muted-foreground'}
                          >
                            {rule.action || 'accept'}
                          </Badge>
                        </TableCell>
                        <TableCell className="font-mono text-xs">{describeMatch(rule)}</TableCell>
                        <TableCell className="text-muted-foreground">
                          {rule.family || 'inet'}
                        </TableCell>
                        <TableCell className="text-right text-muted-foreground tabular-nums">
                          {rule.position}
                        </TableCell>
                      </TableRow>
                    )),
                  ])}
                </TableBody>
              </Table>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
