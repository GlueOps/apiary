import React, { useState } from 'react';

import { useMutation } from '@tanstack/react-query';
import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { authForgotPasswordMutation } from '@/sdk/@tanstack/react-query.gen';

export const ForgotPassword = () => {
  const [email, setEmail] = useState('');
  const [submitted, setSubmitted] = useState(false);

  const forgot = useMutation({
    ...authForgotPasswordMutation(),
    onSuccess: () => setSubmitted(true),
    onError: () => setSubmitted(true), // Don't reveal errors
  });

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault();
    forgot.mutate({ body: { email } });
  };

  if (submitted) {
    return (
      <div className="flex min-h-screen w-full items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="text-lg md:text-xl">Check your email</CardTitle>
            <CardDescription className="text-xs md:text-sm">
              If an account with that email exists, we've sent a password reset link.
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

  return (
    <div className="flex min-h-screen w-full items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-lg md:text-xl">Forgot password</CardTitle>
          <CardDescription className="text-xs md:text-sm">
            Enter your email and we'll send you a reset link.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                placeholder="you@example.com"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <Button type="submit" className="w-full" disabled={forgot.isPending}>
              {forgot.isPending ? 'Sending...' : 'Send reset link'}
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
        </CardContent>
      </Card>
    </div>
  );
};
