import { IconAlertTriangle, IconInfoCircle } from '@tabler/icons-react';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import type { RefusalView } from '@/sdk';

/**
 * Drain state is DERIVED by the server, never stored, and 'unknown' is the one
 * that must not be collapsed into the others.
 *
 * 'unknown' means the capacity observation is too stale to say whether the node
 * is empty — it is the absence of an answer, not a synonym for 'draining'. The
 * server refuses a departure on it, and the fix is a sync rather than moving
 * guests, so an operator shown 'draining' would go looking for workloads that
 * are not there.
 */
const DRAIN_TONE: Record<string, { label: string; className: string; hint: string }> = {
  active: {
    label: 'active',
    className: 'bg-emerald-500/15 text-emerald-500',
    hint: 'Accepting new placements.',
  },
  draining: {
    label: 'draining',
    className: 'bg-amber-500/15 text-amber-500',
    hint: 'Cordoned, but work is still on it. Move or delete the guests.',
  },
  drained: {
    label: 'drained',
    className: 'bg-sky-500/15 text-sky-500',
    hint: 'Cordoned and empty. Ready to leave its cluster.',
  },
  unknown: {
    label: 'unknown',
    className: 'bg-muted text-muted-foreground',
    hint: 'The capacity observation is too stale to say whether this node is empty. Run a sync.',
  },
};

export function DrainBadge({ state }: { state: string }) {
  const tone = DRAIN_TONE[state] ?? DRAIN_TONE.unknown;
  return (
    <Badge className={tone.className} title={tone.hint}>
      {tone.label}
    </Badge>
  );
}

export function drainHint(state: string): string {
  return (DRAIN_TONE[state] ?? DRAIN_TONE.unknown).hint;
}

/**
 * Refusals are rendered from the server's list, never recomputed here.
 *
 * The gate lives in one place — the same code the departure job runs — and a
 * second implementation in the browser is how a button comes to be enabled for
 * an operation the server will refuse. This component only decides how to say it.
 */
/**
 * Every distinct refusal on the page, with its remedy, once.
 *
 * The remedies were briefly rendered under each machine's own list of reasons,
 * which put the same paragraphs beside every host that shared a problem — a
 * fleet where nothing is cordoned repeats "cordon it" per node, and each row
 * grows wide enough to scroll sideways. The REASON is about one machine and
 * belongs on its row; the FIX for a reason is the same wherever it appears, so
 * it belongs once, at the bottom, where somebody goes looking for what to do.
 *
 * Keyed by CODE and rendered from SUMMARY, which is the generic form of the
 * rule. Grouping on the detail instead would mean picking one host's figures to
 * display — "the provider reports 2 guest(s)" shown for a group that also
 * contains a host with five is not a shorter message, it is a wrong one.
 *
 * The count of affected hosts is shown instead, which is the number that is
 * actually true of the group and more useful than any one host's: it says how
 * much work the fix is.
 */
export function RemedyCard({ refusals, onSync }: { refusals: RefusalView[]; onSync?: () => void }) {
  const byCode = new Map<string, { refusal: RefusalView; hosts: number }>();
  for (const r of refusals) {
    if (!r.remedy) continue;
    const seen = byCode.get(r.code);
    if (seen) seen.hosts += 1;
    else byCode.set(r.code, { refusal: r, hosts: 1 });
  }
  if (byCode.size === 0) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">What is blocking these hosts</CardTitle>
        <CardDescription>
          Each reason appears once, whichever hosts it applies to. The hosts themselves are marked
          above.
        </CardDescription>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left text-xs text-muted-foreground">
              <th className="w-1/3 pr-4 pb-2 font-medium">Detail</th>
              <th className="pb-2 font-medium">Remedy</th>
            </tr>
          </thead>
          <tbody>
            {[...byCode.values()].map(({ refusal: r, hosts }) => (
              <tr key={r.code} className="border-b align-top last:border-0">
                <td className="py-2 pr-4 text-muted-foreground">
                  <span className="flex gap-2">
                    <IconAlertTriangle
                      className="mt-0.5 size-3.5 shrink-0 text-amber-500"
                      aria-hidden
                    />
                    <span>
                      {/* The generic form. Falling back to the detail only for a
                          refusal that has no summary yet, where showing one
                          host's figures is still better than showing nothing. */}
                      {r.summary || r.detail}
                      <span className="ml-1 text-muted-foreground/70">
                        · {hosts} {hosts === 1 ? 'host' : 'hosts'}
                      </span>
                    </span>
                  </span>
                </td>
                <td className="py-2">
                  {r.remedy}
                  {/* drain_unknown is the one refusal resolved by touching no
                      host at all, so the action is offered rather than left for
                      the operator to work out that a sync is what is wanted. */}
                  {r.code === 'drain_unknown' && onSync ? (
                    <>
                      {' '}
                      <button
                        type="button"
                        onClick={onSync}
                        className="underline underline-offset-2 hover:text-foreground"
                      >
                        Sync now
                      </button>
                    </>
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </CardContent>
    </Card>
  );
}

export function Hint({ children }: { children: React.ReactNode }) {
  return (
    <p className="flex gap-2 text-xs text-muted-foreground">
      <IconInfoCircle className="mt-0.5 size-3.5 shrink-0" aria-hidden />
      <span>{children}</span>
    </p>
  );
}

export function capacityLabel(c: { cpu: number; ram_gb: number; disk_gb: number }) {
  return `${c.cpu} vCPU · ${c.ram_gb} GiB · ${c.disk_gb} GiB`;
}

/**
 * Rendered when a list query FAILS.
 *
 * Without it a failed request is indistinguishable from an empty result: the
 * page falls through to its empty state and says "nothing here", which is a
 * different and much more reassuring claim than "we could not ask". That is
 * exactly how a create-then-refetch failure reads as "the thing I just made does
 * not exist".
 */
export function LoadError({ what, error }: { what: string; error: unknown }) {
  const detail =
    (error as { detail?: string; message?: string } | undefined)?.detail ??
    (error as { message?: string } | undefined)?.message ??
    'The server did not answer.';
  return (
    <Card className="border-destructive/50">
      <CardHeader>
        <CardTitle className="text-base">{what} could not be loaded</CardTitle>
        <CardDescription>
          {detail} Nothing below reflects the current state — this is a failed request, not an empty
          result.
        </CardDescription>
      </CardHeader>
    </Card>
  );
}
