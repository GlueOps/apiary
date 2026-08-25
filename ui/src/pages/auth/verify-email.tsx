import { type SubmitEvent, useState } from 'react';

import { useMutation } from '@tanstack/react-query';
import { Link, useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  authResendVerificationMutation,
  authVerifyEmailMutation,
} from '@/sdk/@tanstack/react-query.gen';

type Phase = 'ready' | 'verified' | 'failed';

export const VerifyEmail = () => {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') ?? '';

  const [phase, setPhase] = useState<Phase>('ready');
  // Prefilled when login bounced an unconfirmed account here, so the person does
  // not have to retype an address they just typed. Only ever their own: they
  // reached this page by supplying that account's correct password.
  const [resendEmail, setResendEmail] = useState(searchParams.get('email') ?? '');
  const [resent, setResent] = useState(false);

  // Deliberately not fired from an effect on mount. Confirming an address
  // consumes a single-use token and provisions a workspace, so it happens on a
  // click: a link prefetcher or an email scanner following the URL must not be
  // able to burn the token before the person reaches it.
  const verify = useMutation({
    ...authVerifyEmailMutation(),
    onSuccess: () => setPhase('verified'),
    onError: () => setPhase('failed'),
  });

  const resend = useMutation({
    ...authResendVerificationMutation(),
    onSuccess: () => setResent(true),
    onError: () => setResent(true), // Same answer either way.
  });

  const shell = (title: string, description: string, children: React.ReactNode) => (
    <div className="flex min-h-screen w-full items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-lg md:text-xl">{title}</CardTitle>
          <CardDescription className="text-xs md:text-sm">{description}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">{children}</CardContent>
      </Card>
    </div>
  );

  if (phase === 'verified') {
    return shell(
      'Address confirmed',
      'Your workspace is being set up. You can sign in now.',
      <Button render={<Link to="/login" />} nativeButton={false} className="w-full">
        Sign in
      </Button>,
    );
  }

  if (phase === 'failed' || !token) {
    if (resent) {
      return shell(
        'Check your email',
        'If that address still needs confirming, a new link is on its way.',
        <Button
          render={<Link to="/login" />}
          nativeButton={false}
          variant="ghost"
          className="w-full"
        >
          Back to sign in
        </Button>,
      );
    }

    return shell(
      "That link didn't work",
      "Confirmation links are single-use and expire. Enter your address and we'll send a new one.",
      <>
        <form
          className="space-y-4"
          onSubmit={(e: SubmitEvent<HTMLFormElement>) => {
            e.preventDefault();
            resend.mutate({ body: { email: resendEmail } });
          }}
        >
          <div className="space-y-2">
            <Label htmlFor="resend-email">Email</Label>
            <Input
              id="resend-email"
              type="email"
              required
              placeholder="you@example.com"
              value={resendEmail}
              onChange={(e) => setResendEmail(e.target.value)}
            />
          </div>
          <Button type="submit" className="w-full" disabled={resend.isPending}>
            {resend.isPending ? 'Sending...' : 'Send a new link'}
          </Button>
        </form>
        <Button
          render={<Link to="/login" />}
          nativeButton={false}
          variant="ghost"
          className="w-full"
        >
          Back to sign in
        </Button>
      </>,
    );
  }

  return shell(
    'Confirm your email address',
    'One click and your workspace is set up.',
    <>
      <Button
        className="w-full"
        disabled={verify.isPending}
        onClick={() => {
          if (!token) {
            toast.error('That link is missing its confirmation token');
            return;
          }
          verify.mutate({ body: { token } });
        }}
      >
        {verify.isPending ? 'Confirming...' : 'Confirm my email address'}
      </Button>
      <Button render={<Link to="/login" />} nativeButton={false} variant="ghost" className="w-full">
        Back to sign in
      </Button>
    </>,
  );
};
