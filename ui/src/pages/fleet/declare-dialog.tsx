import { useState } from 'react';

import { IconAlertTriangle, IconServerBolt } from '@tabler/icons-react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router';
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
  fleetHostsFleetlessQueryKey,
  fleetHypervisorsDeclareMutation,
  fleetHypervisorsListQueryKey,
} from '@/sdk/@tanstack/react-query.gen';
import type { DeclaredHostView } from '@/sdk/types.gen';

export type DeclareDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

/**
 * Bringing bare hosts into the fleet by address.
 *
 * This is the entry point the automated path was missing. Machine records were
 * only ever created by the capacity sync — which needs the host to already be a
 * node in a configured cluster — or by an agent enrolling, which needs somebody
 * to have run a command on it by hand. Neither describes a machine that has just
 * been racked, so there was a trigger, a ticker and a worker with no way to name
 * a host for them to work on.
 *
 * A textarea rather than a repeated single field: the list comes out of a
 * spreadsheet, an inventory file or a terminal, and making an operator add
 * thirty-two hosts one at a time is where a typo enters.
 */
export function DeclareDialog({ open, onOpenChange }: DeclareDialogProps) {
  const qc = useQueryClient();
  const [text, setText] = useState('');
  const [results, setResults] = useState<DeclaredHostView[] | null>(null);

  const declare = useMutation({
    ...fleetHypervisorsDeclareMutation(),
    onSuccess: (res) => {
      setResults(res.hosts);
      toast.success(
        res.created === 0
          ? 'Every address was already known'
          : `${res.created} queued for onboarding`,
      );
      void qc.invalidateQueries({ queryKey: fleetHypervisorsListQueryKey() });
      void qc.invalidateQueries({ queryKey: fleetHostsFleetlessQueryKey() });
    },
    // The server rejects a whole paste when one line is malformed, and names the
    // corrected form. Surfaced verbatim: with thirty addresses on screen, "what
    // to change" is the only useful message.
    onError: (e) =>
      toast.error((e as { detail?: string })?.detail ?? 'Could not record those hosts'),
  });

  const close = (next: boolean) => {
    if (!next) {
      setText('');
      setResults(null);
    }
    onOpenChange(next);
  };

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Add hypervisors</DialogTitle>
          <DialogDescription>
            Records these machines and queues them. A worker connects to each over SSH, installs the
            agent and enrols it — no command is run on the host by hand.
          </DialogDescription>
        </DialogHeader>

        {results ? (
          <div className="space-y-3">
            <div className="max-h-80 space-y-1 overflow-auto rounded border p-2">
              {results.map((r) => (
                <div key={r.address} className="flex items-baseline justify-between gap-3 text-sm">
                  <code className="font-mono">{r.address}</code>
                  <span
                    className={
                      r.created ? 'text-emerald-600 dark:text-emerald-400' : 'text-muted-foreground'
                    }
                  >
                    {r.created ? 'queued' : (r.detail ?? 'already known')}
                  </span>
                </div>
              ))}
            </div>
            <p className="text-sm text-muted-foreground">
              Queued machines are claimed on the next tick. Watch their state on this page —
              onboarding failures are shown against the machine, with the reason.
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2 rounded-lg border border-amber-500/40 bg-amber-500/5 p-3">
              <p className="flex items-center gap-2 text-sm font-medium text-amber-600 dark:text-amber-500">
                <IconAlertTriangle className="size-4" /> These hosts must already trust the fleet
                key
              </p>
              <p className="text-xs text-muted-foreground">
                The control plane connects with the private half of your fleet SSH key, so its
                public half has to be in each host&apos;s{' '}
                <code className="rounded bg-muted px-1 py-0.5">authorized_keys</code> before this
                can do anything. Without it every attempt fails as a refused connection.{' '}
                <Link to="/settings/ssh-keys" className="underline">
                  Fleet SSH keys
                </Link>{' '}
                has the command to run on each host.
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="declare-hosts">Addresses</Label>
              <textarea
                id="declare-hosts"
                className="min-h-40 w-full rounded-md border bg-transparent p-3 font-mono text-sm"
                value={text}
                onChange={(e) => setText(e.target.value)}
                placeholder={'pve1\npve2, pve3\n10.10.10.4\n\n# spares\npve7'}
              />
              <p className="text-xs text-muted-foreground">
                One per line, or separated by commas or spaces. Blank lines and #-comments are
                ignored. Hostnames or IPs — not URLs, and not host:port, because the connection is
                SSH.
              </p>
            </div>
          </div>
        )}

        <DialogFooter>
          {results ? (
            <Button onClick={() => close(false)}>Done</Button>
          ) : (
            <Button
              disabled={declare.isPending || text.trim() === ''}
              onClick={() => declare.mutate({ body: { hosts: text } })}
            >
              <IconServerBolt />
              {declare.isPending ? 'Recording…' : 'Add and queue'}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
