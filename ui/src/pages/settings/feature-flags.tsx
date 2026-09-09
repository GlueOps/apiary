import { useState } from 'react';

import { IconRefresh, IconRotate } from '@tabler/icons-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { FlagView } from '@/sdk';
import {
  platformFlagClearMutation,
  platformFlagOverridesOptions,
  platformFlagOverridesQueryKey,
  platformFlagSetMutation,
  platformFlagsListOptions,
  platformFlagsListQueryKey,
  platformOrgFlagSetMutation,
  platformOrganizationsListOptions,
} from '@/sdk/@tanstack/react-query.gen';

import { LoadError } from '../fleet/drain';

/**
 * Where a value came from, said in words rather than left to a bare toggle.
 *
 * "Off", "off because the platform says so" and "off because this tenant was
 * switched off" send an operator to three different places, and a checkbox sends
 * them to the wrong one twice. This is the same reasoning the policy editor uses
 * for showing the effective result beside the level being edited — with the
 * difference that here the narrower level can go either way, so the effective
 * value is not derivable from the row on screen.
 */
function SourceBadge({ flag }: { flag: FlagView }) {
  if (flag.source === 'organization') {
    return <Badge className="bg-sky-500/15 text-sky-500">set for one organization</Badge>;
  }
  if (flag.source === 'platform') {
    return <Badge className="bg-emerald-500/15 text-emerald-500">set here</Badge>;
  }
  return <Badge variant="outline">built-in default</Badge>;
}

function ValuePill({ on }: { on: boolean }) {
  return on ? (
    <Badge className="bg-emerald-500/15 text-emerald-500">on</Badge>
  ) : (
    <Badge variant="secondary">off</Badge>
  );
}

/**
 * The organizations that have set this flag themselves.
 *
 * Loaded per flag and only for tenant-scoped ones, because it is the question an
 * operator asks immediately before changing an installation-wide value: these
 * are exactly the tenants the change will NOT reach. Showing the count in the
 * collapsed state is what makes that visible without a click.
 */
function Overrides({ flag }: { flag: FlagView }) {
  const qc = useQueryClient();
  const [search, setSearch] = useState('');

  const query = useQuery({
    ...platformFlagOverridesOptions({ path: { key: flag.key } }),
    enabled: flag.scope === 'tenant',
  });

  // Only once something is typed, and only for a flag that can carry an
  // override. Fetching every tenant for every flag card on the page would be one
  // full directory read per flag, for a list nobody has asked to see yet.
  const orgs = useQuery({
    ...platformOrganizationsListOptions({ query: { q: search.trim(), limit: 20 } }),
    enabled: flag.scope === 'tenant' && search.trim().length > 0,
  });

  const setOrg = useMutation({
    ...platformOrgFlagSetMutation(),
    onSuccess: () => {
      toast.success('Override saved');
      setSearch('');
      void qc.invalidateQueries({
        queryKey: platformFlagOverridesQueryKey({ path: { key: flag.key } }),
      });
    },
    onError: (e: { detail?: string }) => toast.error(e?.detail ?? 'Could not set the override'),
  });

  if (flag.scope !== 'tenant') {
    return (
      <p className="text-xs text-muted-foreground">
        Installation-wide. There is no per-organization value for this one — a single tenant cannot
        have its own login page.
      </p>
    );
  }

  const rows = query.data?.overrides ?? [];
  // An organization that already has a row is offered through the list above,
  // where its current value is visible. Offering it twice invites setting it
  // from the half of the screen that does not say what it is now.
  const already = new Set(rows.map((o) => o.organization_id));
  const candidates = (orgs.data?.organizations ?? []).filter((o) => !already.has(o.id));

  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <p className="text-xs font-medium">
          {rows.length === 0
            ? 'No organization has overridden this'
            : `${rows.length} organization${rows.length === 1 ? '' : 's'} override this`}
        </p>
        {/* The reason this list is here at all, stated once. */}
        <p className="text-xs text-muted-foreground">
          A change to the value above does not reach these — each has an opinion of its own.
        </p>
      </div>

      {rows.length > 0 ? (
        <ul className="space-y-1">
          {rows.map((o) => (
            <li key={o.organization_id} className="flex items-center gap-2 text-xs">
              <ValuePill on={o.value} />
              {/* The name leads; the id is the thing you paste back in, so it
                  stays visible rather than hidden behind a tooltip. */}
              <span>{o.organization_name || '(deleted organization)'}</span>
              <span className="font-mono text-muted-foreground">
                {o.organization_id.slice(0, 8)}
              </span>
              <Button
                size="sm"
                variant="ghost"
                className="h-6 px-2"
                onClick={() =>
                  setOrg.mutate({
                    path: { org_id: o.organization_id, key: flag.key },
                    body: { value: !o.value },
                  })
                }
              >
                turn {o.value ? 'off' : 'on'}
              </Button>
            </li>
          ))}
        </ul>
      ) : null}

      <div className="space-y-2">
        <Label htmlFor={`org-${flag.key}`} className="text-xs">
          Override for an organization
        </Label>
        <Input
          id={`org-${flag.key}`}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search organizations by name or domain"
          className="h-8 w-80 text-xs"
        />
        {/* Only once something has been typed. A list of every tenant sitting
            open under each flag is noise on a screen whose subject is the flag,
            and it invites clicking the wrong one. */}
        {search.trim() ? (
          orgs.isLoading ? (
            <p className="text-xs text-muted-foreground">Searching…</p>
          ) : candidates.length === 0 ? (
            <p className="text-xs text-muted-foreground">
              No organization matches. Deleted ones are not listed.
            </p>
          ) : (
            <ul className="max-h-48 space-y-1 overflow-y-auto rounded-md border p-2">
              {candidates.map((o) => (
                <li key={o.id} className="flex items-center justify-between gap-3 text-xs">
                  <span className="min-w-0">
                    <span className="block truncate">{o.name}</span>
                    {o.domain ? (
                      <span className="block truncate text-muted-foreground">{o.domain}</span>
                    ) : null}
                  </span>
                  <span className="flex shrink-0 items-center gap-1">
                    <Button
                      size="sm"
                      variant="outline"
                      className="h-6 px-2"
                      disabled={setOrg.isPending}
                      onClick={() =>
                        setOrg.mutate({
                          path: { org_id: o.id, key: flag.key },
                          body: { value: true },
                        })
                      }
                    >
                      on
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      className="h-6 px-2"
                      disabled={setOrg.isPending}
                      onClick={() =>
                        setOrg.mutate({
                          path: { org_id: o.id, key: flag.key },
                          body: { value: false },
                        })
                      }
                    >
                      off
                    </Button>
                  </span>
                </li>
              ))}
            </ul>
          )
        ) : null}
      </div>
    </div>
  );
}

function FlagCard({ flag }: { flag: FlagView }) {
  const qc = useQueryClient();
  const invalidate = () => void qc.invalidateQueries({ queryKey: platformFlagsListQueryKey() });

  // The server refuses to remove the last way of signing in. It is a refusal an
  // operator may legitimately override — configuring a provider straight
  // afterwards is a real sequence — so the refusal has to be actionable here
  // rather than a toast telling them to go and use curl.
  const [lockout, setLockout] = useState<string | null>(null);

  const set = useMutation({
    ...platformFlagSetMutation(),
    onSuccess: () => {
      toast.success('Saved');
      setLockout(null);
      invalidate();
    },
    onError: (e: { detail?: string; status?: number }) => {
      const detail = e?.detail ?? 'Could not set the flag';
      if (detail.includes('no way to sign in')) {
        // Shown in place, not as a toast. A toast disappears, and this is a
        // decision the operator has to read before making.
        setLockout(detail);
        return;
      }
      toast.error(detail);
    },
  });

  const clear = useMutation({
    ...platformFlagClearMutation(),
    onSuccess: () => {
      toast.success('Restored the built-in default');
      invalidate();
    },
    onError: (e: { detail?: string }) => toast.error(e?.detail ?? 'Could not clear the flag'),
  });

  const busy = set.isPending || clear.isPending;
  const isSetHere = flag.source === 'platform';

  return (
    <Card>
      <CardHeader className="gap-2">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <CardTitle className="font-mono text-base">{flag.key}</CardTitle>
          <div className="flex items-center gap-2">
            <ValuePill on={flag.value} />
            <SourceBadge flag={flag} />
          </div>
        </div>
        <CardDescription>{flag.why}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-wrap items-center gap-2">
          <Button
            size="sm"
            variant={flag.value && isSetHere ? 'default' : 'outline'}
            disabled={busy}
            onClick={() => set.mutate({ path: { key: flag.key }, body: { value: true } })}
          >
            On
          </Button>
          <Button
            size="sm"
            variant={!flag.value && isSetHere ? 'default' : 'outline'}
            disabled={busy}
            onClick={() => set.mutate({ path: { key: flag.key }, body: { value: false } })}
          >
            Off
          </Button>
          {/* CLEARING IS NOT SETTING FALSE, and the button says so rather than
              looking like a third state. A cleared flag falls back to what this
              build was written to do, which is the only way to restore a default
              without knowing what it used to be. */}
          <Button
            size="sm"
            variant="ghost"
            disabled={busy || !isSetHere}
            onClick={() => clear.mutate({ path: { key: flag.key } })}
            title={
              isSetHere
                ? `Remove the installation-wide value; falls back to ${flag.default ? 'on' : 'off'}`
                : 'Nothing is set here to remove'
            }
          >
            <IconRotate className="size-4" />
            Use the default ({flag.default ? 'on' : 'off'})
          </Button>
        </div>

        {lockout ? (
          <div className="space-y-2 rounded-md border border-destructive/40 bg-destructive/5 p-3">
            <p className="text-xs font-medium text-destructive">Refused: {lockout}</p>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                size="sm"
                variant="destructive"
                disabled={busy}
                onClick={() =>
                  set.mutate({
                    path: { key: flag.key },
                    body: { value: false, acknowledge_lockout: true },
                  })
                }
              >
                Turn it off anyway
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setLockout(null)}>
                Cancel
              </Button>
            </div>
          </div>
        ) : null}

        <div className="border-t pt-3">
          <Overrides flag={flag} />
        </div>
      </CardContent>
    </Card>
  );
}

export const FeatureFlags = () => {
  const query = useQuery(platformFlagsListOptions());
  const flags = query.data?.flags ?? [];

  return (
    <div className="w-full space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-heading text-2xl">Feature flags</h1>
          <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
            What is switched on, for whom. Unlike platform policy — which sets a floor an
            organization can only raise — a flag set for one organization overrides the value here
            in either direction, which is the point of being able to leave one tenant behind during
            a migration.
          </p>
          {/* Why the list is not editable in the way an operator might expect. */}
          <p className="mt-2 max-w-3xl text-xs text-muted-foreground">
            Flags are declared in the build. This list is every flag this version knows about — a
            key that is not here cannot be set, which is what stops a mistyped name from being
            stored and reported as saved.
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

      {query.isError ? (
        <LoadError what="The feature flags" error={query.error} />
      ) : query.isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : flags.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">This build declares no flags</CardTitle>
            <CardDescription>
              Flags are registered in code, so an empty list means none are declared rather than
              none being set.
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <div className="grid gap-4">
          {flags.map((f) => (
            <FlagCard key={f.key} flag={f} />
          ))}
        </div>
      )}
    </div>
  );
};
