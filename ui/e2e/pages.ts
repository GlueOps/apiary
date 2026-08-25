/** Every route the suite exercises, and whether it needs a session. */
export const PUBLIC_PAGES = [
  { path: '/login', name: 'login', heading: /sign in|welcome/i },
  { path: '/signup', name: 'signup', heading: /create an account/i },
  { path: '/forgot-password', name: 'forgot-password', heading: /reset|forgot/i },
  { path: '/reset-password?token=demo-token', name: 'reset-password', heading: /password/i },
  { path: '/verify-email', name: 'verify-email-resend', heading: /link|confirm/i },
  {
    path: '/verify-email?token=demo-token',
    name: 'verify-email-confirm',
    heading: /confirm|link/i,
  },
  { path: '/accept-invitation?token=demo-token', name: 'accept-invitation', heading: /invit/i },
];

export const PROTECTED_PAGES = [
  { path: '/', name: 'dashboard' },
  { path: '/estates', name: 'credentials' },
  { path: '/fleet', name: 'fleet-agents' },
  { path: '/fleet/hypervisors', name: 'fleet-hypervisors' },
  { path: '/fleet/hives', name: 'fleet-hives' },
  { path: '/settings/profile', name: 'settings-profile' },
  { path: '/settings/security', name: 'settings-security' },
  { path: '/settings/team', name: 'settings-team' },
  { path: '/firewall/rulesets', name: 'firewall-rulesets' },
  { path: '/firewall/bindings', name: 'firewall-bindings' },
  { path: '/settings/api-keys', name: 'settings-api-keys' },
  { path: '/settings/access-rules', name: 'settings-access-rules' },
  { path: '/settings/policies', name: 'settings-policies' },
  { path: '/platform/policies', name: 'platform-policies' },
];
