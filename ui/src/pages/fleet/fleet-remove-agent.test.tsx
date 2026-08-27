import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { Fleet } from './fleet';

// The page reports failure through a toast, and no <Toaster /> is mounted here.
// Asserting on the call is asserting the same thing the operator sees, without
// dragging sonner's portal into the test.
const toastError = vi.fn();
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: (...args: unknown[]) => toastError(...args) },
}));

/**
 * Removing an agent had an endpoint and no way to reach it.
 *
 * The thing worth guarding is that it stays distinct from eviction. Both live on
 * the same row and both read as "get rid of this", but eviction is about cluster
 * membership and runs a task on the hypervisors, while this revokes a credential
 * and deletes a record. Wiring the row to the wrong one produces a control that
 * looks right and does something else, so these assert the WIRE: a DELETE to the
 * agent's own id, and no task queued.
 */
type Sent = { url: string; method: string; body?: unknown };

const HIVE = { id: 'h1', name: 'dc1', provider: 'proxmox', effective_minimum_nodes: 3 };

const BEES = [
  { id: 'b1', address: '10.0.0.11', hypervisor_id: 'hv1', hive_id: 'h1', status: 'online' },
  { id: 'b2', address: '10.0.0.12', hypervisor_id: 'hv2', hive_id: 'h1', status: 'online' },
];

const HYPERVISORS = [
  { id: 'hv1', name: 'pve-alpha', hive_id: 'h1', address: '10.0.0.11' },
  { id: 'hv2', name: 'pve-bravo', hive_id: 'h1', address: '10.0.0.12' },
];

function stubApi(onDelete?: () => Response) {
  const sent: Sent[] = [];
  let removed = false;

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

      if (method === 'DELETE' && url.includes('/fleet/bees/')) {
        if (onDelete) return onDelete();
        removed = true;
        return new Response(null, { status: 204 });
      }
      if (method === 'POST' && url.includes('/fleet/tasks')) return json({ id: 't1' }, 201);
      if (url.includes('/fleet/hypervisors')) return json({ hypervisors: HYPERVISORS });
      if (url.includes('/fleet/hives')) return json({ hives: [HIVE] });
      if (url.includes('/fleet/bees')) {
        return json({ bees: removed ? BEES.filter((b) => b.id !== 'b1') : BEES });
      }
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
  toastError.mockClear();
});

describe('Removing an agent', () => {
  it('deletes the agent record and queues no task', async () => {
    const sent = stubApi();
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('10.0.0.11');
    await user.click(screen.getByRole('button', { name: /remove the agent on 10\.0\.0\.11/i }));

    // The consequence an operator would otherwise discover on the host: the
    // process keeps running, and only its credential stops working.
    expect(
      await screen.findByText(/keeps running and starts failing authentication/i),
    ).toBeTruthy();
    await user.click(screen.getByRole('button', { name: /^remove the agent$/i }));

    await waitFor(() => {
      expect(sent.some((s) => s.method === 'DELETE' && s.url.includes('/fleet/bees/b1'))).toBe(
        true,
      );
    });
    // Not an eviction: nothing runs on any hypervisor.
    expect(sent.some((s) => s.method === 'POST' && s.url.includes('/fleet/tasks'))).toBe(false);

    await waitFor(() => {
      expect(screen.queryByText('10.0.0.11')).toBeNull();
    });
  });

  it("shows the server's reason when an onboarding lease blocks the removal", async () => {
    // 409 while a worker is installing and enrolling this machine. "Try again in
    // a minute" and "something broke" are different facts and the operator can
    // only tell them apart if the detail survives.
    stubApi(
      () =>
        new Response(
          JSON.stringify({ detail: 'an onboarding worker holds a lease on this machine' }),
          {
            status: 409,
            headers: { 'content-type': 'application/problem+json' },
          },
        ),
    );
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('10.0.0.11');
    await user.click(screen.getByRole('button', { name: /remove the agent on 10\.0\.0\.11/i }));
    await user.click(await screen.findByRole('button', { name: /^remove the agent$/i }));

    await waitFor(() => {
      expect(toastError).toHaveBeenCalledWith('an onboarding worker holds a lease on this machine');
    });
  });
});
