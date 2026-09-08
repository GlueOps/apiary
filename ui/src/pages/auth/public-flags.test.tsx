import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { Signup } from './signup';

function stubFlags(flags: unknown[] | null) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request) => {
      if (input.url.includes('/flags/public')) {
        if (flags === null) {
          return new Response('nope', { status: 503 });
        }
        return new Response(JSON.stringify({ flags }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        });
      }
      return new Response('{}', {
        status: 200,
        headers: { 'content-type': 'application/json' },
      });
    }),
  );
}

function renderSignup() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <Signup />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('public flags on the auth pages', () => {
  // TOLD, NOT HIDDEN. Someone who followed a sign-up link needs to know that
  // registration is closed and what to do instead; rendering nothing reads as
  // broken.
  it('explains that registration is by invitation when the flag is off', async () => {
    stubFlags([{ key: 'legacy_registration', value: false }]);
    renderSignup();

    await waitFor(() => {
      expect(screen.getByText(/registration is by invitation/i)).toBeTruthy();
    });
    expect(screen.queryByLabelText(/password/i)).toBeNull();
  });

  it('renders the form when the flag is on', async () => {
    stubFlags([{ key: 'legacy_registration', value: true }]);
    renderSignup();

    await waitFor(() => {
      expect(screen.queryByText(/registration is by invitation/i)).toBeNull();
    });
  });

  // ABSENT MEANS ON, and this is the direction that matters. A page that hid its
  // form because the flag read failed would lock everyone out of a system they
  // can still sign in to — including whoever would fix it. The server refuses
  // for real; this only decides what to draw, so drawing it when unsure is the
  // recoverable error.
  it('does not close registration because the flag read failed', async () => {
    stubFlags(null);
    renderSignup();

    await waitFor(() => {
      expect(screen.queryByText(/registration is by invitation/i)).toBeNull();
    });
  });
});
