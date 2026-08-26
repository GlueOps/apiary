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
export function RefusalList({
  refusals,
  onSync,
}: {
  refusals: RefusalView[];
  onSync?: () => void;
}) {
  if (refusals.length === 0) return null;

  return (
    <ul className="space-y-1.5 text-xs">
      {refusals.map((r) => (
        <li key={r.code} className="flex gap-2">
          <IconAlertTriangle className="mt-0.5 size-3.5 shrink-0 text-amber-500" aria-hidden />
          <span className="text-muted-foreground">
            {r.detail}
            {/* drain_unknown is the one refusal an operator resolves by doing
                nothing to the host at all, so it gets the action inline rather
                than leaving them to work out that a sync is what is wanted. */}
            {r.code === 'drain_unknown' && onSync ? (
              <>
                {' '}
                <button
                  type="button"
                  onClick={onSync}
                  className="underline underline-offset-2 hover:text-foreground"
                >
                  Sync this cluster now
                </button>
              </>
            ) : null}
          </span>
        </li>
      ))}
    </ul>
  );
}

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
 * Keyed by CODE rather than by message. Two hosts refused under the same rule
 * carry details differing only in their numbers, and one row per number is the
 * repetition this exists to remove. The detail shown is the first one seen,
 * which reads as an example of the rule rather than a claim about one host.
 */
export function RemedyCard({ refusals }: { refusals: RefusalView[] }) {
  const byCode = new Map<string, RefusalView>();
  for (const r of refusals) {
    if (r.remedy && !byCode.has(r.code)) byCode.set(r.code, r);
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
            {[...byCode.values()].map((r) => (
              <tr key={r.code} className="border-b align-top last:border-0">
                <td className="py-2 pr-4 text-muted-foreground">{r.detail}</td>
                <td className="py-2">{r.remedy}</td>
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
