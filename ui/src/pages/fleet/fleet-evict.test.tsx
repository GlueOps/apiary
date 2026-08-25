import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { Fleet } from './fleet';

/**
 * Guards the one thing about this page that cannot be checked by reading it.
 *
 * `fleet.datacenter.evict` names a COROSYNC NODE, and the agent refuses a name
 * that is not a member of the cluster. That name used to live on BeeView as
 * `hostname`; it now lives only on the machine record, as HypervisorView.name.
 * BeeView's `address` is the socket the agent answers on, which corosync has
 * never heard of — so a page that reaches for the nearest string on the row in
 * hand produces a task that fails on every hypervisor, every time, with a
 * message about membership that points nowhere near the mistake.
 *
 * This asserts the WIRE, not the label: what is in the task spec.
 */
type Sent = { url: string; method: string; body?: unknown };

const HIVE = { id: 'h1', name: 'dc1', provider: 'proxmox', effective_minimum_nodes: 3 };

const BEES = [
  { id: 'b1', address: '10.0.0.11', hypervisor_id: 'hv1', hive_id: 'h1', status: 'online' },
  { id: 'b2', address: '10.0.0.12', hypervisor_id: 'hv2', hive_id: 'h1', status: 'online' },
];

// Node names deliberately unlike the addresses: if the page ever passes an
// address through, no amount of coincidence hides it.
const HYPERVISORS = [
  { id: 'hv1', name: 'pve-alpha', hive_id: 'h1', address: '10.0.0.11' },
  { id: 'hv2', name: 'pve-bravo', hive_id: 'h1', address: '10.0.0.12' },
];

function stubApi() {
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

      if (method === 'POST' && url.includes('/fleet/tasks')) return json({ id: 't1' }, 201);
      if (url.includes('/fleet/hypervisors')) return json({ hypervisors: HYPERVISORS });
      if (url.includes('/fleet/hives')) return json({ hives: [HIVE] });
      if (url.includes('/fleet/bees')) return json({ bees: BEES });
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
        <Fleet />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('Evicting a node', () => {
  it('sends the cluster node name, not the address the agent answers on', async () => {
    const sent = stubApi();
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('10.0.0.11');

    // The first row's evict dialog: its only cluster-mate is b2 / hv2 / pve-bravo.
    const evictButtons = await screen.findAllByRole('button', { name: /evict a node/i });
    await user.click(evictButtons[0]);

    await user.click(await screen.findByRole('combobox'));
    await user.click(await screen.findByRole('option', { name: /pve-bravo/i }));

    await user.click(await screen.findByRole('button', { name: /^remove pve-bravo$/i }));

    await waitFor(() => {
      expect(sent.some((s) => s.method === 'POST' && s.url.includes('/fleet/tasks'))).toBe(true);
    });

    const task = sent.find((s) => s.method === 'POST' && s.url.includes('/fleet/tasks'))?.body as
      { bee_id?: string; kind?: string; spec?: { node?: string } } | undefined;

    expect(task?.kind).toBe('fleet.datacenter.evict');
    // Issued to a node that STAYS, naming the one that goes.
    expect(task?.bee_id).toBe('b1');
    expect(task?.spec?.node).toBe('pve-bravo');
  });
});
