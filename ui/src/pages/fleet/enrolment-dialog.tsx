import { useState } from 'react';

import { IconAlertTriangle, IconCopy } from '@tabler/icons-react';
import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { fleetEnrolmentTokensMintMutation } from '@/sdk/@tanstack/react-query.gen';

const TTL_MINUTES = 60;

export type EnrolmentDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  clusters: { id: string; name: string }[];
};

/**
 * Mints an enrolment token and shows it once.
 *
 * The token admits a hypervisor into the ORGANIZATION. It does not carry a cluster, and
 * a hypervisor that has just enrolled belongs to none — the same state a hypervisor passes
 * through mid-migration. The cluster field below is not a target: the endpoint
 * still requires one to exist before it will mint, and nothing about it reaches
 * the token or the hypervisor.
 */
export function EnrolmentDialog({ open, onOpenChange, clusters }: EnrolmentDialogProps) {
  const [hiveId, setHiveId] = useState<string>('');
  const [minted, setMinted] = useState<{ token: string; expires_at: string } | null>(null);

  const mint = useMutation({
    ...fleetEnrolmentTokensMintMutation(),
    // Held in state rather than toasted: it is shown once and the operator has
    // to copy it. A toast that disappears after four seconds would mean minting
    // another, and every mint is a credential that can admit a machine.
    onSuccess: (res) => setMinted({ token: res.token, expires_at: res.expires_at }),
    onError: (e) => toast.error((e as { detail?: string })?.detail ?? 'Could not mint a token'),
  });

  const selected = hiveId || clusters[0]?.id || '';
  const command = minted ? `apiary-agent enrol --token ${minted.token}` : '';

  const close = (next: boolean) => {
    if (!next) {
      // Dropped on close, not kept for a second look. The server stores only a
      // hash, so a token left rendered in a closed dialog is a live credential
      // sitting in the DOM of a tab nobody is watching.
      setMinted(null);
      setHiveId('');
    }
    onOpenChange(next);
  };

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Enrol a hypervisor</DialogTitle>
          <DialogDescription>
            A token admits a hypervisor into this organization, not into a cluster. The hypervisor
            arrives belonging to no cluster and joins one afterwards, as a task — the same path a
            hypervisor migrating between clusters takes.
          </DialogDescription>
        </DialogHeader>

        {minted ? (
          <div className="space-y-4">
            <div className="space-y-2 rounded-lg border border-amber-500/40 bg-amber-500/5 p-3">
              <p className="flex items-center gap-2 text-sm font-medium text-amber-600 dark:text-amber-500">
                <IconAlertTriangle className="size-4" /> Copy this now — it is never shown again
              </p>
              <code className="block overflow-x-auto rounded bg-muted p-3 text-xs break-all">
                {command}
              </code>
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    void navigator.clipboard.writeText(command);
                    toast.success('Copied');
                  }}
                >
                  <IconCopy /> Copy command
                </Button>
                <span className="text-xs text-muted-foreground">
                  Expires {new Date(minted.expires_at).toLocaleString()}
                </span>
              </div>
              <p className="text-xs text-muted-foreground">
                Only a hash of the token is stored. Any hypervisor can redeem it until it expires,
                which is what lets a rack be enrolled in one batch.
              </p>
            </div>

            <div className="space-y-1">
              <p className="text-sm font-medium">What the hypervisor receives</p>
              <p className="text-sm text-muted-foreground">
                Redeeming the token issues that hypervisor an agent key and secret, which it sends
                as <code className="rounded bg-muted px-1 py-0.5 text-xs">X-AGENT-Key</code> and{' '}
                <code className="rounded bg-muted px-1 py-0.5 text-xs">X-AGENT-Secret</code> from
                then on. The secret is returned once, straight into the agent's local store, and
                nothing can reissue it — an endpoint that could would let anyone holding a
                hypervisor id take that hypervisor over. A hypervisor that loses it enrols again
                with the same machine id, which replaces its bee rather than creating a second one.
              </p>
            </div>
          </div>
        ) : clusters.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            The mint endpoint requires a cluster to exist before it will issue a token, even though
            the token does not carry one. Create a cluster first, then come back here.
          </p>
        ) : (
          <div className="space-y-2">
            <Label htmlFor="enrol-cluster">Cluster</Label>
            <Select
              value={selected}
              onValueChange={(v) => {
                if (typeof v === 'string') setHiveId(v);
              }}
            >
              <SelectTrigger id="enrol-cluster" className="w-full">
                <SelectValue>
                  {(v: string | null) =>
                    clusters.find((c) => c.id === v)?.name ?? 'Select a cluster'
                  }
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {clusters.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Not a destination. The endpoint checks a cluster exists before minting; the token
              carries only the organization, and the enrolled hypervisor lands in no cluster.
            </p>
          </div>
        )}

        {minted ? null : (
          <DialogFooter>
            <Button
              disabled={mint.isPending || !selected}
              onClick={() => mint.mutate({ body: { hive_id: selected, ttl_minutes: TTL_MINUTES } })}
            >
              {mint.isPending ? 'Minting…' : `Mint a token (${TTL_MINUTES} min)`}
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}
