import { useMemo } from 'react';

import { IconAlertTriangle } from '@tabler/icons-react';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { BindingView, RulesetView } from '@/sdk/types.gen';

import type { Fleet } from './bindings-fleets';

type Reach = {
  ruleset: RulesetView;
  fleets: Fleet[];
  orgWide: boolean;
  disabledBindings: number;
};

/**
 * What editing a ruleset actually touches.
 *
 * A rule is edited on one screen, but it re-renders on every host the ruleset
 * is bound to, and nothing on the rule itself says how many that is. An
 * org-wide binding is the sharp case: it reaches every fleet plus the hosts in
 * none, so a one-line edit there is an organization-wide change.
 */
export function BlastRadius({
  rulesets,
  bindings,
  fleets,
}: {
  rulesets: RulesetView[];
  bindings: BindingView[];
  fleets: Fleet[];
}) {
  const reach = useMemo<Reach[]>(() => {
    return rulesets
      .map((ruleset) => {
        const mine = bindings.filter((b) => b.ruleset_id === ruleset.id);
        const live = mine.filter((b) => b.enabled);
        const orgWide = live.some((b) => !b.hive_id);
        const direct = new Set(live.map((b) => b.hive_id).filter((id): id is string => !!id));

        return {
          ruleset,
          // An org-wide binding lands on every fleet, so a fleet-specific
          // binding on top of it adds nothing to the count — the union, not the
          // sum, is the blast radius.
          fleets: orgWide ? fleets : fleets.filter((f) => direct.has(f.id)),
          orgWide,
          disabledBindings: mine.length - live.length,
        };
      })
      .sort((a, b) => {
        const byOrgWide = Number(b.orgWide) - Number(a.orgWide);
        return byOrgWide !== 0 ? byOrgWide : b.fleets.length - a.fleets.length;
      });
  }, [rulesets, bindings, fleets]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Blast radius</CardTitle>
        <CardDescription>
          Where each ruleset lands. Editing one rule re-renders every fleet listed on its row, which
          is not visible from the rule itself.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {rulesets.length === 0 ? (
          // A header row with nothing under it and a footnote about counting
          // tells a new tenant nothing. Say what the card is for and what the
          // first move is, because this is the state every organization starts in.
          <p className="text-sm text-muted-foreground">
            No rulesets yet, so nothing reaches any fleet. Create a ruleset, add rules to it, then
            bind it — org-wide to cover every host including ones between fleets, or to a single
            fleet. Once bound, this card shows how far an edit to each ruleset travels.
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Ruleset</TableHead>
                <TableHead>Reaches</TableHead>
                <TableHead>Fleets</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {reach.map((r) => (
                <TableRow key={r.ruleset.id}>
                  <TableCell className="font-medium">
                    {r.ruleset.name}
                    {r.disabledBindings > 0 ? (
                      <span className="block text-xs text-muted-foreground">
                        {r.disabledBindings} binding{r.disabledBindings === 1 ? '' : 's'} disabled,
                        not counted
                      </span>
                    ) : null}
                  </TableCell>
                  <TableCell>
                    {r.orgWide ? (
                      <div className="flex flex-col gap-1">
                        <Badge className="w-fit bg-amber-500/15 text-amber-500">
                          <IconAlertTriangle className="size-3" /> every fleet
                        </Badge>
                        <span className="text-xs text-muted-foreground">
                          and hosts in no fleet at all
                        </span>
                      </div>
                    ) : r.fleets.length === 0 ? (
                      <span className="text-sm text-muted-foreground">nowhere — not bound</span>
                    ) : (
                      <span className="text-sm tabular-nums">
                        {r.fleets.length} fleet{r.fleets.length === 1 ? '' : 's'}
                      </span>
                    )}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {r.fleets.length === 0 ? '—' : r.fleets.map((f) => f.label).join(', ')}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        <p className="text-xs text-muted-foreground">
          Fleets are counted from the agents that have enrolled and from existing bindings, because
          the API has no endpoint that lists fleets. A fleet with neither makes this a floor rather
          than a total.
        </p>
      </CardContent>
    </Card>
  );
}
