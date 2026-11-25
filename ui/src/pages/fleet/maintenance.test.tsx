import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { Maintenance } from './maintenance';

/**
 * A maintenance window reboots hypervisors, so WHICH hypervisors is the thing
 * the page must never get wrong. The API refuses a schedule that names no scope
 * rather than reading it as the standalone pool; these check the page always
 * names one, and the right one.
 */
type Sent = { url: string; method: string; body: Record<string, unknown> | null };

function json(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'content-type': 'application/json' },
  });
}

function stubApi(runs: unknown[] = [], schedules: unknown[] = [], strikes: unknown[] = []) {
  const sent: Sent[] = [];
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request) => {
      const method = input.method.toUpperCase();
      const text = method === 'GET' ? '' : await input.text();
      sent.push({ url: input.url, method, body: text ? JSON.parse(text) : null });
      if (method === 'POST' && input.url.includes('/fleet/maintenance/schedules')) {
        return new Response(JSON.stringify({ id: 's1', next_openings: [] }), {
          status: 201,
          headers: { 'content-type': 'application/json' },
        });
      }
      if (input.url.includes('/fleet/maintenance/strikes')) return json({ hypervisors: strikes });
      if (input.url.includes('/fleet/maintenance/schedules')) return json({ schedules });
      if (input.url.includes('/fleet/maintenance/runs')) return json({ runs });
      if (input.url.includes('/fleet/hives')) {
        return json({ hives: [{ id: 'h1', name: 'dc1', provider: 'proxmox' }] });
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
    <QueryClientProvider client={client}>
      <Maintenance />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe('Maintenance', () => {
  it('runs now against the cluster on screen, by id, and not as standalone', async () => {
    const sent = stubApi();
    renderPage();
    await screen.findByText('dc1');

    await userEvent.click(screen.getByRole('button', { name: /run now/i }));
    await userEvent.click(await screen.findByRole('button', { name: 'Start' }));

    await waitFor(() => expect(sent.some((s) => s.method === 'POST')).toBe(true));
    const post = sent.find((s) => s.method === 'POST')!;
    expect(post.body).toMatchObject({ hive_id: 'h1', kind: 'once' });
    expect(post.body).not.toHaveProperty('standalone');
  });

  it('will not start a second run while a run-now window is still waiting to start', async () => {
    stubApi(
      [],
      [
        {
          id: 's1',
          hive_id: 'h1',
          standalone: false,
          name: 'Run now',
          kind: 'once',
          once_at: new Date().toISOString(),
          duration_minutes: 240,
          reboot: 'if-needed',
          enabled: true,
          open_now: true,
          next_openings: null,
          weekdays: null,
        },
      ],
    );
    renderPage();
    await screen.findByText('open now');
    const button = screen.getByRole('button', { name: 'Run now' }) as HTMLButtonElement;
    expect(button.disabled).toBe(true);
  });

  it('offers no Resume on a one-off window that has already happened', async () => {
    stubApi(
      [],
      [
        {
          id: 's1',
          hive_id: 'h1',
          standalone: false,
          name: 'last week',
          kind: 'once',
          once_at: '2026-10-01T02:00:00Z',
          last_window_at: '2026-10-01T02:00:00Z',
          duration_minutes: 240,
          reboot: 'if-needed',
          enabled: false,
          open_now: false,
          next_openings: null,
          weekdays: null,
        },
      ],
    );
    renderPage();
    await screen.findByText('last week');
    expect(screen.queryByRole('button', { name: 'Resume' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Delete last week' })).toBeTruthy();
  });

  it('shows why a hypervisor was skipped and that a failed one was left cordoned', async () => {
    stubApi([
      {
        id: 'r1',
        hive_id: 'h1',
        standalone: false,
        status: 'failed',
        error: 'hv2 did not complete: it upgraded but was not back within 1h0m0s',
        stop_requested: false,
        reboot: 'if-needed',
        window_start: '2026-11-01T02:00:00Z',
        window_end: '2026-11-01T04:00:00Z',
        started_at: '2026-11-01T02:00:00Z',
        nodes: [
          {
            hypervisor_id: 'a',
            name: 'hv1',
            status: 'skipped',
            reason: 'no agent is attached to this machine',
            left_cordoned: false,
          },
          {
            hypervisor_id: 'b',
            name: 'hv2',
            status: 'failed',
            reason: 'it upgraded but was not back within 1h0m0s',
            left_cordoned: true,
          },
        ],
      },
    ]);
    renderPage();
    expect(await screen.findByText('no agent is attached to this machine')).toBeTruthy();
    expect(screen.getByText('left cordoned')).toBeTruthy();
  });

  it('keeps the failure budget when a window is paused', async () => {
    const sent = stubApi(
      [],
      [
        {
          id: 's1',
          hive_id: 'h1',
          standalone: false,
          name: 'weekly',
          kind: 'recurring',
          weekdays: [0],
          start_time: '02:00',
          timezone: 'Europe/Helsinki',
          duration_minutes: 120,
          reboot: 'if-needed',
          enabled: true,
          open_now: false,
          next_openings: [],
          max_nodes_per_run: 12,
          maintenance_pct: 30,
        },
      ],
    );
    renderPage();
    expect(
      await screen.findByText(/up to 12 per run; carries on until more than 30%/),
    ).toBeTruthy();
    await userEvent.click(screen.getByRole('button', { name: 'Pause' }));
    await waitFor(() => expect(sent.some((s) => s.method === 'PUT')).toBe(true));
    // The update replaces the window: a pause that left these out would reset
    // the budget to "every node, stop at the first failure".
    expect(sent.find((s) => s.method === 'PUT')!.body).toMatchObject({
      enabled: false,
      max_nodes_per_run: 12,
      maintenance_pct: 30,
    });
  });

  it('lists a flagged hypervisor in scope and acknowledges it by id', async () => {
    const sent = stubApi(
      [],
      [],
      [
        {
          hypervisor_id: 'hv-bad',
          name: 'hv7',
          hive_id: 'h1',
          strikes: 2,
          flagged: true,
          latest_at: '2026-11-20T02:30:00Z',
          latest_reason: 'the upgrade failed: dpkg exploded',
          latest_run_id: 'r9',
          schedulable: false,
        },
        {
          hypervisor_id: 'hv-other',
          name: 'loose1',
          strikes: 1,
          flagged: false,
          latest_at: '2026-11-20T02:30:00Z',
          latest_reason: 'elsewhere',
          latest_run_id: 'r8',
          schedulable: true,
        },
      ],
    );
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    renderPage();
    expect(await screen.findByText('skipped until acknowledged')).toBeTruthy();
    expect(screen.getByText('cordoned')).toBeTruthy();
    // The standalone machine is not in this cluster's scope.
    expect(screen.queryByText('loose1')).toBeNull();

    await userEvent.click(screen.getByRole('button', { name: 'Acknowledge' }));
    await waitFor(() =>
      expect(
        sent.some(
          (s) =>
            s.method === 'POST' &&
            s.url.endsWith('/fleet/hypervisors/hv-bad/maintenance/acknowledge'),
        ),
      ).toBe(true),
    );
  });
});
