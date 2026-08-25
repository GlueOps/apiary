import { useState } from 'react';

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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { platformAccessRulesCreateMutation } from '@/sdk/@tanstack/react-query.gen';

import {
  KIND_HINT,
  type Kind,
  PURPOSES,
  type Policy,
  PolicyChoice,
  type Purpose,
  errorDetail,
  purposeMeta,
} from './access-rules-shared';

export function CreateAccessRuleDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => void;
}) {
  // Null, not a first option. The API refuses a rule without a purpose because a
  // defaulted purpose produces a rule that reads as active policy while deciding
  // a question nobody asked; a pre-selected dropdown would hand that default
  // back through the UI.
  const [purpose, setPurpose] = useState<Purpose | null>(null);
  const [kind, setKind] = useState<Kind>('domain');
  const [value, setValue] = useState('');
  const [policy, setPolicy] = useState<Policy>('deny');
  const [reason, setReason] = useState('');

  const reset = () => {
    setPurpose(null);
    setKind('domain');
    setValue('');
    setPolicy('deny');
    setReason('');
  };

  const create = useMutation({
    ...platformAccessRulesCreateMutation(),
    onSuccess: (rule) => {
      toast.success(`${rule.policy === 'allow' ? 'Allow' : 'Deny'} rule added for ${rule.value}`);
      reset();
      onCreated();
      onOpenChange(false);
    },
    onError: (e) =>
      toast.error(
        errorDetail(
          e,
          'Could not add that rule. A second rule on the same kind, value and purpose is refused — precedence between two rules answering the same question would depend on row order.',
        ),
      ),
  });

  const chosen = purpose ? purposeMeta(purpose) : null;
  const trimmed = value.trim();
  const ready = purpose !== null && trimmed !== '';

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) reset();
        onOpenChange(next);
      }}
    >
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Add an access rule</DialogTitle>
          <DialogDescription>
            The rule takes effect on the next request that asks its question — there is no publish
            step and no restart. Kind, value and purpose are the rule&rsquo;s identity: changing any
            of them later means deleting this rule and writing another.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="rule-purpose">Which question does this rule answer?</Label>
            <Select value={purpose} onValueChange={(v) => setPurpose(v as Purpose | null)}>
              <SelectTrigger id="rule-purpose" className="w-full">
                <SelectValue placeholder="Choose a purpose — there is no default" />
              </SelectTrigger>
              <SelectContent>
                {PURPOSES.map((p) => (
                  <SelectItem key={p.value} value={p.value}>
                    {p.label} — {p.question}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              {chosen
                ? chosen.blurb
                : 'Required. A rule decides only the purpose it is stored under — an org-creation rule can never block a login, and vice versa.'}
            </p>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="rule-kind">Matches</Label>
              <Select value={kind} onValueChange={(v) => setKind(v as Kind)}>
                <SelectTrigger id="rule-kind" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="domain">A domain</SelectItem>
                  <SelectItem value="email">One address</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="rule-value">{kind === 'email' ? 'Address' : 'Domain'}</Label>
              <Input
                id="rule-value"
                value={value}
                autoComplete="off"
                spellCheck={false}
                placeholder={kind === 'email' ? 'ada@example.com' : 'example.com'}
                onChange={(e) => setValue(e.target.value)}
              />
            </div>
          </div>
          <p className="text-xs text-muted-foreground">{KIND_HINT[kind]}</p>

          <div className="grid gap-2">
            <Label>Policy</Label>
            <PolicyChoice
              size="sm"
              value={policy}
              onChange={setPolicy}
              label="Policy for the new rule"
            />
            <p className="text-xs text-muted-foreground">
              {policy === 'allow'
                ? 'Allow is the only value that permits. It also beats a broader deny on the same question — an allow on one address survives a deny on its whole domain.'
                : 'Anything that is not exactly “allow” denies, so a deny is also what an unrecognised value would do.'}
            </p>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="rule-reason">Reason (optional)</Label>
            <Input
              id="rule-reason"
              value={reason}
              placeholder="Why this exists, for whoever reads the row in a year"
              onChange={(e) => setReason(e.target.value)}
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            disabled={!ready || create.isPending}
            onClick={() => {
              if (!purpose) return;
              create.mutate({
                body: {
                  purpose,
                  kind,
                  value: trimmed,
                  policy,
                  ...(reason.trim() === '' ? {} : { reason: reason.trim() }),
                },
              });
            }}
          >
            {create.isPending ? 'Adding…' : 'Add rule'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
