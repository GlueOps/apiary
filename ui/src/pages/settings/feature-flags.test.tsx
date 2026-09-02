import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { FeatureFlags } from './feature-flags';

function stubApi(flags: unknown[], overrides: unknown[] = []) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request) => {
      const body = input.url.includes('/overrides') ? { overrides } : { flags };
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

  // A platform-scoped flag has no per-tenant value at all, and the page says so
  // rather than offering a form that would be refused with a 409.
  it('offers no per-organization override for a platform-scoped flag', async () => {
    stubApi([platformFlag]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText(/installation-wide/i)).toBeTruthy();
    });
    expect(screen.queryByPlaceholderText('organization id')).toBeNull();
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
