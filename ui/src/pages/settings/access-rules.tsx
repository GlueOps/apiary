import { useState } from 'react';

import { IconAlertTriangle, IconInfoCircle, IconPlus, IconSearch } from '@tabler/icons-react';
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
  platformAccessRulesDeleteMutation,
  platformAccessRulesListOptions,
  platformAccessRulesListQueryKey,
  platformAccessRulesUpdateMutation,
} from '@/sdk/@tanstack/react-query.gen';
import type { AccessRuleView } from '@/sdk/types.gen';

import { CreateAccessRuleDialog } from './access-rules-create-dialog';
import {
  PURPOSES,
  type Policy,
  PolicyChoice,
  errorDetail,
  purposeMeta,
} from './access-rules-shared';

type Origin = 'all' | 'seeded' | 'operator' | 'overridden';

const ORIGINS: { value: Origin; label: string }[] = [
  { value: 'all', label: 'All rules' },
  { value: 'overridden', label: 'Overridden seeds' },
  { value: 'seeded', label: 'From the seed corpus' },
  { value: 'operator', label: 'Written by an operator' },
];

/**
 * The order the database checks rows in, reproduced exactly: email rules first,
 * then domain rules with the longest value first.
 *
 * This mirrors EvaluateAccessRules' `ORDER BY CASE kind WHEN 'email' THEN 0 ELSE
 * 1 END, LENGTH(value) DESC`. It is reproduced rather than approximated because
 * the number shown against each row is a claim about which rule decides, and an
 * operator who cannot see that will write a rule that never fires. The final
 * comparison by value is display-only — it breaks ties between rows that can
 * never both match the same address, so it cannot disagree with the server.
 */
const inEvaluationOrder = (rules: AccessRuleView[]) =>
  [...rules].sort((a, b) => {
    if (a.kind !== b.kind) return a.kind === 'email' ? -1 : 1;
    if (a.value.length !== b.value.length) return b.value.length - a.value.length;
    return a.value.localeCompare(b.value);
  });

/**
 * What this row does NOT decide, phrased concretely.
 *
 * A domain rule is out-ranked, for addresses at a sub-domain, by a rule written
 * on that sub-domain — the lookup considers only the literal domain and its
 * registrable domain, and takes the longer of the two. An email rule beats every
 * domain rule that covers the same address. Both are invisible in a flat list,
 * and both are how an operator ends up with a rule they believe is in force.
 */
function precedenceNote(rule: AccessRuleView, siblings: AccessRuleView[]): string | null {
  if (rule.kind === 'email') {
    const at = rule.value.lastIndexOf('@');
    const domain = at < 0 ? '' : rule.value.slice(at + 1);
    const covering = siblings.filter(
      (o) => o.kind === 'domain' && (o.value === domain || domain.endsWith(`.${o.value}`)),
    );
    if (covering.length === 0) return null;
    return `Decides this address instead of the domain rule on ${covering
      .map((o) => o.value)
      .join(' and ')}.`;
  }

  const narrower = siblings.filter(
    (o) => o.kind === 'domain' && o.id !== rule.id && o.value.endsWith(`.${rule.value}`),
  );
  if (narrower.length === 0) return null;
  return `Does not decide addresses at ${narrower
    .map((o) => o.value)
    .join(' or ')} — the more specific rule wins there.`;
}

function OriginCell({ rule }: { rule: AccessRuleView }) {
  return (
    <div className="flex flex-col items-start gap-1">
      {rule.seeded ? (
        <Badge variant="secondary">Seed corpus</Badge>
      ) : (
        <Badge variant="outline">Operator</Badge>
      )}
      {rule.seed_overridden ? <Badge variant="destructive">Overridden to allow</Badge> : null}
      {rule.seed_restores_on_restart ? (
        <span className="text-xs text-muted-foreground">Re-created on every restart</span>
      ) : null}
    </div>
  );
}

export const AccessRules = () => {
  const queryClient = useQueryClient();
  const { data, isLoading, isError, error } = useQuery({ ...platformAccessRulesListOptions() });

  const [createOpen, setCreateOpen] = useState(false);
  // The rule being loosened, not a boolean: the confirmation has to name what it
  // permits, and it carries a reason field so the divergence this page exists to
  // surface arrives with an explanation attached.
  const [allowing, setAllowing] = useState<AccessRuleView | null>(null);
  const [allowReason, setAllowReason] = useState('');
  const [removing, setRemoving] = useState<AccessRuleView | null>(null);
  // The server's own warning, kept on the page rather than only toasted. A
  // deletion that the next boot undoes is not something to learn from a banner
  // that fades after four seconds.
  const [undoneOnRestart, setUndoneOnRestart] = useState<string | null>(null);

  const [search, setSearch] = useState('');
  const [origin, setOrigin] = useState<Origin>('all');

  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: platformAccessRulesListQueryKey() });

  const update = useMutation({
    ...platformAccessRulesUpdateMutation(),
    onSuccess: (rule) => {
      toast.success(
        rule.policy === 'allow'
          ? `${rule.value} is now permitted for ${purposeMeta(rule.purpose).label.toLowerCase()}`
          : `${rule.value} is now denied for ${purposeMeta(rule.purpose).label.toLowerCase()}`,
      );
      setAllowing(null);
      setAllowReason('');
      refresh();
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not change that rule')),
  });

  const remove = useMutation({
    ...platformAccessRulesDeleteMutation(),
    onSuccess: (res) => {
      if (res.seed_restores_on_restart) {
        setUndoneOnRestart(
          res.warning ??
            'That rule is in the seed corpus. The next restart re-creates it as a create_org deny, so the deletion is temporary — an update to allow is the durable form.',
        );
        toast.warning('Deleted, but it comes back on the next restart');
      } else {
        toast.success('Rule deleted');
      }
      setRemoving(null);
      refresh();
    },
    onError: (e) => toast.error(errorDetail(e, 'Could not delete that rule')),
  });

  const busy = update.isPending || remove.isPending;

  const setPolicy = (rule: AccessRuleView, next: Policy) => {
    if (next === rule.policy) return;
    if (next === 'allow') {
      // Loosening asks; tightening does not. Permitting something is the change
      // that outlives a restart and the change nobody else can see from the
      // source, so it is the one that gets a confirmation and a reason field.
      setAllowing(rule);
      setAllowReason(rule.reason ?? '');
      return;
    }
    // reason is resent unchanged: the update body replaces it, and omitting it
    // clears the column.
    update.mutate({ path: { id: rule.id }, body: { policy: 'deny', reason: rule.reason } });
  };

  const rules = data?.rules ?? [];
  const seedingEnabled = data?.seeding_enabled ?? false;
  const overridden = rules.filter((r) => r.seed_overridden);

  const matchesFilters = (r: AccessRuleView) => {
    const q = search.trim().toLowerCase();
    if (
      q !== '' &&
      !r.value.toLowerCase().includes(q) &&
      !(r.reason ?? '').toLowerCase().includes(q)
    )
      return false;
    if (origin === 'seeded') return r.seeded;
    if (origin === 'operator') return !r.seeded;
    if (origin === 'overridden') return r.seed_overridden;
    return true;
  };

  const filtering = search.trim() !== '' || origin !== 'all';

  return (
    <div className="w-full space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="max-w-3xl">
          <h1 className="font-heading text-2xl">Access rules</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Who may sign in, found an organization, or accept an invitation. This is a separate
            mechanism from platform and organization policy: those decide what an account must do to
            authenticate, these decide whether the address is admitted at all. Evaluation is
            entirely table-driven — every rule that is enforced is a row below, and nothing outside
            this table adds to it.
          </p>
        </div>
        <Button size="sm" onClick={() => setCreateOpen(true)}>
          <IconPlus /> Add a rule
        </Button>
      </div>

      {isError ? (
        // An unreachable server and an empty table look identical otherwise, and
        // one of them means an operator concludes nothing is gating registration
        // when in fact dozens of rows are.
        <Card>
          <CardHeader>
            <CardTitle>Rules could not be loaded</CardTitle>
            <CardDescription>
              {errorDetail(
                error,
                'The server did not answer. Nothing below reflects what is actually being enforced.',
              )}
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <>
          {undoneOnRestart ? (
            <Card className="border-amber-500/40 bg-amber-500/5">
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-amber-600 dark:text-amber-500">
                  <IconAlertTriangle className="size-4" /> That deletion does not last
                </CardTitle>
                <CardDescription>{undoneOnRestart}</CardDescription>
              </CardHeader>
              <CardContent>
                <Button size="sm" variant="outline" onClick={() => setUndoneOnRestart(null)}>
                  Dismiss
                </Button>
              </CardContent>
            </Card>
          ) : null}

          {overridden.length > 0 ? (
            // The headline. A seeded deny turned into an allow exists only in
            // this table: the corpus in the source still lists the domain, the
            // seed pass never updates an existing row, and no other screen
            // mentions it. Anyone reading the source would conclude the opposite
            // of what is enforced, so it goes above everything else.
            <Card className="border-destructive/40 bg-destructive/5">
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-destructive">
                  <IconAlertTriangle className="size-4" />
                  {overridden.length} seeded {overridden.length === 1 ? 'domain is' : 'domains are'}{' '}
                  permitted despite the corpus
                </CardTitle>
                <CardDescription>
                  These domains are in the public-mailbox corpus that ships denied, and an operator
                  has changed them to allow. The change is permanent — the seed pass only ever
                  inserts, so it will never restore the deny. Reading the corpus in the source would
                  tell you these are blocked. They are not.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Domain</TableHead>
                        <TableHead>Permitted for</TableHead>
                        <TableHead>Reason given</TableHead>
                        <TableHead>Changed</TableHead>
                        <TableHead className="sr-only">Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {overridden.map((r) => (
                        <TableRow key={r.id}>
                          <TableCell className="font-mono text-xs break-all">{r.value}</TableCell>
                          <TableCell>{purposeMeta(r.purpose).label}</TableCell>
                          <TableCell className="text-muted-foreground">
                            {r.reason ?? 'No reason recorded'}
                          </TableCell>
                          <TableCell className="text-muted-foreground">
                            {new Date(r.updated_at).toLocaleString()}
                          </TableCell>
                          <TableCell>
                            <div className="flex justify-end">
                              <Button
                                size="xs"
                                variant="outline"
                                disabled={busy}
                                onClick={() => setPolicy(r, 'deny')}
                              >
                                Restore the deny
                              </Button>
                            </div>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </CardContent>
            </Card>
          ) : null}

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <IconInfoCircle className="size-4" /> How a rule is chosen
              </CardTitle>
              <CardDescription>
                Precedence is not visible in a flat list, and a rule that never fires looks exactly
                like one that does.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-muted-foreground">
              <p>
                <span className="font-medium text-foreground">Purpose first.</span> Only rules
                stored under the question being asked are considered. A rule that denies
                organization creation has no bearing on signing in.
              </p>
              <p>
                <span className="font-medium text-foreground">Then email beats domain.</span> An
                allow on one address wins over a deny covering its whole domain, and the reverse.
              </p>
              <p>
                <span className="font-medium text-foreground">
                  Then the most specific domain wins.
                </span>{' '}
                A domain rule matches an address whose domain is exactly its value, or whose
                registrable domain is its value — so a rule on{' '}
                <code className="rounded bg-muted px-1 py-0.5 text-xs">example.com</code> also
                covers{' '}
                <code className="rounded bg-muted px-1 py-0.5 text-xs">mail.example.com</code>{' '}
                unless a rule written on{' '}
                <code className="rounded bg-muted px-1 py-0.5 text-xs">mail.example.com</code> takes
                over for it.
              </p>
              <p>
                <span className="font-medium text-foreground">The first match decides</span>, and
                nothing below it is consulted. Only the exact value{' '}
                <code className="rounded bg-muted px-1 py-0.5 text-xs">allow</code> permits;
                anything else denies. If no rule matches at all, the request is allowed — a purpose
                with no rules gates nothing.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Where the seeded rules come from</CardTitle>
              <CardDescription>
                {seedingEnabled
                  ? 'At every boot the platform inserts its public-mailbox corpus as create_org denies, so a throwaway mailbox cannot mint a tenant. The insert only ever adds: a row that already exists is left exactly as it is.'
                  : 'Seeding is switched off (auth.seed_public_domain_rules). The rows marked as coming from the corpus were written by an earlier boot and nothing restores them now — deleting one is permanent, and no new public-mailbox domain will appear here.'}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-muted-foreground">
              <p>
                <span className="font-medium text-foreground">Changing a seeded rule lasts.</span>{' '}
                An update to allow is never overwritten, because the seed pass does nothing to a row
                that is already there.
              </p>
              <p>
                <span className="font-medium text-foreground">Deleting one does not</span>
                {seedingEnabled
                  ? ' — the next restart puts it back as a deny. That is why seeded rows here are switched between allow and deny rather than removed: the reversible-looking action is the one that silently reverts.'
                  : ', normally. With seeding off nothing re-creates it, so a deletion here stays deleted; turning seeding back on would restore every corpus domain that is missing.'}
              </p>
              <p>
                The corpus in the source is seed data, not a live check. Adding a domain to it
                changes nothing for a domain that already has a row, and removing one never deletes
                anything. What this table says is what is enforced.
              </p>
            </CardContent>
          </Card>

          <div className="flex flex-wrap items-center gap-3">
            <div className="relative">
              <IconSearch className="absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Filter by domain, address or reason"
                className="w-72 pl-8"
                aria-label="Filter rules"
              />
            </div>
            <div className="flex flex-wrap gap-1">
              {ORIGINS.map((o) => (
                <Button
                  key={o.value}
                  size="xs"
                  variant={origin === o.value ? 'secondary' : 'ghost'}
                  aria-pressed={origin === o.value}
                  onClick={() => setOrigin(o.value)}
                >
                  {o.label}
                </Button>
              ))}
            </div>
          </div>

          {isLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : rules.length === 0 ? (
            <Card>
              <CardHeader>
                <CardTitle>Nothing is gating access</CardTitle>
                <CardDescription>
                  This table decides which email addresses may sign in, found an organization, or
                  accept an invitation, and it is empty — so every address that reaches the service
                  is admitted for all three. Normally it is not: the platform seeds the public
                  mailbox domains (gmail.com, outlook.com and the rest) as denials for organization
                  creation, so a tenant cannot be minted from a throwaway address.
                  {seedingEnabled
                    ? ' Seeding is on, so those rows should appear here on the next restart. An empty table before then is worth investigating.'
                    : ' Seeding is switched off (auth.seed_public_domain_rules), which is why they are absent. Add rules here, or turn seeding back on.'}
                </CardDescription>
              </CardHeader>
            </Card>
          ) : (
            // One card per purpose, always all three, because they answer
            // different questions and a single merged list is exactly the
            // conflation the backend's purpose split exists to prevent. A purpose
            // with no rules is information, not an empty state to hide.
            PURPOSES.map((p) => {
              const forPurpose = rules.filter((r) => r.purpose === p.value);
              // Numbered against the whole purpose, then filtered: an ordinal
              // computed after filtering would claim a row is checked first when
              // it is not.
              const ordered = inEvaluationOrder(forPurpose).map((rule, i) => ({
                rule,
                position: i + 1,
              }));
              const shown = ordered.filter((row) => matchesFilters(row.rule));

              return (
                <Card key={p.value}>
                  <CardHeader>
                    <CardTitle className="flex flex-wrap items-center gap-2">
                      {p.label}
                      <Badge variant="outline" className="font-mono">
                        {p.value}
                      </Badge>
                    </CardTitle>
                    <CardDescription>
                      {p.question} {p.blurb} Listed in the order the database checks them: the first
                      row that matches an address decides.
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    {forPurpose.length === 0 ? (
                      <p className="text-sm text-muted-foreground">
                        No rules, so this question is never refused — every address that gets this
                        far is permitted. Add a rule only if this should be restricted.
                      </p>
                    ) : shown.length === 0 ? (
                      <p className="text-sm text-muted-foreground">
                        {forPurpose.length} rule{forPurpose.length === 1 ? '' : 's'} here, none
                        matching the current filter. They are still enforced.
                      </p>
                    ) : (
                      <div className="overflow-x-auto">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead className="w-12">#</TableHead>
                              <TableHead>Matches</TableHead>
                              <TableHead>Policy</TableHead>
                              <TableHead>Origin</TableHead>
                              <TableHead>Reason</TableHead>
                              <TableHead className="sr-only">Actions</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {shown.map(({ rule, position }) => {
                              const note = precedenceNote(rule, forPurpose);
                              return (
                                <TableRow key={rule.id}>
                                  <TableCell className="text-muted-foreground tabular-nums">
                                    {position}
                                  </TableCell>
                                  <TableCell>
                                    <div className="flex flex-col gap-1">
                                      <div className="flex items-center gap-2">
                                        <Badge variant="outline">{rule.kind}</Badge>
                                        <code className="text-xs break-all">{rule.value}</code>
                                      </div>
                                      {note ? (
                                        <span className="text-xs text-muted-foreground">
                                          {note}
                                        </span>
                                      ) : null}
                                    </div>
                                  </TableCell>
                                  <TableCell>
                                    <PolicyChoice
                                      value={rule.policy}
                                      disabled={busy}
                                      label={`Policy for ${rule.value} when ${p.question.toLowerCase()}`}
                                      onChange={(next) => setPolicy(rule, next)}
                                    />
                                  </TableCell>
                                  <TableCell>
                                    <OriginCell rule={rule} />
                                  </TableCell>
                                  <TableCell className="text-muted-foreground">
                                    {rule.reason ?? '—'}
                                  </TableCell>
                                  <TableCell>
                                    <div className="flex justify-end">
                                      {/* No delete on a row the seeder restores.
                                          Removing it looks like the way to permit
                                          a domain and is undone by the next boot,
                                          while the allow beside it is permanent —
                                          offering both side by side would put the
                                          two a keystroke apart. The removal is
                                          still reachable, behind a dialog that
                                          says what happens. */}
                                      <Button
                                        size="xs"
                                        variant="ghost"
                                        disabled={busy}
                                        onClick={() => setRemoving(rule)}
                                      >
                                        {rule.seed_restores_on_restart ? 'Remove…' : 'Delete'}
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
              );
            })
          )}

          {filtering && rules.length > 0 ? (
            <p className="text-xs text-muted-foreground">
              A filter hides rows; it does not disable them. Everything in this table is enforced.
            </p>
          ) : null}
        </>
      )}

      <CreateAccessRuleDialog open={createOpen} onOpenChange={setCreateOpen} onCreated={refresh} />

      <Dialog
        open={allowing !== null}
        onOpenChange={(o) => {
          if (!o) {
            setAllowing(null);
            setAllowReason('');
          }
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Permit {allowing?.value}?</DialogTitle>
            <DialogDescription>
              {allowing
                ? `${allowing.value} will be permitted for ${purposeMeta(
                    allowing.purpose,
                  ).label.toLowerCase()} from the next request onwards. Only that question — the other purposes are decided by their own rules.`
                : null}
              {allowing?.seeded || allowing?.seed_restores_on_restart
                ? ' This rule comes from the public-mailbox corpus, so the change is permanent: the seed pass only inserts rows that are missing and will never restore the deny. Nothing outside this table will show that the domain is permitted, which is why the reason below is worth filling in.'
                : ''}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-2">
            <Label htmlFor="allow-reason">Reason</Label>
            <Input
              id="allow-reason"
              value={allowReason}
              placeholder="Who asked for this and why"
              onChange={(e) => setAllowReason(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Recorded on the row. It replaces whatever reason the rule carries now, and an empty
              box clears it.
            </p>
          </div>

          <DialogFooter>
            <Button
              disabled={update.isPending}
              onClick={() => {
                if (!allowing) return;
                update.mutate({
                  path: { id: allowing.id },
                  body: {
                    policy: 'allow',
                    ...(allowReason.trim() === '' ? {} : { reason: allowReason.trim() }),
                  },
                });
              }}
            >
              {update.isPending ? 'Permitting…' : `Permit ${allowing?.value ?? 'this'}`}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={removing !== null} onOpenChange={(o) => !o && setRemoving(null)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {removing?.seed_restores_on_restart
                ? `Removing ${removing.value} does not last`
                : `Delete the rule for ${removing?.value}?`}
            </DialogTitle>
            <DialogDescription>
              {removing?.seed_restores_on_restart
                ? 'This rule is in the public-mailbox corpus. Deleting it works until the next restart, when the seed pass re-creates it as a create_org deny — so if the intent is to permit this domain, change the policy to allow instead. An update is never overwritten; a deletion is.'
                : `Deleting a rule removes it from evaluation immediately. If no other rule matches an address that this one was deciding, the request is allowed — a missing rule is not a denial. ${
                    removing?.policy === 'deny'
                      ? 'This rule currently denies, so deleting it grants access rather than removing it.'
                      : 'This rule currently allows, so deleting it hands the decision back to whatever broader rule was being overridden.'
                  }`}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            {removing?.seed_restores_on_restart && removing.policy !== 'allow' ? (
              <Button
                disabled={busy}
                onClick={() => {
                  const rule = removing;
                  setRemoving(null);
                  setAllowing(rule);
                  setAllowReason(rule.reason ?? '');
                }}
              >
                Permit it instead — that lasts
              </Button>
            ) : null}
            <Button
              variant="destructive"
              disabled={remove.isPending}
              onClick={() => {
                if (!removing) return;
                remove.mutate({ path: { id: removing.id } });
              }}
            >
              {remove.isPending
                ? 'Deleting…'
                : removing?.seed_restores_on_restart
                  ? 'Delete anyway'
                  : 'Delete rule'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};
