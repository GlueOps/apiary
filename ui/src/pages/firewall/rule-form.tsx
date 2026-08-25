import { useState } from 'react';
import type { ReactNode } from 'react';

import { IconAlertTriangle } from '@tabler/icons-react';

import { Button } from '@/components/ui/button';
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
import type { RuleBodyWritable, RuleView } from '@/sdk/types.gen';

export type Family = 'inet' | 'ip' | 'ip6';
export type Protocol = 'tcp' | 'udp' | 'icmp' | 'any';
export type RuleAction = 'accept' | 'drop' | 'reject';

/**
 * Only TCP and UDP carry ports. nft DISCARDS a port match on icmp or any while
 * rendering, so "allow icmp from here on 22" is stored and enforced as "allow
 * all icmp from here" — wider than what was written, with no error anywhere.
 * The form has to stop that reaching the API rather than explain it afterwards.
 */
export const carriesPorts = (protocol: string) => protocol === 'tcp' || protocol === 'udp';

const isIpv4 = (host: string) =>
  /^\d{1,3}(\.\d{1,3}){3}$/.test(host) && host.split('.').every((octet) => Number(octet) <= 255);

const isIpv6 = (host: string) => {
  if (!/^[0-9a-f:.]+$/i.test(host)) return false;
  const halves = host.split('::');
  if (halves.length > 2) return false;
  const groups = halves.flatMap((half) => (half === '' ? [] : half.split(':')));
  let seen = 0;
  for (let i = 0; i < groups.length; i += 1) {
    const group = groups[i];
    if (group.includes('.')) {
      // A dotted tail (::ffff:10.0.0.1) is legal only as the last group, and it
      // occupies two of the eight.
      if (i !== groups.length - 1 || !isIpv4(group)) return false;
      seen += 2;
    } else if (/^[0-9a-f]{1,4}$/i.test(group)) {
      seen += 1;
    } else {
      return false;
    }
  }
  return halves.length === 2 ? seen <= 7 : seen === 8;
};

/** The family a CIDR belongs to, or null if it is not a well-formed CIDR. */
export function addressFamilyOf(value: string): 'ip' | 'ip6' | null {
  const [host, prefix, ...extra] = value.trim().split('/');
  if (extra.length > 0 || prefix === undefined || !/^\d{1,3}$/.test(prefix)) return null;
  const bits = Number(prefix);
  if (isIpv4(host)) return bits <= 32 ? 'ip' : null;
  if (isIpv6(host)) return bits <= 128 ? 'ip6' : null;
  return null;
}

/**
 * A mismatched address is the expensive mistake in this form. nft rejects the
 * WHOLE ruleset over one bad line, so a v6 range in a rule declared `ip` does
 * not disable that rule — it takes the host's entire firewall down with it.
 */
export function cidrProblem(value: string, family: Family): string | null {
  const trimmed = value.trim();
  if (trimmed === '') return null;

  const slash = trimmed.indexOf('/');
  if (slash === -1) {
    return 'Needs a prefix length: a single host is 10.0.0.5/32, or 2001:db8::5/128.';
  }

  const found = addressFamilyOf(trimmed);
  if (found === null) {
    return 'Not an address range. Write it as a CIDR, like 10.0.0.0/8 or 2001:db8::/32.';
  }
  if (family === 'ip' && found === 'ip6') {
    return 'This is IPv6 but the rule is declared ip. nft rejects the whole ruleset over one mismatch — every rule on the host stops applying, not just this one.';
  }
  if (family === 'ip6' && found === 'ip') {
    return 'This is IPv4 but the rule is declared ip6. nft rejects the whole ruleset over one mismatch — every rule on the host stops applying, not just this one.';
  }
  return null;
}

/** Both ends of one port range. An empty end means just the start port. */
export function portProblem(start: string, end: string, protocol: Protocol): string | null {
  if (!carriesPorts(protocol)) return null;
  const from = start.trim();
  const to = end.trim();
  if (from === '' && to === '') return null;
  if (from === '') return 'A range needs a start port. Leave both empty to match any port.';

  const valid = (v: string) => /^\d{1,5}$/.test(v) && Number(v) >= 1 && Number(v) <= 65535;
  if (!valid(from) || (to !== '' && !valid(to))) return 'Ports run from 1 to 65535.';
  if (to !== '' && Number(to) < Number(from)) return 'The range ends before it starts.';
  return null;
}

/**
 * The generated RuleView widens every enum to `string`, so a rule read back from
 * the API has to be narrowed before it can be written again. The values come
 * from the same server that validates them, hence the casts rather than a parse.
 */
export function toRuleBody(rule: RuleView): RuleBodyWritable {
  return {
    name: rule.name,
    enabled: rule.enabled,
    family: rule.family as Family,
    table_name: rule.table_name,
    chain_name: rule.chain_name,
    position: rule.position,
    src_cidr: rule.src_cidr,
    dst_cidr: rule.dst_cidr,
    protocol: rule.protocol as Protocol,
    src_port: rule.src_port,
    src_port_end: rule.src_port_end,
    dst_port: rule.dst_port,
    dst_port_end: rule.dst_port_end,
    in_iface: rule.in_iface,
    out_iface: rule.out_iface,
    action: rule.action as RuleAction,
    comment: rule.comment,
  };
}

interface Draft {
  name: string;
  enabled: boolean;
  family: Family;
  table_name: string;
  chain_name: string;
  position: string;
  protocol: Protocol;
  src_cidr: string;
  src_port: string;
  src_port_end: string;
  dst_cidr: string;
  dst_port: string;
  dst_port_end: string;
  in_iface: string;
  out_iface: string;
  action: RuleAction;
  comment: string;
}

// 0 is the API's "unset" for a port, and it is not a port anyone can type.
const asText = (n: number) => (n ? String(n) : '');

function draftFrom(rule: RuleView | null, nextPosition: number, mode: 'new' | 'edit' | 'clone'): Draft {
  if (!rule) {
    return {
      name: '',
      enabled: true,
      family: 'inet',
      table_name: 'filter',
      chain_name: 'input',
      position: String(nextPosition),
      protocol: 'tcp',
      src_cidr: '',
      src_port: '',
      src_port_end: '',
      dst_cidr: '',
      dst_port: '',
      dst_port_end: '',
      in_iface: '',
      out_iface: '',
      action: 'accept',
      comment: '',
    };
  }
  return {
    // A clone keeps everything but its name. Two rules called "ssh from
    // management" are indistinguishable in the list, and the name is what an
    // operator uses to find the one they meant to change later.
    name: mode === 'clone' ? `${rule.name} (copy)` : rule.name,
    enabled: rule.enabled,
    family: rule.family as Family,
    table_name: rule.table_name,
    chain_name: rule.chain_name,
    position: String(rule.position),
    protocol: rule.protocol as Protocol,
    src_cidr: rule.src_cidr,
    src_port: asText(rule.src_port),
    src_port_end: asText(rule.src_port_end),
    dst_cidr: rule.dst_cidr,
    dst_port: asText(rule.dst_port),
    dst_port_end: asText(rule.dst_port_end),
    in_iface: rule.in_iface,
    out_iface: rule.out_iface,
    action: rule.action as RuleAction,
    comment: rule.comment,
  };
}

/**
 * Every field on every write. The API replaces the whole rule rather than
 * patching it, precisely so half an edit — a new port against the old action —
 * can never be what ends up on a host, and a body that omitted a field would
 * hand it back its default instead of the value the operator is looking at.
 */
function toBody(draft: Draft): RuleBodyWritable {
  const port = (value: string) =>
    carriesPorts(draft.protocol) && value.trim() !== '' ? Number(value) : 0;
  return {
    name: draft.name.trim(),
    enabled: draft.enabled,
    family: draft.family,
    table_name: draft.table_name.trim() || 'filter',
    chain_name: draft.chain_name.trim() || 'input',
    position: Number(draft.position),
    protocol: draft.protocol,
    src_cidr: draft.src_cidr.trim(),
    dst_cidr: draft.dst_cidr.trim(),
    src_port: port(draft.src_port),
    src_port_end: port(draft.src_port_end),
    dst_port: port(draft.dst_port),
    dst_port_end: port(draft.dst_port_end),
    in_iface: draft.in_iface.trim(),
    out_iface: draft.out_iface.trim(),
    action: draft.action,
    comment: draft.comment.trim(),
  };
}

const FAMILIES: { value: Family; label: string }[] = [
  { value: 'inet', label: 'inet — IPv4 and IPv6' },
  { value: 'ip', label: 'ip — IPv4 only' },
  { value: 'ip6', label: 'ip6 — IPv6 only' },
];

const PROTOCOLS: Protocol[] = ['tcp', 'udp', 'icmp', 'any'];

const ACTIONS: { value: RuleAction; label: string }[] = [
  { value: 'accept', label: 'accept — let it through' },
  { value: 'drop', label: 'drop — discard silently' },
  { value: 'reject', label: 'reject — refuse and say so' },
];

function Field({
  id,
  label,
  hint,
  error,
  children,
}: {
  id: string;
  label: string;
  hint?: string;
  error?: string | null;
  children: ReactNode;
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      {children}
      {error ? (
        <p className="text-xs text-destructive">{error}</p>
      ) : hint ? (
        <p className="text-xs text-muted-foreground">{hint}</p>
      ) : null}
    </div>
  );
}

/**
 * Create or replace one rule.
 *
 * Mount it with a `key` that changes per subject so the draft starts from the
 * rule being edited; there is no effect syncing props into state.
 */
export function RuleFormDialog({
  rule,
  mode,
  rulesetName,
  nextPosition,
  isPending,
  onSubmit,
  onClose,
}: {
  rule: RuleView | null;
  /**
   * Which of three things this is. `clone` is prefilled from an existing rule
   * but CREATES — distinct from `edit`, which replaces one, and the difference
   * has to be visible: a dialog that looks identical either way is one keystroke
   * from replacing the rule you meant to copy.
   */
  mode: 'new' | 'edit' | 'clone';
  rulesetName: string;
  nextPosition: number;
  isPending: boolean;
  onSubmit: (body: RuleBodyWritable) => void;
  onClose: () => void;
}) {
  const [draft, setDraft] = useState<Draft>(() => draftFrom(rule, nextPosition, mode));
  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((d) => ({ ...d, [key]: value }));

  const ports = carriesPorts(draft.protocol);

  const srcCidrError = cidrProblem(draft.src_cidr, draft.family);
  const dstCidrError = cidrProblem(draft.dst_cidr, draft.family);
  const srcPortError = portProblem(draft.src_port, draft.src_port_end, draft.protocol);
  const dstPortError = portProblem(draft.dst_port, draft.dst_port_end, draft.protocol);
  const positionError = /^\d{1,6}$/.test(draft.position.trim())
    ? null
    : 'A position is a whole number, 0 or more.';
  const nameError =
    draft.name.trim() === '' ? 'A rule needs a name you will recognise later.' : null;

  // Legal under inet, and never what anyone means: one rule cannot match an
  // IPv4 source and an IPv6 destination, so it would sit in the chain matching
  // nothing. Warned rather than blocked — the API accepts it.
  const srcFamily = srcCidrError ? null : addressFamilyOf(draft.src_cidr);
  const dstFamily = dstCidrError ? null : addressFamilyOf(draft.dst_cidr);
  const crossFamily =
    draft.family === 'inet' && srcFamily !== null && dstFamily !== null && srcFamily !== dstFamily;

  const blocked =
    nameError !== null ||
    positionError !== null ||
    srcCidrError !== null ||
    dstCidrError !== null ||
    srcPortError !== null ||
    dstPortError !== null;

  const portHint = ports
    ? 'Leave both empty to match any port. An empty end means just the start port.'
    : `${draft.protocol} has no ports — nft would discard them, storing a wider rule than you wrote.`;

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {mode === 'edit'
              ? `Edit ${rule?.name}`
              : mode === 'clone'
                ? `Copy of ${rule?.name}`
                : `New rule in ${rulesetName}`}
          </DialogTitle>
          {mode === 'clone' ? (
            <DialogDescription>
              Everything is copied except the name, including the position — a rule cloned from
              above a drop has to stay above it, and moving the copy to the end of the chain would
              invert what it does. Two rules on one position render oldest first, so this copy
              lands immediately after its source. Change the source address, then save.
            </DialogDescription>
          ) : null}
          <DialogDescription>
            {rule
              ? 'Saving replaces the whole rule. There is no partial edit, so what you see here is exactly what every bound host will enforce.'
              : 'This rule applies on every host in every fleet bound to this ruleset, in position order.'}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <Field id="rule-name" label="Name" error={nameError}>
              <Input
                id="rule-name"
                className="w-full"
                value={draft.name}
                onChange={(e) => set('name', e.target.value)}
                placeholder="ssh from the management range"
              />
            </Field>
          </div>

          <Field
            id="rule-position"
            label="Position"
            hint="Lowest first, and the first match wins. Spaced by 10 so a rule can be slotted in later without renumbering."
            error={positionError}
          >
            <Input
              id="rule-position"
              inputMode="numeric"
              className="w-full"
              value={draft.position}
              onChange={(e) => set('position', e.target.value)}
            />
          </Field>

          <Field id="rule-action" label="Action">
            <Select
              value={draft.action}
              onValueChange={(value) => set('action', value as RuleAction)}
            >
              <SelectTrigger id="rule-action" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {ACTIONS.map((a) => (
                  <SelectItem key={a.value} value={a.value}>
                    {a.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>

          <Field
            id="rule-family"
            label="Family"
            hint="Every address in this rule must belong to the family."
          >
            <Select value={draft.family} onValueChange={(value) => set('family', value as Family)}>
              <SelectTrigger id="rule-family" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {FAMILIES.map((f) => (
                  <SelectItem key={f.value} value={f.value}>
                    {f.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>

          <Field id="rule-protocol" label="Protocol">
            <Select
              value={draft.protocol}
              onValueChange={(value) => {
                const protocol = value as Protocol;
                // Clearing rather than hiding: a port left in state would be
                // invisible on screen and still describe a rule the operator
                // did not write.
                setDraft((d) =>
                  carriesPorts(protocol)
                    ? { ...d, protocol }
                    : {
                        ...d,
                        protocol,
                        src_port: '',
                        src_port_end: '',
                        dst_port: '',
                        dst_port_end: '',
                      },
                );
              }}
            >
              <SelectTrigger id="rule-protocol" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PROTOCOLS.map((p) => (
                  <SelectItem key={p} value={p}>
                    {p}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>

          <Field
            id="rule-src-cidr"
            label="Source range"
            hint="Empty matches any source."
            error={srcCidrError}
          >
            <Input
              id="rule-src-cidr"
              className="w-full font-mono"
              value={draft.src_cidr}
              onChange={(e) => set('src_cidr', e.target.value)}
              placeholder={draft.family === 'ip6' ? '2001:db8::/32' : '10.0.0.0/8'}
            />
          </Field>

          <Field
            id="rule-dst-cidr"
            label="Destination range"
            hint="Empty matches any destination."
            error={dstCidrError}
          >
            <Input
              id="rule-dst-cidr"
              className="w-full font-mono"
              value={draft.dst_cidr}
              onChange={(e) => set('dst_cidr', e.target.value)}
              placeholder={draft.family === 'ip6' ? '::/0' : '0.0.0.0/0'}
            />
          </Field>

          {crossFamily ? (
            <p className="flex items-start gap-2 text-xs text-amber-500 sm:col-span-2">
              <IconAlertTriangle className="mt-px size-4 shrink-0" />
              <span>
                The source is IPv{srcFamily === 'ip' ? '4' : '6'} and the destination is IPv
                {dstFamily === 'ip' ? '4' : '6'}. A single rule cannot match both, so this one would
                never fire.
              </span>
            </p>
          ) : null}

          <Field id="rule-src-port" label="Source ports" hint={portHint} error={srcPortError}>
            <div className="flex items-center gap-2">
              <Input
                id="rule-src-port"
                inputMode="numeric"
                disabled={!ports}
                className="w-full"
                value={draft.src_port}
                onChange={(e) => set('src_port', e.target.value)}
                placeholder="from"
              />
              <span className="text-muted-foreground">–</span>
              <Input
                aria-label="Source port range end"
                inputMode="numeric"
                disabled={!ports}
                className="w-full"
                value={draft.src_port_end}
                onChange={(e) => set('src_port_end', e.target.value)}
                placeholder="to"
              />
            </div>
          </Field>

          <Field id="rule-dst-port" label="Destination ports" hint={portHint} error={dstPortError}>
            <div className="flex items-center gap-2">
              <Input
                id="rule-dst-port"
                inputMode="numeric"
                disabled={!ports}
                className="w-full"
                value={draft.dst_port}
                onChange={(e) => set('dst_port', e.target.value)}
                placeholder="from"
              />
              <span className="text-muted-foreground">–</span>
              <Input
                aria-label="Destination port range end"
                inputMode="numeric"
                disabled={!ports}
                className="w-full"
                value={draft.dst_port_end}
                onChange={(e) => set('dst_port_end', e.target.value)}
                placeholder="to"
              />
            </div>
          </Field>

          <Field id="rule-in-iface" label="Inbound interface" hint="Empty matches any interface.">
            <Input
              id="rule-in-iface"
              className="w-full font-mono"
              value={draft.in_iface}
              onChange={(e) => set('in_iface', e.target.value)}
              placeholder="vmbr0"
            />
          </Field>

          <Field id="rule-out-iface" label="Outbound interface" hint="Empty matches any interface.">
            <Input
              id="rule-out-iface"
              className="w-full font-mono"
              value={draft.out_iface}
              onChange={(e) => set('out_iface', e.target.value)}
              placeholder="vmbr0"
            />
          </Field>

          <Field id="rule-table" label="Table">
            <Input
              id="rule-table"
              className="w-full font-mono"
              value={draft.table_name}
              onChange={(e) => set('table_name', e.target.value)}
              placeholder="filter"
            />
          </Field>

          <Field
            id="rule-chain"
            label="Chain"
            hint="input is traffic to the host; forward is traffic through it."
          >
            <Input
              id="rule-chain"
              className="w-full font-mono"
              value={draft.chain_name}
              onChange={(e) => set('chain_name', e.target.value)}
              placeholder="input"
            />
          </Field>

          <div className="sm:col-span-2">
            <Field
              id="rule-comment"
              label="Comment"
              hint="Rendered into the nft ruleset, so it is what the next person reads on the host."
            >
              <Input
                id="rule-comment"
                className="w-full"
                value={draft.comment}
                onChange={(e) => set('comment', e.target.value)}
                placeholder="ticket OPS-1421"
              />
            </Field>
          </div>

          <label className="flex items-start gap-2 text-sm sm:col-span-2">
            <input
              type="checkbox"
              className="mt-1"
              checked={draft.enabled}
              onChange={(e) => set('enabled', e.target.checked)}
            />
            <span>
              Enabled
              <span className="block text-xs text-muted-foreground">
                A disabled rule keeps its position but is not rendered onto any host, so turning it
                back on restores it to the same place in the order.
              </span>
            </span>
          </label>
        </div>

        <DialogFooter>
          <Button type="button" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button disabled={blocked || isPending} onClick={() => onSubmit(toBody(draft))}>
            {isPending ? 'Saving…' : mode === 'edit' ? 'Replace rule' : 'Add rule'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
