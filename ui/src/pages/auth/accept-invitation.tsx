import { useState } from 'react';

import { useMutation, useQuery } from '@tanstack/react-query';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useAuth } from '@/hooks/use-auth';
import {
  invitationsAcceptMutation,
  invitationsPreviewOptions,
} from '@/sdk/@tanstack/react-query.gen';

/**
 * There is deliberately NO email field on this page.
 *
 * The address an invitation admits is bound to the token server-side — Accept
 * reads it from the invitation row for the access check, the account lookup and
 * the account it creates, and AcceptInvitationInput carries no email at all. An
 * input here would be decorative at best; at worst, if the server were ever
 * changed to trust it, someone holding a token minted for one address could
 * redeem it as another and walk into the organization under an identity nobody
 * invited.
 *
 * So the email is DISPLAYED, read-only, from the preview. It tells the invitee
 * which address this is for without offering to change it.
 */
export const AcceptInvitation = () => {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') ?? '';
  const navigate = useNavigate();
  const { authed } = useAuth();

  const [error, setError] = useState<string | null>(null);
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [password, setPassword] = useState('');

  const preview = useQuery({
    ...invitationsPreviewOptions({ query: { token } }),
    enabled: token !== '',
    retry: false,
  });

  const accept = useMutation({
    ...invitationsAcceptMutation(),
    onSuccess: () => navigate('/', { replace: true }),
    onError: (err) => {
      setError((err as { detail?: string })?.detail ?? 'That invitation could not be accepted');
    },
  });

  const shell = (children: React.ReactNode) => (
    <div className="flex min-h-screen w-full items-center justify-center p-4">
      <Card className="w-full max-w-md">{children}</Card>
    </div>
  );

  if (!token) {
    return shell(
      <>
        <CardHeader>
          <CardTitle className="text-lg md:text-xl">Invalid link</CardTitle>
          <CardDescription className="text-xs md:text-sm">
            This invitation link is missing its token. Ask whoever invited you to send it again.
          </CardDescription>
        </CardHeader>
      </>,
    );
  }

  if (preview.isPending) {
    return shell(
      <CardHeader>
        <CardTitle className="text-lg md:text-xl">Checking your invitation…</CardTitle>
      </CardHeader>,
    );
  }

  if (preview.isError || !preview.data) {
    return shell(
      <>
        <CardHeader>
          <CardTitle className="text-lg md:text-xl">This invitation is no longer valid</CardTitle>
          <CardDescription className="text-xs md:text-sm">
            It may have expired, been revoked, or already been used. Ask whoever invited you for a
            new one.
          </CardDescription>
        </CardHeader>
      </>,
    );
  }

  const { email, organization, role, needs_account: needsAccount } = preview.data;

  // Already signed in: nothing to collect, the session names who is asking.
  if (authed && !needsAccount) {
    return shell(
      <>
        <CardHeader>
          <CardTitle className="text-lg md:text-xl">Join {organization}</CardTitle>
          <CardDescription className="text-xs md:text-sm">
            {error ?? `You were invited as ${role}. Your existing workspace is unaffected.`}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {!error && (
            <Button
              className="w-full"
              disabled={accept.isPending}
              onClick={() => accept.mutate({ body: { token } })}
            >
              {accept.isPending ? 'Joining…' : 'Accept invitation'}
            </Button>
          )}
          <Button render={<Link to="/" />} nativeButton={false} variant="ghost" className="w-full">
            Go to dashboard
          </Button>
        </CardContent>
      </>,
    );
  }

  // The account exists but nobody is signed in. Sign in and come straight back;
  // signing in as somebody else is refused by the server, not by this page.
  if (!needsAccount) {
    const target = `/accept-invitation?token=${encodeURIComponent(token)}`;
    return shell(
      <>
        <CardHeader>
          <CardTitle className="text-lg md:text-xl">Join {organization}</CardTitle>
          <CardDescription className="text-xs md:text-sm">
            <span className="font-medium">{email}</span> already has an account. Sign in and we'll
            add you as {role}.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button
            render={<Link to={`/login?to=${encodeURIComponent(target)}`} />}
            nativeButton={false}
            className="w-full"
          >
            Sign in to accept
          </Button>
        </CardContent>
      </>,
    );
  }

  // No account yet — create one HERE, through accept.
  //
  // This is the path that was previously a link to /signup, which was not merely
  // a longer route but a dead end: registration is gated on the create_org
  // access rules, and every public mailbox domain is denied there. Accepting an
  // invitation is gated on join_org instead, precisely because joining somebody
  // else's organization is a different question from founding your own. Sending
  // an invitee to /signup asked the wrong question and refused most of them.
  return shell(
    <>
      <CardHeader>
        <CardTitle className="text-lg md:text-xl">Join {organization}</CardTitle>
        <CardDescription className="text-xs md:text-sm">
          You were invited as {role}. Set up your account for{' '}
          <span className="font-medium">{email}</span> to accept.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            setError(null);
            accept.mutate({
              body: { token, first_name: firstName, last_name: lastName, password },
            });
          }}
        >
          {/*
            Shown, never edited. The invitation names this address and the server
            reads it from the token, so an editable field could only ever
            disagree with what will actually happen.
          */}
          <div className="space-y-2">
            <Label htmlFor="invited-email">Email</Label>
            <Input id="invited-email" value={email} readOnly disabled />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="first-name">First name</Label>
              <Input
                id="first-name"
                required
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="last-name">Last name</Label>
              <Input
                id="last-name"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
              />
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">Password</Label>
            <Input
              id="password"
              type="password"
              required
              minLength={8}
              placeholder="At least 8 characters"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <Button type="submit" className="w-full" disabled={accept.isPending}>
            {accept.isPending ? 'Creating your account…' : `Join ${organization}`}
          </Button>
        </form>
      </CardContent>
    </>,
  );
};
