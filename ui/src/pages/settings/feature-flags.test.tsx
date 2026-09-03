import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { FeatureFlags } from './feature-flags';

function stubApi(flags: unknown[], overrides: unknown[] = [], organizations: unknown[] = []) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request) => {
      const body = input.url.includes('/overrides')
        ? { overrides }
        : input.url.includes('/platform/organizations')
          ? { organizations }
          : { flags };
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      });
    }),
  );
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <FeatureFlags />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

const tenantFlag = {
  key: 'proxmox.provisioning',
  value: false,
  source: 'platform',
  scope: 'tenant',
  default: false,
  why: 'provision Proxmox guests for this organization',
};

const platformFlag = {
  key: 'legacy_login',
  value: true,
  source: 'default',
  scope: 'platform',
  default: true,
  why: 'the pre-OIDC username and password login form',
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('FeatureFlags', () => {
  // WHERE A VALUE CAME FROM IS NOT DERIVABLE FROM THE VALUE. "Off", "off because
  // the platform says so" and "off for this tenant" need three different
  // actions, and a bare toggle shows the same thing for all three.
  it('says which level decided each value', async () => {
    stubApi([platformFlag, tenantFlag]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('legacy_login')).toBeTruthy();
    });
    expect(screen.getByText(/built-in default/i)).toBeTruthy();
    expect(screen.getByText(/set here/i)).toBeTruthy();
  });

  // The overrides list exists to answer "who will a change here NOT reach". A
  // count alone does not, so the organizations are named.
  it('names the organizations a change will not reach', async () => {
    stubApi(
      [tenantFlag],
      [
        {
          organization_id: '11111111-1111-1111-1111-111111111111',
          organization_name: 'Acme',
          value: true,
        },
      ],
    );
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Acme')).toBeTruthy();
    });
    expect(screen.getByText(/does not reach these/i)).toBeTruthy();
  });

  // THE DIRECTORY IS NOT FETCHED UNTIL SOMETHING IS TYPED. A list of every
  // tenant open under each flag card is one full directory read per flag, for a
  // list nobody has asked to see.
  it('does not search organizations until asked', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const body = input.url.includes('/overrides') ? { overrides: [] } : { flags: [tenantFlag] };
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      });
    });
    vi.stubGlobal('fetch', fetchSpy);
    renderPage();

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/search organizations/i)).toBeTruthy();
    });
    const urls = fetchSpy.mock.calls.map((c) => (c[0] as Request).url);
    expect(urls.some((u) => u.includes('/platform/organizations'))).toBe(false);
  });

  // A platform-scoped flag has no per-tenant value at all, and the page says so
  // rather than offering a form that would be refused with a 409.
  it('offers no per-organization override for a platform-scoped flag', async () => {
    stubApi([platformFlag]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText(/installation-wide/i)).toBeTruthy();
    });
    expect(screen.queryByPlaceholderText(/search organizations/i)).toBeNull();
  });

  // CLEARING IS NOT SETTING FALSE, and the control has to say which default it
  // falls back to — otherwise restoring one means knowing what it used to be.
  it('names the default that clearing restores', async () => {
    stubApi([platformFlag]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText(/use the default \(on\)/i)).toBeTruthy();
    });
  });
});
