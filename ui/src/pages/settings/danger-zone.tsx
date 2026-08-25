import { useState } from 'react';

import { IconAlertTriangle, IconBuildingOff, IconUserX } from '@tabler/icons-react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Link, useNavigate } from 'react-router-dom';
import { toast } from 'sonner';

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
import { useAuth } from '@/hooks/use-auth';
import {
  meDeleteAccountMutation,
  meGetOptions,
  meOrgsListOptions,
  meSoleOwnershipsOptions,
  orgDeleteMutation,
} from '@/sdk/@tanstack/react-query.gen';

const errorDetail = (err: unknown, fallback: string) =>
  (err as { detail?: string })?.detail ?? fallback;

/**
 * A confirmation that cannot be cleared by muscle memory: the operator has to
 * reproduce the name of the thing being destroyed. Both endpoints behind this
 * page demand the same phrase in their request body, so the control mirrors the
 * server rather than inventing a softer gate the API would not honour.
 */
function ConfirmByTyping({
  open,
  onOpenChange,
  title,
  description,
  consequences,
  phrase,
  phraseLabel,
  inputId,
  confirmLabel,
  pendingLabel,
  pending,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  consequences: string[];
  phrase: string;
  phraseLabel: string;
  inputId: string;
  confirmLabel: string;
  pendingLabel: string;
  pending: boolean;
  onConfirm: (phrase: string) => void;
}) {
  const [typed, setTyped] = useState('');

  // Surrounding whitespace is stripped on both the comparison and the value
  // sent, so a phrase pasted out of an email cannot arm the button here and
  // then be refused by the server's exact-match check.
  const value = typed.trim();
  const matches = value === phrase && phrase.length > 0;

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) setTyped('');
        onOpenChange(next);
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-destructive">{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <ul className="space-y-1 text-sm text-muted-foreground">
          {consequences.map((line) => (
            <li key={line} className="flex gap-2">
              <span aria-hidden="true" className="text-destructive">
                &bull;
              </span>
              <span>{line}</span>
            </li>
          ))}
        </ul>

        <div className="space-y-2">
          <Label htmlFor={inputId}>{phraseLabel}</Label>
          <Input
            id={inputId}
            value={typed}
            autoComplete="off"
            spellCheck={false}
            placeholder={phrase}
            onChange={(e) => setTyped(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            Type <span className="font-medium text-foreground">{phrase}</span> exactly to enable the
            button.
          </p>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            disabled={!matches || pending}
            onClick={() => onConfirm(value)}
          >
            {pending ? pendingLabel : confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export const DangerZone = () => {
  const navigate = useNavigate();
  const { clear } = useAuth();

  const [accountOpen, setAccountOpen] = useState(false);
  const [orgOpen, setOrgOpen] = useState(false);

  const { data: me, isLoading: meLoading } = useQuery({ ...meGetOptions() });
  const { data: myOrgs } = useQuery({ ...meOrgsListOptions() });

  // Asked before anything is offered, not after a refusal: an account that is
  // the last owner of a tenant cannot be deleted, and the useful answer is
  // WHICH tenants, so the operator can go and fix it.
  const {
    data: ownerships,
    isLoading: ownershipsLoading,
    isError: ownershipsFailed,
    error: ownershipsError,
  } = useQuery({ ...meSoleOwnershipsOptions() });

  const email = me?.email ?? '';
  const orgName = me?.organization?.name ?? '';

  // The ORG role lives on the membership, not on me.role (that one is the
  // platform role). Deleting an organization is owner-only, so the button is
  // gated on the membership for the org this session is scoped to.
  const membership = (myOrgs?.organizations ?? []).find((o) => o.id === me?.organization?.id);
  const orgRole = membership?.role;
  const knownNonOwner = orgRole !== undefined && orgRole !== 'owner';

  const blockedOrgs = ownerships?.organizations ?? [];
  const blocked = ownerships?.blocks_account_deletion === true;

  const deleteAccount = useMutation({
    ...meDeleteAccountMutation(),
    onSuccess: () => {
      setAccountOpen(false);
      // The server has already revoked every session for this user, so the
      // usual logout() call would 401. Drop the local session and leave the
      // protected routes before a refetch fires against a dead token.
      clear();
      navigate('/login', { replace: true });
      toast.success('Your account has been deleted.');
    },
    onError: (err) => toast.error(errorDetail(err, 'Could not delete your account')),
  });

  const deleteOrg = useMutation({
    ...orgDeleteMutation(),
    onSuccess: (res) => {
      setOrgOpen(false);
      clear();
      navigate('/login', { replace: true });
      toast.success(res?.note ?? `${orgName} has been taken out of service.`);
    },
    onError: (err) => toast.error(errorDetail(err, 'Could not delete this organization')),
  });

  return (
    <div className="w-full space-y-6 p-6">
      <div>
        <h1 className="flex items-center gap-2 font-heading text-2xl text-destructive">
          <IconAlertTriangle className="size-6" /> Danger zone
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Two actions that take effect the moment you confirm them, and neither is undoable from
          this screen. Both refuse an impersonated session: support acting as you cannot delete what
          is yours.
        </p>
      </div>

      <div className="rounded-md border border-border bg-muted/40 p-4 text-sm text-muted-foreground">
        <span className="font-medium text-foreground">Nothing is erased here.</span> Data is
        retained encrypted until an operator purges it — the organization record holds the only key
        that can decrypt its database, so deleting it retires the tenant rather than destroying it.
        Your user record is kept for the same reason: audit events, fleet tasks and invitations name
        the actor, and a trail that vanishes with the account is one the audited person can erase.
      </div>

      <Card className="border-destructive/40 bg-destructive/5">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <IconUserX className="size-5 text-destructive" /> Delete your account
          </CardTitle>
          <CardDescription>
            {meLoading
              ? 'Loading your account…'
              : `Removes ${email} from every organization it belongs to, revokes every session, and deletes every authentication factor — password, passkeys and TOTP.`}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {meLoading || ownershipsLoading ? (
            <p className="text-sm text-muted-foreground">
              Checking which organizations you solely own…
            </p>
          ) : ownershipsFailed ? (
            // Rendering the button on a failed check would offer an action whose
            // one precondition is unknown, so it stays shut until the check answers.
            <div className="space-y-2 rounded-md border border-border bg-background p-4">
              <p className="text-sm font-medium">Sole-ownership check could not be loaded</p>
              <p className="text-sm text-muted-foreground">
                {errorDetail(ownershipsError, 'The server did not answer.')}
              </p>
              <p className="text-sm text-muted-foreground">
                Account deletion stays disabled until it does: deleting while you are the last owner
                of an organization would leave a tenant nobody can administer.
              </p>
            </div>
          ) : blocked ? (
            <div className="space-y-3 rounded-md border border-destructive/40 bg-background p-4">
              <p className="text-sm font-medium">
                You are the only owner of{' '}
                {blockedOrgs.length === 1 ? 'this organization' : 'these organizations'}
              </p>
              <ul className="space-y-1 text-sm">
                {blockedOrgs.map((org) => (
                  <li key={org.id} className="font-medium">
                    {org.name}
                  </li>
                ))}
              </ul>
              <p className="text-sm text-muted-foreground">
                Deleting your account now would leave{' '}
                {blockedOrgs.length === 1 ? 'a tenant' : 'tenants'} nobody can administer, so it is
                refused. Do one of two things first: promote another member to owner on the{' '}
                <Link to="/settings/team" className="font-medium text-foreground underline">
                  Team
                </Link>{' '}
                page, or delete the organization below. Once neither is true, this button unlocks.
              </p>
              <Button variant="destructive" disabled>
                Delete my account
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground">
                You are not the last owner of any organization, so this account can be deleted.
              </p>
              <Button variant="destructive" onClick={() => setAccountOpen(true)}>
                Delete my account
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="border-destructive/40 bg-destructive/5">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <IconBuildingOff className="size-5 text-destructive" /> Delete{' '}
            {orgName || 'this organization'}
          </CardTitle>
          <CardDescription>
            This takes an entire tenant out of service, for everyone in it — not just for you.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <ul className="space-y-1 text-sm text-muted-foreground">
            <li className="flex gap-2">
              <span aria-hidden="true" className="text-destructive">
                &bull;
              </span>
              <span>Its database stops resolving immediately; nothing in it can be read.</span>
            </li>
            <li className="flex gap-2">
              <span aria-hidden="true" className="text-destructive">
                &bull;
              </span>
              <span>
                Every session scoped to it is revoked — every member is signed out, including you.
              </span>
            </li>
            <li className="flex gap-2">
              <span aria-hidden="true" className="text-destructive">
                &bull;
              </span>
              <span>
                Its agents can no longer authenticate, so the fleet stops reporting and stops taking
                tasks.
              </span>
            </li>
          </ul>

          {meLoading ? (
            <p className="text-sm text-muted-foreground">Loading your organization…</p>
          ) : knownNonOwner ? (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">
                Only an owner can delete an organization, and you are {orgRole} here. Ask an owner
                of {orgName} to do it.
              </p>
              <Button variant="destructive" disabled>
                Delete this organization
              </Button>
            </div>
          ) : (
            <Button variant="destructive" onClick={() => setOrgOpen(true)}>
              Delete this organization
            </Button>
          )}
        </CardContent>
      </Card>

      <ConfirmByTyping
        open={accountOpen}
        onOpenChange={setAccountOpen}
        title="Delete your account"
        description={`This deletes ${email}. It cannot be undone from here.`}
        consequences={[
          'Every session is revoked, on every device, right away.',
          'Your password, passkeys and TOTP factor are removed — there is nothing left to sign in with.',
          'Your memberships end; organizations you belong to keep running without you.',
        ]}
        phrase={email}
        phraseLabel="Confirm your email address"
        inputId="confirm-account-email"
        confirmLabel="Delete my account"
        pendingLabel="Deleting…"
        pending={deleteAccount.isPending}
        onConfirm={(confirmEmail) =>
          deleteAccount.mutate({ body: { confirm_email: confirmEmail } })
        }
      />

      <ConfirmByTyping
        open={orgOpen}
        onOpenChange={setOrgOpen}
        title={`Delete ${orgName}`}
        description={`This takes ${orgName} out of service for every one of its members and agents.`}
        consequences={[
          'The tenant database stops resolving immediately.',
          'Every session scoped to this organization is revoked, including the one you are using now.',
          'Its agents can no longer authenticate.',
          'Data stays encrypted at rest until an operator purges it; this call does not destroy it.',
        ]}
        phrase={orgName}
        phraseLabel="Confirm the organization name"
        inputId="confirm-org-name"
        confirmLabel="Delete this organization"
        pendingLabel="Deleting…"
        pending={deleteOrg.isPending}
        onConfirm={(confirmName) => deleteOrg.mutate({ body: { confirm_name: confirmName } })}
      />
    </div>
  );
};
