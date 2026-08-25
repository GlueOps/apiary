import { Button } from '@/components/ui/button';
import type { AccessRuleView } from '@/sdk/types.gen';

export type Purpose = AccessRuleView['purpose'];
export type Kind = AccessRuleView['kind'];
export type Policy = AccessRuleView['policy'];

/**
 * The three questions the table answers, in the order an account meets them.
 *
 * `question` is phrased as a question on purpose. A rule stored under the wrong
 * purpose is not a weaker rule — it is a rule that never fires on the thing the
 * operator meant to gate, and the backend refuses to guess a default precisely
 * so that mistake cannot be made silently. The UI's job is to make choosing
 * unavoidable and the consequence legible.
 */
export const PURPOSES: {
  value: Purpose;
  label: string;
  question: string;
  blurb: string;
}[] = [
  {
    value: 'authenticate',
    label: 'Signing in',
    question: 'May this identity authenticate at all?',
    blurb:
      'The ban list. A deny here locks the account out of everything, including organizations it already belongs to.',
  },
  {
    value: 'create_org',
    label: 'Founding an organization',
    question: 'May this identity create a new organization?',
    blurb:
      'Where the public-mailbox corpus lands. A deny here stops someone minting a tenant from a throwaway mailbox; it does not stop them signing in or accepting an invitation.',
  },
  {
    value: 'join_org',
    label: 'Accepting an invitation',
    question: 'May this identity join an organization it was invited to?',
    blurb:
      'Decides invitation acceptance only. Someone denied here can still authenticate if an authenticate rule permits it.',
  },
];

export const purposeMeta = (p: Purpose) => PURPOSES.find((x) => x.value === p) ?? PURPOSES[0];

export const KIND_HINT: Record<Kind, string> = {
  email:
    'One address. Matched after lowercasing and stripping plus-addressing, so a rule on ada@example.com also decides ada+ci@example.com.',
  domain:
    'A bare domain. It matches an address whose domain is exactly this, or whose registrable domain is this — so a rule on example.com also decides mail.example.com.',
};

export const errorDetail = (err: unknown, fallback: string) =>
  (err as { detail?: string })?.detail ?? fallback;

/**
 * Allow and deny as one two-state control rather than a button per action.
 *
 * Deliberately not a delete-and-recreate pair. An UPDATE to 'allow' survives
 * every restart while a DELETE is undone by the next boot's seed pass, so a
 * control that made both look equivalent would make the reversible-looking
 * action the one that silently reverts.
 */
export function PolicyChoice({
  value,
  onChange,
  disabled,
  label,
  size = 'xs',
}: {
  value: Policy;
  onChange: (next: Policy) => void;
  disabled?: boolean;
  label: string;
  size?: 'xs' | 'sm';
}) {
  return (
    <div
      role="group"
      aria-label={label}
      className="inline-flex w-fit gap-0.5 rounded-lg border border-border p-0.5"
    >
      <Button
        size={size}
        variant={value === 'allow' ? 'default' : 'ghost'}
        aria-pressed={value === 'allow'}
        disabled={disabled}
        onClick={() => onChange('allow')}
      >
        Allow
      </Button>
      <Button
        size={size}
        variant={value === 'deny' ? 'destructive' : 'ghost'}
        aria-pressed={value === 'deny'}
        disabled={disabled}
        onClick={() => onChange('deny')}
      >
        Deny
      </Button>
    </div>
  );
}
