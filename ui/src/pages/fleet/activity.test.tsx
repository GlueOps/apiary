import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { FleetActivity } from './activity';

function stubApi(activity: unknown[]) {
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async () =>
        new Response(JSON.stringify({ activity }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        }),
    ),
  );
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <FleetActivity />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

const failedDisarm = {
  id: '11111111-1111-1111-1111-111111111111',
  bee_id: '22222222-2222-2222-2222-222222222222',
  address: '192.168.240.12',
  node_name: 'pve12',
  kind: 'fleet.datacenter.disarm',
  operation: 'disarm',
  status: 'failed',
  error:
    '/etc/pve/priv/pve-root-ca.srl is still byte-identical to the copy this node left the hive with',
  attempt: 1,
  created_at: '2026-08-31T07:26:54Z',
  claimed_at: '2026-08-31T07:26:54Z',
  finished_at: '2026-08-31T07:26:56Z',
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('FleetActivity', () => {
  // The reason a task failed is the whole point of the page. It is the agent's
  // own words, written to be acted on, and it must reach the screen intact
  // rather than being summarised into a status.
  it("shows the agent's own account of a failure", async () => {
    stubApi([failedDisarm]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText(/byte-identical/i)).toBeTruthy();
    });
    expect(screen.getByText('192.168.240.12')).toBeTruthy();
    expect(screen.getByText('failed')).toBeTruthy();
  });

  // A task outlives the machine it ran on: an agent can be removed and a host
  // rebuilt. Falling back to the agent id keeps that row readable, where a blank
  // cell would hide exactly the history worth reading.
  it('still names a row whose machine record has gone', async () => {
    stubApi([{ ...failedDisarm, address: '', node_name: '' }]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('22222222')).toBeTruthy();
    });
  });

  // An empty list must not read as "nothing ever ran" — the record starts when
  // it was added, and anything before that is simply not here.
  // started_at is null on every row the control plane has ever written — a bee
  // never reports "running" — so a duration read from it showed a dash on every
  // task in the fleet. claimed_at is when the work was actually picked up.
  it('times the work from when an agent picked it up', async () => {
    stubApi([failedDisarm]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText('2.0s')).toBeTruthy();
    });
  });

  // The counter increments on every claim, including a re-claim after a lease
  // expiry, so labelling it "attempt" reported retries that never happened.
  it('does not call a re-claim an attempt', async () => {
    stubApi([{ ...failedDisarm, attempt: 9 }]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText(/picked up 9/i)).toBeTruthy();
    });
    expect(screen.queryByText(/attempt 9/i)).toBeNull();
  });

  it('does not claim an empty trail means nothing happened', async () => {
    stubApi([]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText(/not evidence that nothing ran/i)).toBeTruthy();
    });
  });
});
