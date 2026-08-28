import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { Clusters } from './hives';

/**
 * Deleting a cluster had an endpoint and no way to reach it.
 *
 * The endpoint refuses while the cluster still holds hypervisors, and the two
 * assertions here are the two halves of honouring that: the action is offered
 * when it can succeed, and it is withheld — with the reason on screen — when it
 * cannot. Letting the click go out to be refused would spend a round trip to
 * learn something the list already said.
 */
type Sent = { url: string; method: string };

function cluster(overrides: Record<string, unknown> = {}) {
  return {
    id: 'h1',
    name: 'dc1',
    provider: 'proxmox',
    minimum_nodes: 0,
    effective_minimum_nodes: 3,
    node_count: 0,
    configured: false,
    insecure_skip_verify: false,
    cpu_overcommit_ratio: 1,
    ...overrides,
  };
}

function stubApi(hive: Record<string, unknown>) {
  const sent: Sent[] = [];
  let deleted = false;

  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request) => {
      const url = input.url;
      const method = input.method.toUpperCase();
      sent.push({ url, method });

      if (method === 'DELETE' && url.includes('/fleet/hives/')) {
        deleted = true;
        return new Response(null, { status: 204 });
      }
      if (method === 'GET' && url.includes('/fleet/hives')) {
        return new Response(JSON.stringify({ hives: deleted ? [] : [hive] }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        });
      }
      return new Response(JSON.stringify({ estates: [] }), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      });
    }),
  );
  return sent;
}

function renderPage() {
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

describe('Deleting a cluster', () => {
  it('deletes an empty cluster and drops it from the list', async () => {
    const sent = stubApi(cluster({ node_count: 0 }));
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('dc1');
    await user.click(screen.getByRole('button', { name: /delete dc1/i }));

    // The consequences are stated before the click that does it, because nothing
    // about this action is visible afterwards.
    expect(await screen.findByText(/nothing is done to proxmox/i)).toBeTruthy();
    await user.click(screen.getByRole('button', { name: /delete this cluster/i }));

    await waitFor(() => {
      expect(sent.some((s) => s.method === 'DELETE' && s.url.includes('/fleet/hives/h1'))).toBe(
        true,
      );
    });
    await waitFor(() => {
      expect(screen.queryByText('dc1')).toBeNull();
    });
  });

  it('withholds the action while the cluster still holds hypervisors, and says why', async () => {
    const sent = stubApi(cluster({ node_count: 2 }));
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('dc1');

    const button = screen.getByRole('button', { name: /delete dc1/i });
    expect((button as HTMLButtonElement).disabled).toBe(true);

    // Disabled alone is a dead end. The reason has to be on the card, not only
    // in a title attribute a keyboard user never sees.
    expect(screen.getByText(/depart them first/i)).toBeTruthy();

    await user.click(button);
    expect(sent.some((s) => s.method === 'DELETE')).toBe(false);
  });
});
