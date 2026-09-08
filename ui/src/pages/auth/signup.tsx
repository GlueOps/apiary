import React, { useState } from 'react';

import { useMutation } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { usePublicFlags } from '@/hooks/use-public-flags';
import { authRegisterMutation } from '@/sdk/@tanstack/react-query.gen';

export const Signup = () => {
  const { enabled } = usePublicFlags();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [submitted, setSubmitted] = useState(false);

  const register = useMutation({
    ...authRegisterMutation(),
    // The API answers identically whether or not the address is already
    // registered, so this screen must too. Anything that distinguished the two
    // here would hand back the enumeration oracle the server works to avoid.
    onSuccess: () => setSubmitted(true),
    onError: (err) => {
      const detail = (err as { detail?: string })?.detail ?? 'Could not create your account';
      toast.error(detail);
    },
  });

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (password !== confirmPassword) {
      toast.error('Passwords do not match');
      return;
    }
    register.mutate({
      body: {
        email,
        password,
        first_name: firstName,
        last_name: lastName,
      },
    });
  };

  if (submitted) {
    return (
      <div className="flex min-h-screen w-full items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="text-lg md:text-xl">Check your email</CardTitle>
            <CardDescription className="text-xs md:text-sm">
              If that address can be registered, a confirmation link is on its way. Your workspace
              is set up once you confirm.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              render={<Link to="/login" />}
              nativeButton={false}
              variant="ghost"
              className="w-full"
            >
              Back to sign in
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  // TOLD, NOT HIDDEN. Someone who followed a sign-up link needs to know that
  // registration is closed here and what to do instead; a page that simply
  // rendered nothing would read as broken. The server refuses the request in any
  // case — this only decides what the visitor is shown.
  if (!enabled('legacy_registration')) {
    return (
      <div className="flex min-h-screen w-full items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="text-lg md:text-xl">Registration is by invitation</CardTitle>
            <CardDescription className="text-xs md:text-sm">
              Self-service sign-up is not enabled on this installation. Ask someone in your
              organization to invite you, then follow the link in that invitation.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button render={<Link to="/login" />} variant="outline" className="w-full">
              Back to sign in
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen w-full items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-lg md:text-xl">Create an account</CardTitle>
          <CardDescription className="text-xs md:text-sm">
            Start your trial. We'll email you a link to confirm your address.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="first-name">First name</Label>
                <Input
                  id="first-name"
                  required
                  autoComplete="given-name"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="last-name">Last name</Label>
                <Input
                  id="last-name"
                  autoComplete="family-name"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                required
                autoComplete="email"
                placeholder="you@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                required
                minLength={8}
                autoComplete="new-password"
                placeholder="At least 8 characters"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirm-password">Confirm password</Label>
              <Input
                id="confirm-password"
                type="password"
                required
                minLength={8}
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
              />
            </div>
            <Button type="submit" className="w-full" disabled={register.isPending}>
              {register.isPending ? 'Creating account...' : 'Create account'}
            </Button>
          </form>
          <Button
            render={<Link to="/login" />}
            nativeButton={false}
            variant="ghost"
            className="w-full"
          >
            Already have an account? Sign in
          </Button>
        </CardContent>
      </Card>
    </div>
  );
};
