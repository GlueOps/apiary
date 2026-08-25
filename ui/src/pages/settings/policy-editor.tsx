import { useState } from 'react';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

export type PolicyState = 'inherit' | 'enabled' | 'disabled';

export interface PolicyConfig {
  min_length?: number;
  require_uppercase?: boolean;
  require_lowercase?: boolean;
  require_digit?: boolean;
  require_special?: boolean;
  max_lifetime_minutes?: number;
}

export interface Policy {
  kind: string;
  state: string;
  config: PolicyConfig;
  effective_enabled: boolean;
  effective_config: PolicyConfig;
}

/**
 * Human names and, more usefully, what each policy actually costs someone who
 * fails it. An operator deciding whether to turn something on is asking "who
 * does this lock out", and a bare identifier does not answer that.
 */
const KINDS: Record<string, { title: string; blurb: string }> = {
  require_mfa: {
    title: 'Require MFA',
    blurb: 'Members without a second factor cannot sign in once this applies.',
  },
  require_passkey: {
    title: 'Require a passkey',
    blurb: 'Members without a registered passkey cannot sign in once this applies.',
  },
  password_min_length: {
    title: 'Minimum password length',
    blurb: 'Applies when a password is set or changed, not retroactively.',
  },
  password_complexity: {
    title: 'Password complexity',
    blurb: 'Character classes a new password must contain.',
  },
  session_max_lifetime: {
    title: 'Maximum session lifetime',
    blurb: 'Sessions are refused once they exceed this age, and the member signs in again.',
  },
};

const STATE_TONE: Record<string, string> = {
  enabled: 'bg-emerald-500/15 text-emerald-500',
  disabled: 'bg-muted text-muted-foreground',
  inherit: 'bg-muted text-muted-foreground',
};

/**
 * Renders one level of the policy hierarchy.
 *
 * It always shows the EFFECTIVE result beside this level's own setting, because
 * they routinely differ and the difference is the point: an organization can
 * raise the platform's floor but never lower it, so "disabled here" and "off"
 * are not the same thing. A screen that showed only the row you were editing
 * would let someone switch a policy off and reasonably conclude it no longer
 * applied.
 */
export function PolicyEditor({
  title,
  description,
  listOptions,
  listQueryKey,
  setMutation,
  clearMutation,
  canEdit,
  inherits = true,
}: {
  title: string;
  description: string;
  /**
   * Whether a level ABOVE this one can decide the outcome.
   *
   * True for an organization, which sits under the platform. FALSE for the
   * platform, which is the top: nothing is above it, so an unset policy there
   * is not "inherited from" anywhere, and the effective result cannot differ
   * from the setting on the page. EffectivePolicy is called with uuid.Nil at
   * platform level, so the org half is always absent and the two values are the
   * same number by construction.
   *
   * Showing the org vocabulary at platform level asks the reader to work out
   * what it inherits FROM, and the honest answer is nothing.
   */
  inherits?: boolean;
  // Deliberately loose: these are the generated SDK's query and mutation
  // factories, whose types are parameterised per operation. Naming them
  // precisely would mean a generic signature that has to be updated every time
  // an operation is added — for a component whose whole point is that both
  // levels of the hierarchy are the same shape. `object` rather than
  // Record<string, unknown> because the generated option types are interfaces
  // without index signatures.
  listOptions: () => object;
  listQueryKey: () => unknown;
  setMutation: () => object;
  clearMutation: () => object;
  canEdit: boolean;
}) {
  const queryClient = useQueryClient();
  const [drafts, setDrafts] = useState<Record<string, PolicyConfig>>({});

  const { data, isLoading, isError, error } = useQuery(listOptions() as never);
  const refresh = () => queryClient.invalidateQueries({ queryKey: listQueryKey() as never });

  const save = useMutation({
    ...(setMutation() as object),
    onSuccess: () => {
      toast.success('Policy updated');
      refresh();
    },
    onError: (e: unknown) =>
      toast.error((e as { detail?: string })?.detail ?? 'Could not update that policy'),
  });

  const clear = useMutation({
    ...(clearMutation() as object),
    onSuccess: () => {
      toast.success(inherits ? 'Returned to inherit' : 'Cleared');
      refresh();
    },
    onError: (e: unknown) =>
      toast.error((e as { detail?: string })?.detail ?? 'Could not clear that policy'),
  });

  const policies = ((data as { policies?: Policy[] } | undefined)?.policies ?? []) as Policy[];

  const configFor = (p: Policy) => drafts[p.kind] ?? p.config ?? {};
  const patch = (kind: string, next: PolicyConfig) =>
    setDrafts((d) => ({ ...d, [kind]: { ...(d[kind] ?? {}), ...next } }));

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="font-heading text-2xl">{title}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      </div>

      {isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : isError ? (
        // A failure has to say so. Rendering an empty list on error is how a page
        // tells someone their security policy is unset when in fact it could not
        // be read — the opposite of what they need to know.
        <Card>
          <CardHeader>
            <CardTitle>Policies could not be loaded</CardTitle>
            <CardDescription>
              {(error as { detail?: string })?.detail ??
                'The server did not answer. Nothing here reflects what is actually being enforced.'}
            </CardDescription>
          </CardHeader>
        </Card>
      ) : policies.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>No policies returned</CardTitle>
            <CardDescription>
              This should list every policy kind, including ones nobody has set. An empty list means
              the request succeeded but returned nothing, which is worth investigating rather than
              reading as &ldquo;nothing is enforced&rdquo;.
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        policies.map((p) => {
          const meta = KINDS[p.kind] ?? { title: p.kind, blurb: '' };
          const cfg = configFor(p);
          const differs = p.effective_enabled !== (p.state === 'enabled');

          return (
            <Card key={p.kind}>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  {meta.title}
                  <Badge className={STATE_TONE[p.state] ?? STATE_TONE.inherit}>
                    {p.state === 'inherit' && !inherits ? 'not set' : p.state}
                  </Badge>
                </CardTitle>
                <CardDescription>{meta.blurb}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Shown only where a level above can change the answer. At the
                    top of the hierarchy it is the same value twice, and printing
                    it implies a resolution step that does not happen — which is
                    the reading that makes an operator ask what the platform
                    inherits from. Where it CAN differ it is always shown, not
                    only when it does, so nobody has to notice an absence to
                    learn which level is deciding. */}
                {inherits ? (
                <p className="text-sm text-muted-foreground">
                  Effective:{' '}
                  <span className={p.effective_enabled ? 'font-medium text-foreground' : ''}>
                    {p.effective_enabled ? 'enforced' : 'not enforced'}
                  </span>
                  {differs ? (
                    <span className="text-amber-500">
                      {' '}
                      — decided elsewhere, not by this setting
                    </span>
                  ) : null}
                </p>
                ) : null}

                {p.kind === 'password_min_length' ? (
                  <div className="max-w-40 space-y-2">
                    <Label htmlFor={`${p.kind}-min`}>Minimum characters</Label>
                    <Input
                      id={`${p.kind}-min`}
                      type="number"
                      min={8}
                      max={256}
                      value={cfg.min_length ?? 8}
                      disabled={!canEdit}
                      onChange={(e) => patch(p.kind, { min_length: Number(e.target.value) })}
                    />
                  </div>
                ) : null}

                {p.kind === 'session_max_lifetime' ? (
                  <div className="max-w-40 space-y-2">
                    <Label htmlFor={`${p.kind}-life`}>Minutes</Label>
                    <Input
                      id={`${p.kind}-life`}
                      type="number"
                      min={1}
                      value={cfg.max_lifetime_minutes ?? 480}
                      disabled={!canEdit}
                      onChange={(e) =>
                        patch(p.kind, { max_lifetime_minutes: Number(e.target.value) })
                      }
                    />
                  </div>
                ) : null}

                {p.kind === 'password_complexity' ? (
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    {(
                      [
                        ['require_uppercase', 'Uppercase letter'],
                        ['require_lowercase', 'Lowercase letter'],
                        ['require_digit', 'Digit'],
                        ['require_special', 'Special character'],
                      ] as const
                    ).map(([key, label]) => (
                      <label key={key} className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={Boolean(cfg[key])}
                          disabled={!canEdit}
                          onChange={(e) => patch(p.kind, { [key]: e.target.checked })}
                        />
                        {label}
                      </label>
                    ))}
                  </div>
                ) : null}

                {canEdit ? (
                  <div className="flex flex-wrap gap-2">
                    <Button
                      size="sm"
                      onClick={() =>
                        save.mutate({
                          path: { kind: p.kind },
                          body: { state: 'enabled', config: cfg },
                        } as never)
                      }
                    >
                      Enforce
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() =>
                        save.mutate({
                          path: { kind: p.kind },
                          body: { state: 'disabled', config: cfg },
                        } as never)
                      }
                    >
                      Do not enforce
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={p.state === 'inherit'}
                      onClick={() => clear.mutate({ path: { kind: p.kind } } as never)}
                    >
                      Inherit
                    </Button>
                  </div>
                ) : (
                  <p className="text-xs text-muted-foreground">
                    Only an administrator can change this.
                  </p>
                )}
              </CardContent>
            </Card>
          );
        })
      )}
    </div>
  );
}
