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
import { fleetEnrolmentTokensMintMutation } from '@/sdk/@tanstack/react-query.gen';

const TTL_MINUTES = 60;

export type EnrolmentDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

type Minted = { token: string; expires_at: string; base_url: string };

/**
 * Mints an enrolment token and shows the command to run, once.
 *
 * The token admits a hypervisor into the ORGANIZATION. It does not carry a cluster, and a
 * hypervisor that has just enrolled belongs to none — the same state a hypervisor passes
 * through mid-migration. There is deliberately no cluster field: this dialog used to
 * require one and disable minting until a cluster existed, which locked a fresh
 * installation out of the only path that produces a hypervisor able to bootstrap one.
 *
 * The command carries --base-url because the agent defaults to http://localhost:8080,
 * which on a hypervisor is the hypervisor. The server supplies the value; an operator
 * reading a token off a screen has no way to know which address the control plane
 * answers on, and a command that omits it fails as a connection refused nowhere near
 * anything that mentions enrolment.
 */
export function EnrolmentDialog({ open, onOpenChange }: EnrolmentDialogProps) {
  const [minted, setMinted] = useState<Minted | null>(null);

  const mint = useMutation({
    ...fleetEnrolmentTokensMintMutation(),
    // Held in state rather than toasted: it is shown once and the operator has
    // to copy it. A toast that disappears after four seconds would mean minting
    // another, and every mint is a credential that can admit a machine.
    onSuccess: (res) =>
      setMinted({ token: res.token, expires_at: res.expires_at, base_url: res.base_url ?? '' }),
    onError: (e) => toast.error((e as { detail?: string })?.detail ?? 'Could not mint a token'),
  });

  // Empty base_url means the control plane has no address worth handing out —
  // unset, relative, or loopback. Rendering the command without the flag would
  // silently reintroduce the localhost default, so the placeholder is left in
  // deliberately: it does not run, which is the point.
  const knowsItsAddress = Boolean(minted?.base_url);
  // --install, because enrolment alone leaves the host UNSUPERVISED. It obtains
  // a credential and nothing else: no systemd unit, nothing restarting the agent
  // if it dies, no watchdog for one that wedges, and nothing bringing it back
  // after a reboot. Such a host heartbeats once, from the operator's shell, and
  // afterwards is indistinguishable from one nobody onboarded.
  const command = minted
    ? `apiary-agent enrol --token ${minted.token} --base-url ${minted.base_url || '<control-plane-url>'} --install`
    : '';

  const close = (next: boolean) => {
    if (!next) {
      // Dropped on close, not kept for a second look. The server stores only a
      // hash, so a token left rendered in a closed dialog is a live credential
      // sitting in the DOM of a tab nobody is watching.
      setMinted(null);
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
            hypervisor migrating between clusters takes, and the reason this works before any
            cluster exists.
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
              {knowsItsAddress ? null : (
                <p className="text-xs text-amber-600 dark:text-amber-500">
                  This control plane has no external address configured, so the command above has a
                  placeholder in it. Replace <code>&lt;control-plane-url&gt;</code> with the address
                  the hypervisor can reach, and set <code>BASE_URL</code> to that address so the
                  next token comes out ready to paste. Leaving the flag off entirely makes the agent
                  call <code>http://localhost:8080</code>, which from a hypervisor means the
                  hypervisor.
                </p>
              )}
              <p className="text-xs text-muted-foreground">
                Only a hash of the token is stored. Any hypervisor can redeem it until it expires,
                which is what lets a rack be enrolled in one batch.{' '}
                <code className="rounded bg-muted px-1 py-0.5">--install</code> also writes the
                systemd unit and starts the service; without it the host is credentialled but
                nothing supervises the agent or brings it back after a reboot.
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
        ) : (
          <p className="text-sm text-muted-foreground">
            The token is shown once, as a command to run on the hypervisor. It stays valid for{' '}
            {TTL_MINUTES} minutes and any number of hypervisors can redeem it in that window.
          </p>
        )}

        {minted ? null : (
          <DialogFooter>
            <Button
              disabled={mint.isPending}
              onClick={() => mint.mutate({ body: { ttl_minutes: TTL_MINUTES } })}
            >
              {mint.isPending ? 'Minting…' : `Mint a token (${TTL_MINUTES} min)`}
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}
