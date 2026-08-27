import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { AuditTrail } from './audit';

/**
 * The page exists to answer "who did this", so what it must never do is make two
 * different answers look alike: an organization API key is not a person, and a
 * refusal is not an absence.
 */
function stubApi(events: unknown[]) {
  const urls: string[] = [];
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request) => {
      urls.push(input.url);
      return new Response(JSON.stringify({ events }), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      });
    }),
  );
  return urls;
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <AuditTrail />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('Audit trail', () => {
  it('shows an organization key as a key rather than as a person', async () => {
    stubApi([
      {
        id: 'e1',
        principal_id: '11111111-2222-3333-4444-555555555555',
        principal_kind: 'org_key',
        action: 'hypervisor.cordon',
        target_kind: 'hypervisor',
        target_id: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee',
        outcome: 'ok',
        created_at: '2026-08-27T19:00:00Z',
      },
    ]);
    renderPage();

    // "API key", never "user": recording automation against a nil user is the
    // mistake this whole model exists to avoid, and rendering it as a person
    // would reintroduce it at the last step.
    expect(await screen.findByText('API key')).toBeTruthy();
    expect(screen.queryByText('user')).toBeNull();
  });

  it('shows a refusal as its own outcome, with the reason', async () => {
    stubApi([
      {
        id: 'e2',
        principal_id: '11111111-2222-3333-4444-555555555555',
        principal_kind: 'user',
        action: 'hypervisor.depart',
        target_kind: 'hypervisor',
        outcome: 'refused',
        detail: 'drain_unknown,min_nodes',
        created_at: '2026-08-27T19:05:00Z',
      },
    ]);
    renderPage();

    expect(await screen.findByText('refused')).toBeTruthy();
    // The reason codes are the point of recording a refusal at all.
    expect(screen.getByText(/drain_unknown/)).toBeTruthy();
  });

  it('names an impersonated action as both the account and the staff behind it', async () => {
    stubApi([
      {
        id: 'e3',
        principal_id: '11111111-2222-3333-4444-555555555555',
        principal_kind: 'user',
        actor_id: '99999999-8888-7777-6666-555555555555',
        action: 'hypervisor.uncordon',
        target_kind: 'hypervisor',
        outcome: 'ok',
        created_at: '2026-08-27T19:10:00Z',
      },
    ]);
    renderPage();

    // Showing only the principal reports the customer's own action; showing
    // only the actor hides whose account it ran against. Both, or neither is
    // the truth.
    // Matched on the element's whole text: the id is interpolated beside the
    // label, so it is not a text node of its own.
    const line = await screen.findByText(/via platform staff/);
    expect(line.textContent).toContain('99999999');
  });

  it('does not present an empty trail as proof that nothing happened', async () => {
    stubApi([]);
    renderPage();

    expect(await screen.findByText(/nothing recorded yet/i)).toBeTruthy();
    expect(screen.getByText(/not evidence that nothing happened/i)).toBeTruthy();
  });

  it('asks the server to filter rather than filtering what it already fetched', async () => {
    const urls = stubApi([]);
    renderPage();

    // A page that fetched 100 rows and filtered them client-side would show
    // "no results" for an action that happened 300 rows ago.
    await waitFor(() => {
      expect(urls.some((u) => u.includes('/audit/events'))).toBe(true);
    });
    expect(urls.some((u) => u.includes('limit=100'))).toBe(true);
  });
});
