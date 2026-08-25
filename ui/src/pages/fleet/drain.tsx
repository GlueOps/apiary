import { IconAlertTriangle, IconInfoCircle } from '@tabler/icons-react';

import { Badge } from '@/components/ui/badge';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
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
