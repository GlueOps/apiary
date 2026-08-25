import React, { useState } from 'react';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  meChangePasswordMutation,
  meGetOptions,
  meGetQueryKey,
  meUpdateMutation,
} from '@/sdk/@tanstack/react-query.gen';

export const ProfileSettings = () => {
  const queryClient = useQueryClient();
  const { data: me, isLoading } = useQuery({ ...meGetOptions() });

  const [firstName, setFirstName] = useState<string | null>(null);
  const [lastName, setLastName] = useState<string | null>(null);
  const [email, setEmail] = useState<string | null>(null);

  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  // Use local state if user has edited, otherwise fall back to server data
  const displayFirstName = firstName ?? me?.first_name ?? '';
  const displayLastName = lastName ?? me?.last_name ?? '';
  const displayEmail = email ?? me?.email ?? '';

  const updateProfile = useMutation({
    ...meUpdateMutation(),
    onSuccess: () => {
      toast.success('Profile updated');
      queryClient.invalidateQueries({ queryKey: meGetQueryKey() });
    },
    onError: () => toast.error('Failed to update profile'),
  });

  const changePassword = useMutation({
    ...meChangePasswordMutation(),
    onSuccess: () => {
      toast.success('Password changed');
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    },
    onError: (err) => {
      const detail = (err as { detail?: string })?.detail ?? 'Failed to change password';
      toast.error(detail);
    },
  });

  const handleProfileSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault();
    updateProfile.mutate({
      body: {
        first_name: displayFirstName,
        last_name: displayLastName,
        email: displayEmail,
      },
    });
  };

  const handlePasswordSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      toast.error('Passwords do not match');
      return;
    }
    changePassword.mutate({
      body: {
        current_password: currentPassword,
        new_password: newPassword,
      },
    });
  };

  if (isLoading) {
    return (
      <div className="p-6">
        <div className="mx-auto max-w-2xl space-y-6">
          <div className="h-8 w-48 animate-pulse rounded bg-muted" />
          <div className="h-64 animate-pulse rounded bg-muted" />
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mx-auto max-w-2xl space-y-6">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Profile</CardTitle>
                <CardDescription>Manage your personal information.</CardDescription>
              </div>
              {me?.role && (
                <Badge variant="secondary" className="capitalize">
                  {me.role}
                </Badge>
              )}
            </div>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleProfileSubmit} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="first-name">First name</Label>
                  <Input
                    id="first-name"
                    value={displayFirstName}
                    onChange={(e) => setFirstName(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="last-name">Last name</Label>
                  <Input
                    id="last-name"
                    value={displayLastName}
                    onChange={(e) => setLastName(e.target.value)}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  value={displayEmail}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>
              {me?.created_at && (
                <p className="text-xs text-muted-foreground">
                  Member since {new Date(me.created_at).toLocaleDateString()}
                </p>
              )}
              <div className="flex justify-end">
                <Button type="submit" disabled={updateProfile.isPending}>
                  {updateProfile.isPending ? 'Saving...' : 'Save changes'}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Change password</CardTitle>
            <CardDescription>
              Update your password. You'll need your current password.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handlePasswordSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="current-password">Current password</Label>
                <Input
                  id="current-password"
                  type="password"
                  required
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-password">New password</Label>
                <Input
                  id="new-password"
                  type="password"
                  required
                  minLength={8}
                  placeholder="At least 8 characters"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm-password">Confirm new password</Label>
                <Input
                  id="confirm-password"
                  type="password"
                  required
                  minLength={8}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                />
              </div>
              <div className="flex justify-end">
                <Button type="submit" disabled={changePassword.isPending}>
                  {changePassword.isPending ? 'Changing...' : 'Change password'}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};
