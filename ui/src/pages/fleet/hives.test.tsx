import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { Clusters } from './hives';

/**
 * Reproduces a reported bug: creating a cluster toasts success, and the list
 * keeps saying "No clusters yet".
 *
 * The page invalidates the list query in the mutation's onSuccess, so the
 * question this answers is whether that invalidation actually reaches the query
 * — which cannot be established by reading the two call sites, because the key
 * is built by a generated helper at call time.
 */
type Sent = { url: string; method: string };

function stubApi(hivesAfterCreate: unknown[]) {
  const sent: Sent[] = [];
  let created = false;

  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request) => {
      const url = input.url;
      const method = input.method.toUpperCase();
      sent.push({ url, method });

      if (method === 'POST' && url.includes('/fleet/hives')) {
        created = true;
        return new Response(JSON.stringify({ id: 'h1', name: 'dc1', provider: 'proxmox' }), {
          status: 201,
          headers: { 'content-type': 'application/json' },
        });
      }
      if (method === 'GET' && url.includes('/fleet/hives')) {
        return new Response(JSON.stringify({ hives: created ? hivesAfterCreate : [] }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        });
      }
      // Estates, for the credential picker.
      return new Response(JSON.stringify({ estates: [] }), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      });
    }),
  );
  return sent;
}

function renderPage() {
  // retry: false so a failure surfaces as a failure rather than as a timeout.
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <Clusters />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('Clusters page', () => {
  it('shows a created cluster without a manual reload', async () => {
    const sent = stubApi([
      {
        id: 'h1',
        name: 'dc1',
        provider: 'proxmox',
        minimum_nodes: 0,
        effective_minimum_nodes: 3,
        node_count: 0,
        configured: false,
        insecure_skip_verify: false,
        cpu_overcommit_ratio: 1,
      },
    ]);
    const user = userEvent.setup();
    renderPage();

    await screen.findByText(/no clusters yet/i);

    await user.click(screen.getByRole('button', { name: /new cluster/i }));
    await user.type(screen.getByLabelText(/name/i), 'dc1');
    await user.click(screen.getByRole('button', { name: /^create$/i }));

    // The POST must have gone out...
    await waitFor(() => {
      expect(sent.some((s) => s.method === 'POST')).toBe(true);
    });

    // ...and the list must then show it. This is the assertion the bug report is
    // about: success is reported and the empty state stays on screen.
    await waitFor(
      () => {
        expect(screen.queryByText(/no clusters yet/i)).toBeNull();
      },
      { timeout: 3000 },
    );
    expect(await screen.findByText('dc1')).toBeTruthy();
  });

  it('says the list FAILED rather than showing an empty state', async () => {
    // The reported symptom: create succeeds, and the page says "No clusters yet".
    // A failed list must not be indistinguishable from an empty one — that turns
    // a request failure into a claim that the thing you just made does not exist.
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: Request) => {
        if (input.method.toUpperCase() === 'GET' && input.url.includes('/fleet/hives')) {
          return new Response(JSON.stringify({ detail: 'invalid bearer token' }), {
            status: 401,
            headers: { 'content-type': 'application/problem+json' },
          });
        }
        return new Response(JSON.stringify({ estates: [] }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        });
      }),
    );
    renderPage();

    expect(await screen.findByText(/could not be loaded/i)).toBeTruthy();
    expect(screen.queryByText(/no clusters yet/i)).toBeNull();
  });
});
