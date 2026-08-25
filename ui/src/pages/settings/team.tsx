import React, { useState } from 'react';

import { IconTrash, IconUserPlus } from '@tabler/icons-react';
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
import {
  orgInvitationsCreateMutation,
  orgInvitationsListOptions,
  orgInvitationsListQueryKey,
  orgInvitationsRevokeMutation,
  orgMembersListOptions,
  orgMembersListQueryKey,
  orgMembersRemoveMutation,
  orgMembersUpdateRoleMutation,
} from '@/sdk/@tanstack/react-query.gen';

const ROLES = ['owner', 'admin', 'member'] as const;

// The generated SDK types role as this union, so the page carries the same type
// rather than widening to string and casting at the call site.
type Role = (typeof ROLES)[number];

// Owner is deliberately absent from INVITABLE_ROLES: the API refuses to invite
// one, because ownership is transferred by an existing owner rather than handed
// to a stranger through an emailed link. It stays in ROLES because an owner CAN
// promote an existing member to owner.
const INVITABLE_ROLES = ['admin', 'member'] as const;

type InvitableRole = (typeof INVITABLE_ROLES)[number];

// Base UI's Select reports `string | null` — null when the selection is cleared —
// so both guards take the nullable type rather than making every call site
// narrow it first.
const isRole = (value: string | null): value is Role =>
  value !== null && (ROLES as readonly string[]).includes(value);

const isInvitableRole = (value: string | null): value is InvitableRole =>
  value !== null && (INVITABLE_ROLES as readonly string[]).includes(value);

const errorDetail = (err: unknown, fallback: string) =>
  (err as { detail?: string })?.detail ?? fallback;

export const Team = () => {
  const queryClient = useQueryClient();

  const { data: membersData, isLoading: membersLoading } = useQuery({
    ...orgMembersListOptions(),
  });
  const { data: invitesData, isLoading: invitesLoading } = useQuery({
    ...orgInvitationsListOptions(),
  });

  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState<InvitableRole>('member');

  const refreshAll = () => {
    queryClient.invalidateQueries({ queryKey: orgMembersListQueryKey() });
    queryClient.invalidateQueries({ queryKey: orgInvitationsListQueryKey() });
  };

  const invite = useMutation({
    ...orgInvitationsCreateMutation(),
    onSuccess: () => {
      toast.success('Invitation sent');
      setInviteOpen(false);
      setInviteEmail('');
      setInviteRole('member');
      refreshAll();
    },
    onError: (err) => toast.error(errorDetail(err, 'Could not send that invitation')),
  });

  const revoke = useMutation({
    ...orgInvitationsRevokeMutation(),
    onSuccess: () => {
      toast.success('Invitation withdrawn');
      refreshAll();
    },
    onError: (err) => toast.error(errorDetail(err, 'Could not withdraw that invitation')),
  });

  const updateRole = useMutation({
    ...orgMembersUpdateRoleMutation(),
    onSuccess: () => {
      toast.success('Role updated');
      refreshAll();
    },
    onError: (err) => toast.error(errorDetail(err, 'Could not change that role')),
  });

  const removeMember = useMutation({
    ...orgMembersRemoveMutation(),
    onSuccess: () => {
      toast.success('Member removed');
      refreshAll();
    },
    onError: (err) => toast.error(errorDetail(err, 'Could not remove that member')),
  });

  const handleInvite = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault();
    invite.mutate({ body: { email: inviteEmail, role: inviteRole } });
  };

  const members = membersData?.members ?? [];
  // Accepted and revoked invitations stay in the list for the audit trail; only
  // the pending ones are actionable.
  const invitations = (invitesData?.invitations ?? []).filter((i) => i.status === 'pending');

  if (membersLoading || invitesLoading) {
    return (
      <div className="p-6">
        <div className="mx-auto max-w-4xl space-y-6">
          <div className="h-8 w-48 animate-pulse rounded bg-muted" />
          <div className="h-64 animate-pulse rounded bg-muted" />
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mx-auto max-w-4xl space-y-6">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Members</CardTitle>
                <CardDescription>People with access to this organization.</CardDescription>
              </div>
              <Button onClick={() => setInviteOpen(true)}>
                <IconUserPlus className="size-4" />
                Invite
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead className="w-40">Role</TableHead>
                    <TableHead className="w-16" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {members.map((m) => (
                    <TableRow key={m.membership_id}>
                      <TableCell>
                        {[m.first_name, m.last_name].filter(Boolean).join(' ') || '—'}
                      </TableCell>
                      <TableCell className="text-muted-foreground">{m.email}</TableCell>
                      <TableCell>
                        <Select
                          value={m.role}
                          onValueChange={(role) => {
                            if (!isRole(role)) return;
                            updateRole.mutate({
                              path: { id: m.membership_id },
                              body: { role },
                            });
                          }}
                        >
                          <SelectTrigger className="w-32">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {ROLES.map((role) => (
                              <SelectItem key={role} value={role}>
                                {role}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label={`Remove ${m.email}`}
                          onClick={() => removeMember.mutate({ path: { id: m.membership_id } })}
                        >
                          <IconTrash className="size-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                  {members.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center text-muted-foreground">
                        No members yet.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Pending invitations</CardTitle>
            <CardDescription>
              Invitations that haven't been accepted yet. These count towards your member limit.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {invitations.length === 0 ? (
              <p className="text-sm text-muted-foreground">No outstanding invitations.</p>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Email</TableHead>
                      <TableHead className="w-28">Role</TableHead>
                      <TableHead className="w-56">Expires</TableHead>
                      <TableHead className="w-16" />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {invitations.map((inv) => (
                      <TableRow key={inv.id}>
                        <TableCell>{inv.email}</TableCell>
                        <TableCell>
                          <Badge variant="secondary" className="capitalize">
                            {inv.role}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {new Date(inv.expires_at).toLocaleDateString()}
                        </TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="icon"
                            aria-label={`Withdraw invitation to ${inv.email}`}
                            onClick={() => revoke.mutate({ path: { id: inv.id } })}
                          >
                            <IconTrash className="size-4" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <Dialog open={inviteOpen} onOpenChange={setInviteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Invite someone</DialogTitle>
            <DialogDescription>
              They'll get a link that only works for this address. You can't grant a role above your
              own.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleInvite} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="invite-email">Email</Label>
              <Input
                id="invite-email"
                type="email"
                required
                placeholder="colleague@example.com"
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="invite-role">Role</Label>
              <Select
                value={inviteRole}
                onValueChange={(role) => {
                  if (isInvitableRole(role)) setInviteRole(role);
                }}
              >
                <SelectTrigger id="invite-role">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {INVITABLE_ROLES.map((role) => (
                    <SelectItem key={role} value={role}>
                      {role}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <DialogFooter>
              <Button type="button" variant="ghost" onClick={() => setInviteOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={invite.isPending}>
                {invite.isPending ? 'Sending...' : 'Send invitation'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
};
