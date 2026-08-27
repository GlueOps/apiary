import { useState } from 'react';

import { IconAlertTriangle, IconLock, IconPlus, IconTrash } from '@tabler/icons-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Hint } from '@/pages/fleet/drain.tsx';
import {
  firewallSetElementsAddMutation,
  firewallSetElementsDeleteMutation,
  firewallSetsCreateMutation,
  firewallSetsDeleteMutation,
  firewallSetsListOptions,
  firewallSetsListQueryKey,
} from '@/sdk/@tanstack/react-query.gen';
import type { SetView } from '@/sdk/types.gen';

/** The API's problem detail, or a fallback. Mirrors rulesets.tsx. */
const detail = (e: unknown, fallback: string) => (e as { detail?: string })?.detail ?? fallback;

/**
 * Named address lists.
 *
 * The arithmetic is the point. Four management addresses across two ports is TWO
 * rules and ONE set; written as per-rule literals it is eight, and every address
 * added to the whitelist adds another two. A set is also the only thing that can
 * be edited once and take effect everywhere it is referenced.
 *
 * This page did not exist until 2026-08-27, and its absence had a consequence
 * beyond convenience: onboarding REFUSES to install the fleet key without a
 * management set to derive its from= restriction from, so a set that could only
 * be created by writing SQL meant automated onboarding could not be completed by
 * an operator at all.
 */

/** An address family, never inet — see the family select below. */
const FAMILIES = [
  { value: 'ip', label: 'IPv4', hint: 'ipv4_addr' },
  { value: 'ip6', label: 'IPv6', hint: 'ipv6_addr' },
];

const PURPOSES = [
  {
    value: 'general',
    label: 'General',
    hint: 'An ordinary list for rules to reference.',
  },
  {
    value: 'management',
    label: 'Management',
    hint: 'Where the control plane and operators connect FROM. Onboarding derives the from= restriction on the fleet SSH key from these, so widening one widens who can use that key.',
  },
];

export function FirewallSets() {
  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery(firewallSetsListOptions());
  const sets = data?.sets ?? [];

  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [family, setFamily] = useState('ip');
  const [purpose, setPurpose] = useState('general');
  const [description, setDescription] = useState('');

  const [addingTo, setAddingTo] = useState<SetView | null>(null);
  const [value, setValue] = useState('');
  const [label, setLabel] = useState('');

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: firewallSetsListQueryKey() });
  };

  const create = useMutation({
    ...firewallSetsCreateMutation(),
    onSuccess: () => {
      toast.success('Set created');
      setCreating(false);
      setName('');
      setDescription('');
      refresh();
    },
    onError: (e) => toast.error(detail(e, 'Could not create that set')),
  });

  const remove = useMutation({
    ...firewallSetsDeleteMutation(),
    onSuccess: () => {
      toast.success('Set deleted');
      refresh();
    },
    // A set a rule still references cannot be deleted — nft rejects a ruleset
    // naming a set it has never heard of, and it rejects the WHOLE file, so the
    // host would end up with no firewall rather than one missing a line.
    onError: (e) => toast.error(detail(e, 'Could not delete that set')),
  });

  const addElement = useMutation({
    ...firewallSetElementsAddMutation(),
    onSuccess: () => {
      toast.success('Address added');
      setValue('');
      setLabel('');
      refresh();
    },
    onError: (e) => toast.error(detail(e, 'Could not add that address')),
  });

  const removeElement = useMutation({
    ...firewallSetElementsDeleteMutation(),
    onSuccess: () => {
      toast.success('Address removed');
      refresh();
    },
    onError: (e) => toast.error(detail(e, 'Could not remove that address')),
  });

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div>
            <CardTitle>Address sets</CardTitle>
            <CardDescription>
              A named list a rule points at instead of repeating an address. Four addresses across
              two ports is two rules and one set — as literals it is eight, and each address added
              makes it two more.
            </CardDescription>
          </div>
          <Button onClick={() => setCreating(true)}>
            <IconPlus className="size-4" />
            New set
          </Button>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : sets.length === 0 ? (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">No sets yet.</p>
              <Hint>
                Onboarding needs a set whose purpose is <strong>management</strong> before it will
                install the fleet SSH key — it derives the key&apos;s <code>from=</code> restriction
                from those addresses, and an empty list would permit nothing.
              </Hint>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Family</TableHead>
                  <TableHead>Purpose</TableHead>
                  <TableHead>Addresses</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sets.map((s) => (
                  <TableRow key={s.id}>
                    <TableCell>
                      <div className="font-medium">{s.name}</div>
                      {s.description ? (
                        <div className="text-xs text-muted-foreground">{s.description}</div>
                      ) : null}
                    </TableCell>
                    <TableCell>
                      <code className="text-xs">
                        {s.family === 'ip6' ? 'ipv6_addr' : 'ipv4_addr'}
                      </code>
                    </TableCell>
                    <TableCell>
                      {s.purpose === 'management' ? (
                        <Badge variant="outline">management</Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">{s.purpose}</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {(s.elements ?? []).length === 0 ? (
                          <span className="inline-flex items-center gap-1 text-xs text-amber-600">
                            <IconAlertTriangle className="size-3.5" />
                            empty — matches nothing
                          </span>
                        ) : (
                          (s.elements ?? []).map((e) => (
                            <Badge
                              key={e.id}
                              variant="secondary"
                              className="font-mono text-xs"
                              title={e.comment || undefined}
                            >
                              {e.value}
                              {s.managed ? null : (
                                <button
                                  type="button"
                                  aria-label={`Remove ${e.value}`}
                                  className="ml-1 opacity-60 hover:opacity-100"
                                  onClick={() =>
                                    removeElement.mutate({
                                      path: { id: s.id, element_id: e.id },
                                    })
                                  }
                                >
                                  ×
                                </button>
                              )}
                            </Badge>
                          ))
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      {s.managed ? (
                        /* Refused rather than silently reverted: these contents
                           are maintained by the control plane, and an edit that
                           disappears at the next convergence is worse than one
                           that is not offered. */
                        <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
                          <IconLock className="size-3.5" />
                          control plane
                        </span>
                      ) : (
                        <div className="flex justify-end gap-2">
                          <Button variant="outline" size="sm" onClick={() => setAddingTo(s)}>
                            <IconPlus className="size-4" />
                            Address
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={remove.isPending}
                            onClick={() => remove.mutate({ path: { id: s.id } })}
                          >
                            <IconTrash className="size-4" />
                          </Button>
                        </div>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={creating} onOpenChange={(o) => !o && setCreating(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New address set</DialogTitle>
            <DialogDescription>
              A list rules can reference. Its family is fixed once created.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="set-name">Name</Label>
            <Input
              id="set-name"
              value={name}
              placeholder="allowed_ips"
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="set-family">Address family</Label>
            <Select value={family} onValueChange={(v) => setFamily(v ?? 'ip')}>
              <SelectTrigger id="set-family">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {FAMILIES.map((f) => (
                  <SelectItem key={f.value} value={f.value}>
                    {f.label} — {f.hint}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Hint>
              A set holds one address family and cannot be <code>inet</code>. The rule referencing
              it stays <code>inet</code>; this is what the addresses in the list are. It cannot be
              changed later — the family decides how every element is interpreted, and rules are
              validated against the family they were written for.
            </Hint>
          </div>
          <div className="space-y-2">
            <Label htmlFor="set-purpose">Purpose</Label>
            <Select value={purpose} onValueChange={(v) => setPurpose(v ?? 'general')}>
              <SelectTrigger id="set-purpose">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PURPOSES.map((p) => (
                  <SelectItem key={p.value} value={p.value}>
                    {p.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Hint>{PURPOSES.find((p) => p.value === purpose)?.hint}</Hint>
          </div>
          <div className="space-y-2">
            <Label htmlFor="set-desc">Description</Label>
            <Input
              id="set-desc"
              value={description}
              placeholder="Who these addresses belong to"
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreating(false)}>
              Cancel
            </Button>
            <Button
              disabled={!name.trim() || create.isPending}
              onClick={() =>
                create.mutate({
                  body: {
                    name: name.trim(),
                    family: family as 'ip' | 'ip6',
                    purpose: purpose as 'general' | 'management' | 'peers',
                    description: description.trim(),
                  },
                })
              }
            >
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={addingTo !== null} onOpenChange={(o) => !o && setAddingTo(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add an address to {addingTo?.name}</DialogTitle>
            <DialogDescription>
              An address or a prefix. A bare address is treated as a single host.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="el-value">Address</Label>
            <Input
              id="el-value"
              value={value}
              placeholder={addingTo?.family === 'ip6' ? '2001:db8::1' : '192.0.2.10'}
              onChange={(e) => setValue(e.target.value)}
            />
            <Hint>
              Must be {addingTo?.family === 'ip6' ? 'IPv6' : 'IPv4'} to match this set. A mismatch
              renders a declaration nft rejects — and nft rejects the whole ruleset over one bad
              line, taking the host&apos;s entire firewall with it.
            </Hint>
          </div>
          <div className="space-y-2">
            <Label htmlFor="el-label">Label</Label>
            <Input
              id="el-label"
              value={label}
              placeholder="office"
              onChange={(e) => setLabel(e.target.value)}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAddingTo(null)}>
              Cancel
            </Button>
            <Button
              disabled={!value.trim() || addElement.isPending}
              onClick={() => {
                if (!addingTo) return;
                addElement.mutate({
                  path: { id: addingTo.id },
                  body: { value: value.trim(), label: label.trim() },
                });
                setAddingTo(null);
              }}
            >
              Add
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
