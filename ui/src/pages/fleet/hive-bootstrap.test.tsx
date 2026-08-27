import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { Hypervisors } from './hypervisors';

/**
 * THE CHICKEN AND EGG. Arrival refuses with target_hive_empty until a cluster
 * has at least one node, which is every cluster on the day it is created — so
 * until this existed, a new cluster could never acquire a first node and every
 * arrival into it refused forever. On a fleet where nothing was clustered yet,
 * that was every machine.
 *
 * Which operation runs is decided by the node count rather than by the operator,
 * because the count already answers it. These assert the WIRE: an empty cluster
 * must POST to bootstrap and never to arrive, and a populated one the reverse.
 */
type Sent = { url: string; method: string; body?: unknown };

const STRAY = {
  address: '192.168.240.11',
  arrival_refusals: null,
  hypervisor_id: 'hv1',
  provisioning_state: 'provisioned' as const,
  status: 'online' as const,
};

function stubApi(nodeCount: number) {
  const sent: Sent[] = [];
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request) => {
      const url = input.url;
      const method = input.method.toUpperCase();
      let body: unknown;
      if (method === 'POST') {
        try {
          body = JSON.parse(await input.clone().text());
        } catch {
          body = undefined;
        }
      }
      sent.push({ url, method, body });

      const json = (v: unknown, status = 200) =>
        new Response(JSON.stringify(v), {
          status,
          headers: { 'content-type': 'application/json' },
        });

      if (method === 'POST') return json({ queued: true, detail: 'queued' }, 202);
      if (url.includes('/fleet/fleetless')) return json({ hosts: [STRAY] });
      if (url.includes('/fleet/hives')) {
        return json({
          hives: [
            {
              id: 'h1',
              name: 'dc1',
              provider: 'proxmox',
              node_count: nodeCount,
              minimum_nodes: 0,
              effective_minimum_nodes: 3,
              configured: false,
              insecure_skip_verify: false,
              cpu_overcommit_ratio: 1,
            },
          ],
        });
      }
      if (url.includes('/fleet/hypervisors')) return json({ hypervisors: [] });
      return json({});
    }),
  );
  return sent;
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <Hypervisors />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

async function openAndSubmit(user: ReturnType<typeof userEvent.setup>, submitLabel: RegExp) {
  await screen.findByText('192.168.240.11');
  await user.click(await screen.findByRole('button', { name: /join a cluster/i }));
  await user.click(await screen.findByRole('combobox'));
  await user.click(await screen.findByRole('option', { name: /dc1/i }));
  await user.type(screen.getByPlaceholderText(/10\.|address|ring/i), '192.168.240.11');
  await user.click(await screen.findByRole('button', { name: submitLabel }));
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('Putting the first node into a cluster', () => {
  it('creates the cluster when it has no nodes, rather than trying to join it', async () => {
    const sent = stubApi(0);
    const user = userEvent.setup();
    renderPage();

    await openAndSubmit(user, /create the cluster/i);

    await waitFor(() => {
      expect(sent.some((s) => s.method === 'POST' && s.url.includes('/bootstrap'))).toBe(true);
    });
    // Joining an empty cluster is refused by the server and cannot ever work.
    expect(sent.some((s) => s.url.includes('/arrive'))).toBe(false);
  });

  it('joins a cluster that already has nodes', async () => {
    const sent = stubApi(2);
    const user = userEvent.setup();
    renderPage();

    await openAndSubmit(user, /queue arrival/i);

    await waitFor(() => {
      expect(sent.some((s) => s.method === 'POST' && s.url.includes('/arrive'))).toBe(true);
    });
    // Creating a cluster that exists would replace it.
    expect(sent.some((s) => s.url.includes('/bootstrap'))).toBe(false);
  });
});
