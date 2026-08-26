import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { SshKeys } from './ssh-keys.tsx';

/** A real ed25519 public key shape. The assertions look for this exact string,
 *  so a truncated or re-wrapped render cannot pass — a key pasted into
 *  authorized_keys with a line break in it silently does not work. */
const THE_PUBLIC_KEY =
  'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJ9kL2mQ4vXsT8rY1wZ6cB3nF5hK7pD0aG2eR4tU6iOv apiary-fleet';

const THE_FINGERPRINT = 'SHA256:F8BGIn1g2ioKQiqsxBH6yMWevPNcdGrTrpUq3noDb/E';

const aKey = {
  id: '3f2a1b8c-4d5e-4f6a-9b7c-8d9e0f1a2b3c',
  name: 'apiary-fleet',
  public_key: THE_PUBLIC_KEY,
  fingerprint: THE_FINGERPRINT,
  purpose: 'fleet' as const,
  usable: true,
  revoked: false,
  created_at: '2026-08-01T10:00:00Z',
};

/** Stub the NETWORK rather than the SDK, for the reason EstatesPage.test.tsx
 *  gives: the generated client resolves `fetch` at call time, so this leaves the
 *  real query, URL building and auth hook in place. */
function stubFetch(body: unknown) {
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async () =>
        new Response(JSON.stringify(body), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
    ),
  );
}

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <SshKeys />
    </QueryClientProvider>,
  );
}

describe('SshKeys — the automated onboarding prerequisite', () => {
  beforeEach(() => {
    window.localStorage.setItem('access_token', 'a-bearer-token');
  });

  it('renders the public key whole, because a re-wrapped one silently does not work', async () => {
    stubFetch({ keys: [aKey] });
    renderPage();

    // Wait for the table to arrive before reading the document, or this asserts
    // against the loading state.
    await screen.findByText(THE_FINGERPRINT);

    // The install command carries the key VERBATIM. A key broken across lines is
    // accepted by the shell and rejected by sshd, with no error anyone sees until
    // onboarding fails as "permission denied" a step later — so the assertion is
    // on the exact string, not on a fragment of it.
    expect(document.body.textContent).toContain(THE_PUBLIC_KEY);
  });

  it('shows the fingerprint, which is how the operator verifies what they pasted', async () => {
    stubFetch({ keys: [aKey] });
    renderPage();

    // OpenSSH's own SHA256 form, so it compares directly against `ssh-keygen -lf`
    // on the hypervisor. Without it an operator cannot tell the key they
    // installed from one that arrived some other way.
    expect(await screen.findByText(THE_FINGERPRINT)).not.toBeNull();
  });

  it('says the automated path is unavailable when no key exists, rather than showing nothing', async () => {
    stubFetch({ keys: [] });
    renderPage();

    // An empty table would read as "nothing to do here". The truth is that the
    // control plane cannot reach any hypervisor at all, which is the one fact
    // that explains why onboarding a host has to be done by hand.
    expect(await screen.findByText(/no fleet key yet/i)).not.toBeNull();
    expect(screen.getByText(/cannot connect to a hypervisor/i)).not.toBeNull();
  });

  it('never renders a private half, which no endpoint returns', async () => {
    stubFetch({ keys: [aKey] });
    renderPage();
    await screen.findByText(THE_FINGERPRINT);

    expect(document.body.textContent).not.toMatch(/BEGIN [A-Z ]*PRIVATE KEY/);
  });
});

describe('SshKeys — the install command', () => {
  beforeEach(() => {
    window.localStorage.setItem('access_token', 'a-bearer-token');
  });

  it('installs to authorized_keys.d, never to /root/.ssh/authorized_keys', async () => {
    stubFetch({ keys: [aKey] });
    renderPage();
    await screen.findByText(THE_FINGERPRINT);

    const page = document.body.textContent ?? '';

    // On a Proxmox node /root/.ssh/authorized_keys is a symlink onto pmxcfs, and
    // a cluster join REPLACES that database wholesale. A key installed there
    // disappears the moment the host joins a fleet, and the failure reads as the
    // join having broken SSH.
    expect(page).toContain('/etc/ssh/authorized_keys.d/apiary');
    expect(page).not.toMatch(/>\s*\/root\/\.ssh\/authorized_keys/);
  });

  it('writes the sshd drop-in, without which the key is simply ignored', async () => {
    stubFetch({ keys: [aKey] });
    renderPage();
    await screen.findByText(THE_FINGERPRINT);

    const page = document.body.textContent ?? '';
    expect(page).toContain('/etc/ssh/sshd_config.d/60-apiary.conf');
    expect(page).toContain('AuthorizedKeysFile');

    // Validated before the reload: a configuration sshd refuses to parse means
    // it does not come back, and that is the one failure here nobody can undo
    // remotely. Reload, not restart, so the session doing this survives.
    expect(page).toContain('sshd -t');
    expect(page).not.toContain('systemctl restart ssh');
  });
});
