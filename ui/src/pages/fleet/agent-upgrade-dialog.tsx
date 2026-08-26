import { useState } from 'react';

import { IconArrowUpCircle } from '@tabler/icons-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
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
import {
  fleetAgentUpgradeMutation,
  fleetBeesListQueryKey,
  fleetHivesListOptions,
} from '@/sdk/@tanstack/react-query.gen';
import type { UpgradeCandidateView } from '@/sdk/types.gen';

const EVERY_FLEET = '__all__';

export type AgentUpgradeDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

/**
 * Rolling every agent in a fleet onto the version this control plane serves.
 *
 * The hosts have no route to the internet, so the control plane is the only
 * place they can fetch from — which is what makes a fleet-wide upgrade a control
 * plane operation rather than a configuration-management one.
 *
 * There is no version to choose. These binaries ship with the control plane, so
 * its own version is the only thing it could honestly move an agent to, and
 * offering a field would mean advertising versions it cannot serve.
 */
export function AgentUpgradeDialog({ open, onOpenChange }: AgentUpgradeDialogProps) {
  const qc = useQueryClient();
  const [scope, setScope] = useState<string>(EVERY_FLEET);
  const [results, setResults] = useState<{
    version: string;
    agents: UpgradeCandidateView[];
  } | null>(null);

  const hives = useQuery(fleetHivesListOptions());
  const fleets = hives.data?.hives ?? [];

  const upgrade = useMutation({
    ...fleetAgentUpgradeMutation(),
    onSuccess: (res) => {
      // ?? [], because the generated type makes the list nullable. An absent
      // list is "no agents in scope", which renders as an empty result rather
      // than as a crash.
      setResults({ version: res.version, agents: res.agents ?? [] });
      toast.success(
        res.queued === 0
          ? 'Nothing to upgrade — every agent in scope is already current or unavailable'
          : `${res.queued} agent${res.queued === 1 ? '' : 's'} queued`,
      );
      void qc.invalidateQueries({ queryKey: fleetBeesListQueryKey() });
    },
    onError: (e) =>
      toast.error((e as { detail?: string })?.detail ?? 'Could not queue the upgrade'),
  });

  const close = (next: boolean) => {
    if (!next) setResults(null);
    onOpenChange(next);
  };

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Upgrade agents</DialogTitle>
          <DialogDescription>
            Each host downloads the agent this control plane serves, checks it against the
            advertised digest, proves it runs, keeps the previous one, and restarts.
          </DialogDescription>
        </DialogHeader>

        {results ? (
          <div className="space-y-3">
            <p className="text-sm">
              Target version <code className="rounded bg-muted px-1 py-0.5">{results.version}</code>
            </p>
            <div className="max-h-80 space-y-1 overflow-auto rounded border p-2">
              {results.agents.map((a) => (
                <div key={a.bee_id} className="flex items-baseline justify-between gap-3 text-sm">
                  <code className="font-mono text-xs">{a.bee_id.slice(0, 8)}</code>
                  <span className="text-xs text-muted-foreground">
                    {a.agent_version || 'unknown version'}
                  </span>
                  {a.queued ? (
                    <Badge className="bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">
                      queued
                    </Badge>
                  ) : (
                    <span className="text-xs text-muted-foreground">{a.skipped}</span>
                  )}
                </div>
              ))}
            </div>
            <p className="text-xs text-muted-foreground">
              Each agent declares a short unavailable window before it restarts, so the silence is
              expected rather than read as a failure. A host that does not come back keeps its
              previous binary at <code>.previous</code>.
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="upgrade-scope">Scope</Label>
              <Select value={scope} onValueChange={(v) => setScope(String(v))}>
                <SelectTrigger id="upgrade-scope" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={EVERY_FLEET}>
                    Every agent — including hosts in no cluster
                  </SelectItem>
                  {fleets.map((f) => (
                    <SelectItem key={f.id} value={f.id}>
                      {f.name} · {f.node_count} {f.node_count === 1 ? 'host' : 'hosts'}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                One cluster at a time is the cautious order. Hosts in no cluster are covered only by
                the first option, and they are the ones most easily forgotten.
              </p>
            </div>

            <div className="space-y-1 rounded-lg border p-3">
              <p className="text-sm font-medium">What gets skipped</p>
              <p className="text-xs text-muted-foreground">
                Agents already on this version, agents that are offline, and agents already running
                another task. Each one says which. An offline agent is deliberately not queued for
                later — it would collect the task whenever it returned, which is often in the middle
                of whatever brought it back.
              </p>
            </div>
          </div>
        )}

        <DialogFooter>
          {results ? (
            <Button onClick={() => close(false)}>Done</Button>
          ) : (
            <Button
              disabled={upgrade.isPending}
              onClick={() =>
                upgrade.mutate({
                  body: scope === EVERY_FLEET ? {} : { hive_id: scope },
                })
              }
            >
              <IconArrowUpCircle />
              {upgrade.isPending ? 'Queueing…' : 'Queue upgrade'}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
