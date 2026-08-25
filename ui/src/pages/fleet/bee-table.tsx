import { IconPlugConnected, IconTrash } from '@tabler/icons-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { BeeView } from '@/sdk';

/**
 * A bee's cluster membership, or null for a hypervisor that is in no cluster.
 *
 * Membership is nullable on the server and "no cluster" is a correct state, not a
 * missing value: a hypervisor leaves one cluster before it joins the next, and a join
 * that failed leaves it sitting here. The generated type still declares
 * `hive_id` as a required string, so the absent case arrives as undefined, as an
 * empty string, or as the nil uuid depending on how the row was serialised.
 * Treating any of those as a cluster id would put a hypervisor under a cluster card that
 * does not exist — which is exactly the hiding this page is meant to stop.
 */
const NIL_UUID = '00000000-0000-0000-0000-000000000000';

export function hiveIdOf(bee: BeeView): string | null {
  const id: string | undefined = bee.hive_id;
  if (!id || id === NIL_UUID) return null;
  return id;
}

/**
 * How to name an agent on screen.
 *
 * BeeView lost `hostname` when the machine record moved onto the hypervisor row:
 * an agent no longer carries a name of its own, only the address the control
 * plane reaches its machine on. That address is optional on the wire — a row
 * enrolled before the machine answered has none — and a nameless row is still
 * one an operator has to be able to point at, so the fallback is the id rather
 * than a dash.
 *
 * This is NOT the cluster node name. The provider's node name lives on
 * HypervisorView.name and is the only thing `fleet.datacenter.evict` will accept;
 * passing an address there is refused by the agent as a non-member.
 */
export function beeLabel(bee: BeeView): string {
  return bee.address?.trim() || `agent ${bee.id.slice(0, 8)}`;
}

/**
 * Bee status is DERIVED by the server, not stored, and 'unavailable' is the
 * whole point of the model: a hypervisor that declared it was rebooting is not an
 * outage. Rendering it the same colour as offline would undo that distinction
 * at exactly the moment it matters — an operator scanning a rotation for red.
 */
const STATUS_TONE: Record<string, { label: string; className: string }> = {
  online: { label: 'online', className: 'bg-emerald-500/15 text-emerald-500' },
  unavailable: {
    label: 'away (expected)',
    className: 'bg-amber-500/15 text-amber-500',
  },
  offline: { label: 'offline', className: 'bg-destructive/15 text-destructive' },
};

export function StatusBadge({ status }: { status: string }) {
  const tone = STATUS_TONE[status] ?? STATUS_TONE.offline;
  return <Badge className={tone.className}>{tone.label}</Badge>;
}

export function relative(iso?: string) {
  if (!iso) return 'never';
  const seconds = Math.round((Date.now() - new Date(iso).getTime()) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m ago`;
  return `${Math.round(seconds / 3600)}h ago`;
}

export type BeeTableProps = {
  bees: BeeView[];
  /**
   * Rendered for hypervisors in no cluster. Their row offers a way INTO a cluster; a hypervisor
   * that is already in one leaves by being evicted from a node that remains, so
   * the two row shapes are genuinely different and not a toggle.
   */
  hiveless?: boolean;
  onPing: (bee: BeeView) => void;
  onUpgrade: (bee: BeeView) => void;
  onJoin: (bee: BeeView) => void;
  onEvict: (bee: BeeView) => void;
  busy?: boolean;
};

export function BeeTable({
  bees,
  hiveless = false,
  onPing,
  onUpgrade,
  onJoin,
  onEvict,
  busy = false,
}: BeeTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Hypervisor</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Last seen</TableHead>
          <TableHead>Agent</TableHead>
          <TableHead />
        </TableRow>
      </TableHeader>
      <TableBody>
        {bees.map((b) => (
          <TableRow key={b.id}>
            <TableCell className="font-medium">
              <div className="flex flex-col gap-1">
                <span>{beeLabel(b)}</span>
                {hiveless ? (
                  // Repeated on the row and not only on the card heading: a hypervisor
                  // in no cluster is the one an operator screenshots, pastes and
                  // forwards, and the row has to still say so out of context.
                  <Badge variant="outline" className="w-fit">
                    no cluster
                  </Badge>
                ) : null}
              </div>
            </TableCell>
            <TableCell>
              <div className="flex flex-col gap-1">
                <StatusBadge status={b.status} />
                {b.unavailable_reason ? (
                  <span className="text-xs text-muted-foreground">{b.unavailable_reason}</span>
                ) : null}
              </div>
            </TableCell>
            <TableCell className="text-muted-foreground tabular-nums">
              {relative(b.last_seen_at)}
            </TableCell>
            <TableCell className="text-muted-foreground">{b.agent_version || '—'}</TableCell>
            <TableCell className="text-right">
              <div className="flex justify-end gap-2">
                <Button size="xs" variant="outline" disabled={busy} onClick={() => onPing(b)}>
                  Ping
                </Button>
                <Button size="xs" variant="outline" onClick={() => onUpgrade(b)}>
                  Upgrade…
                </Button>
                {hiveless ? (
                  <Button size="xs" onClick={() => onJoin(b)}>
                    <IconPlugConnected /> Join a cluster…
                  </Button>
                ) : (
                  <Button size="xs" variant="ghost" onClick={() => onEvict(b)}>
                    <IconTrash /> Evict a node…
                  </Button>
                )}
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
