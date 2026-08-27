import { createBrowserRouter } from 'react-router-dom';

import { PlatformRoute } from '@/components/platform-route.tsx';
import { ProtectedRoute } from '@/components/protected-route.tsx';
import { RouteErrorBoundary } from '@/components/route-error-boundary.tsx';
import { AppShell } from '@/layouts/app-shell.tsx';
import { AcceptInvitation } from '@/pages/auth/accept-invitation.tsx';
import { ForgotPassword } from '@/pages/auth/forgot-password.tsx';
import { ResetPassword } from '@/pages/auth/reset-password.tsx';
import { Signup } from '@/pages/auth/signup.tsx';
import { VerifyEmail } from '@/pages/auth/verify-email.tsx';
import { Dashboard } from '@/pages/dashboard/dashboard.tsx';
import { EstatesPage } from '@/pages/estates/EstatesPage.tsx';
import { FirewallBindings } from '@/pages/firewall/bindings.tsx';
import { FirewallPublish } from '@/pages/firewall/publish.tsx';
import { FirewallRulesets } from '@/pages/firewall/rulesets.tsx';
import { FirewallSets } from '@/pages/firewall/sets.tsx';
import { Fleet } from '@/pages/fleet/fleet.tsx';
import { Clusters } from '@/pages/fleet/hives.tsx';
import { Hypervisors } from '@/pages/fleet/hypervisors.tsx';
import { AccessRules } from '@/pages/settings/access-rules.tsx';
import { ApiKeys } from '@/pages/settings/api-keys.tsx';
import { DangerZone } from '@/pages/settings/danger-zone.tsx';
import { OrgPolicies } from '@/pages/settings/org-policies.tsx';
import { PlatformPolicies } from '@/pages/settings/platform-policies.tsx';
import { ProfileSettings } from '@/pages/settings/profile.tsx';
import { Security } from '@/pages/settings/security.tsx';
import { SshKeys } from '@/pages/settings/ssh-keys.tsx';
import { Team } from '@/pages/settings/team.tsx';
import { LoginRoute } from '@/routes/login-route.tsx';

export const router = createBrowserRouter([
  {
    errorElement: <RouteErrorBoundary />,
    children: [
      {
        path: '/login',
        element: <LoginRoute />,
        handle: { title: 'Sign In' },
      },
      {
        path: '/signup',
        element: <Signup />,
        handle: { title: 'Sign Up' },
      },
      {
        path: '/verify-email',
        element: <VerifyEmail />,
        handle: { title: 'Confirm Email' },
      },
      {
        path: '/accept-invitation',
        element: <AcceptInvitation />,
        handle: { title: 'Accept Invitation' },
      },
      {
        path: '/forgot-password',
        element: <ForgotPassword />,
        handle: { title: 'Forgot Password' },
      },
      {
        path: '/reset-password',
        element: <ResetPassword />,
        handle: { title: 'Reset Password' },
      },
      {
        element: <ProtectedRoute />,
        children: [
          {
            element: <AppShell />,
            children: [
              {
                index: true,
                element: <Dashboard />,
                handle: { title: 'Dashboard' },
              },
              {
                path: 'fleet',
                element: <Fleet />,
                handle: { title: 'Agents' },
              },
              {
                path: 'fleet/hypervisors',
                element: <Hypervisors />,
                handle: { title: 'Hypervisors' },
              },
              {
                path: 'fleet/hives',
                element: <Clusters />,
                handle: { title: 'Clusters' },
              },
              {
                path: 'estates',
                element: <EstatesPage />,
                handle: { title: 'Credentials' },
              },
              {
                path: 'firewall/rulesets',
                element: <FirewallRulesets />,
                handle: { title: 'Firewall rules' },
              },
              {
                path: 'firewall/sets',
                element: <FirewallSets />,
                handle: { title: 'Address sets' },
              },
              {
                path: 'firewall/bindings',
                element: <FirewallBindings />,
                handle: { title: 'Firewall bindings' },
              },
              {
                path: 'firewall/publish',
                element: <FirewallPublish />,
                handle: { title: 'Publish' },
              },
              {
                path: 'settings/policies',
                element: <OrgPolicies />,
                handle: { title: 'Security policy' },
              },
              {
                // The PLATFORM boundary. These pages edit controls that bind
                // every tenant, so they get a role gate of their own rather than
                // relying on the sidebar to hide them — a hidden link is still a
                // reachable URL.
                element: <PlatformRoute />,
                children: [
                  {
                    path: 'platform/policies',
                    element: <PlatformPolicies />,
                    handle: { title: 'Platform policy' },
                  },
                  {
                    path: 'settings/access-rules',
                    element: <AccessRules />,
                    handle: { title: 'Access rules' },
                  },
                ],
              },
              {
                path: 'settings/team',
                element: <Team />,
                handle: { title: 'Team' },
              },
              {
                path: 'settings/profile',
                element: <ProfileSettings />,
                handle: { title: 'Profile' },
              },
              {
                path: 'settings/security',
                element: <Security />,
                handle: { title: 'Security' },
              },
              {
                path: 'settings/api-keys',
                element: <ApiKeys />,
                handle: { title: 'API keys' },
              },
              {
                path: 'settings/ssh-keys',
                element: <SshKeys />,
                handle: { title: 'Fleet SSH keys' },
              },
              {
                path: 'settings/danger-zone',
                element: <DangerZone />,
                handle: { title: 'Danger zone' },
              },
            ],
          },
        ],
      },
    ],
  },
]);
