import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { EstatesPage } from './EstatesPage.tsx';

/** A sentinel shaped like a real AWS secret access key. Every assertion below
 *  looks for exactly this string, so a partial redaction cannot pass. */
const THE_SECRET = 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY';

/** A complete ssh-style fingerprint: "SHA256:" plus 43 base64 characters. The
 *  length is the point — the assertions below are about it arriving whole. */
const WHOLE_FINGERPRINT = 'SHA256:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU';

const anEstate = {
  id: '9f1c2b4e-6a1d-4a2f-9b3c-1d2e3f4a5b6c',
  name: 'prod-eu-west-1',
  provider: 'aws',
  scope: { account: '111111111111', region: 'eu-west-1' },
  scope_version: 1,
  credential_id: '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d',
  credential_name: 'prod-key',
  credential_kind: 'static_secret',
  fingerprint: WHOLE_FINGERPRINT,
};

type Route = { status: number; body: unknown };

/** Stub the NETWORK, not the SDK.
 *
 *  The generated client resolves `fetch` at call time
 *  (`options.fetch ?? _config.fetch ?? globalThis.fetch`), so replacing the
 *  global leaves the entire real path in place: the real `estatesCreate`, the
 *  real URL building, the real body serialisation and the real auth hook. A test
 *  that mocked `@/sdk` instead would prove the component calls a function, not
 *  that it sends what it thinks it sends — and the assertion that the request
 *  actually CARRIED the secret is what stops "cleared the secret" from passing on
 *  a component that never sent it. */
function stubFetch(routes: { list?: Route; create?: Route; del?: Route }) {
  const sent: { url: string; method: string; body: string }[] = [];

  const impl = vi.fn(async (input: Request) => {
    const url = input.url;
    const method = input.method.toUpperCase();
    const body = input.body ? await input.clone().text() : '';
    sent.push({ url, method, body });

    let route: Route | undefined;
    if (method === 'GET') route = routes.list;
    else if (method === 'POST') route = routes.create;
    else if (method === 'DELETE') route = routes.del;

    const r = route ?? { status: 200, body: {} };
    return new Response(JSON.stringify(r.body), {
      status: r.status,
      headers: { 'Content-Type': 'application/json' },
    });
  });

  vi.stubGlobal('fetch', impl);
  return sent;
}

async function fillAndSubmit(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText('Name'), 'prod-eu-west-1');
  await user.type(screen.getByLabelText('Account ID'), '111111111111');
  await user.type(screen.getByLabelText('Region'), 'eu-west-1');
  await user.type(screen.getByLabelText('Access key ID'), 'AKIAIOSFODNN7EXAMPLE');
  await user.type(screen.getByLabelText('Secret access key'), THE_SECRET);
  await user.click(screen.getByRole('button', { name: /register credential/i }));
}

function secretInput(): HTMLInputElement {
  return screen.getByLabelText('Secret access key') as HTMLInputElement;
}

describe('EstatesPage — the client secret path', () => {
  beforeEach(() => {
    window.localStorage.setItem('access_token', 'a-bearer-token');
  });

  it('clears the secret from React state once a successful submit has sent it', async () => {
    const sent = stubFetch({
      list: { status: 200, body: { estates: [] } },
      create: { status: 200, body: anEstate },
    });
    const user = userEvent.setup();

    render(<EstatesPage />);
    await screen.findByText(/no credentials yet/i);

    await fillAndSubmit(user);

    // Half one: the request really carried the secret. Without this a component
    // that silently dropped the field would satisfy the clearing assertion below
    // while being completely broken.
    await waitFor(() => {
      expect(sent.some((r) => r.method === 'POST' && r.body.includes(THE_SECRET))).toBe(true);
    });

    // Half two: it is gone from state afterwards. A password input still holding
    // a live cloud credential is one shoulder-surf, one screen share or one
    // form-autofill extension away from disclosure, and it stays there for the
    // rest of the session.
    await waitFor(() => {
      expect(secretInput().value).toBe('');
    });

    // The whole form reset, not just the one line — otherwise this pins a single
    // statement rather than the behaviour.
    expect((screen.getByLabelText('Access key ID') as HTMLInputElement).value).toBe('');
    expect((screen.getByLabelText('Name') as HTMLInputElement).value).toBe('');
  });

  it('clears the secret from React state even when the submit is REJECTED', async () => {
    // 422 is the realistic rejection: the server refuses an unknown scope key or
    // a scope missing a required one, both of which an operator hits routinely.
    //
    // The secret has already left the browser at this point. Keeping it in state
    // buys nothing — the form is going to be re-submitted with a re-typed value
    // or abandoned — and costs a live provider credential sitting in a DOM input
    // for the remainder of the session. EstatesPage's own comment says it is
    // cleared "first and unconditionally… whatever else happens next".
    const sent = stubFetch({
      list: { status: 200, body: { estates: [] } },
      create: {
        status: 422,
        body: {
          title: 'Unprocessable Entity',
          status: 422,
          detail: 'scope contains a key this provider does not define: key "nonsense"',
        },
      },
    });
    const user = userEvent.setup();

    render(<EstatesPage />);
    await screen.findByText(/no credentials yet/i);

    await fillAndSubmit(user);

    await waitFor(() => {
      expect(sent.some((r) => r.method === 'POST' && r.body.includes(THE_SECRET))).toBe(true);
    });
    // The operator must still be told what went wrong, so clearing must not be
    // achieved by swallowing the error.
    await screen.findByRole('alert');

    await waitFor(() => {
      expect(secretInput().value).toBe('');
    });
  });

  it('renders the fingerprint whole', async () => {
    stubFetch({ list: { status: 200, body: { estates: [anEstate] } } });

    render(<EstatesPage />);

    const code = await screen.findByText(WHOLE_FINGERPRINT);

    // Exact equality, not startsWith or contains: a prefix comparison is exactly
    // the weakness this guards against. For an assumed role the identifying
    // component is an ARN whose name the customer chooses, so a short prefix is
    // grindable, and an operator comparing a truncated value against their
    // provider console is doing a comparison that proves much less than it looks
    // like it proves.
    expect(code.textContent).toBe(WHOLE_FINGERPRINT);
    expect(code.textContent).not.toContain('…');
    expect(code.textContent).not.toContain('...');

    // jsdom has no layout, so CSS-side truncation is invisible to a textContent
    // assertion. A deny-list of the classes that would clip it is the only check
    // available here — a stated tradeoff, not a claim to have measured pixels.
    const className = code.className;
    expect(className).toContain('break-all');
    for (const clipper of ['truncate', 'text-ellipsis', 'line-clamp', 'overflow-hidden']) {
      expect(className).not.toContain(clipper);
    }
  });

  it('never writes the secret to the console or to localStorage', async () => {
    const spies = {
      log: vi.spyOn(console, 'log').mockImplementation(() => {}),
      info: vi.spyOn(console, 'info').mockImplementation(() => {}),
      warn: vi.spyOn(console, 'warn').mockImplementation(() => {}),
      error: vi.spyOn(console, 'error').mockImplementation(() => {}),
      debug: vi.spyOn(console, 'debug').mockImplementation(() => {}),
    };
    // The page already reads a bearer token out of localStorage, so that store is
    // the nearest place a secret could plausibly be parked "just for the retry".
    const setItem = vi.spyOn(window.localStorage, 'setItem');

    // Both outcomes, because a debug line added to the failure branch is at least
    // as likely as one on the success branch.
    for (const create of [
      { status: 200, body: anEstate },
      { status: 422, body: { detail: 'nope' } },
    ]) {
      stubFetch({ list: { status: 200, body: { estates: [] } }, create });
      const user = userEvent.setup();
      const view = render(<EstatesPage />);
      await screen.findByText(/no credentials yet/i);
      await fillAndSubmit(user);
      await waitFor(() => expect(fetch).toHaveBeenCalled());
      view.unmount();
    }

    for (const [name, spy] of Object.entries(spies)) {
      for (const call of spy.mock.calls) {
        const rendered = call.map((a) => (typeof a === 'string' ? a : JSON.stringify(a))).join(' ');
        expect(rendered, `console.${name} was given the secret`).not.toContain(THE_SECRET);
      }
    }
    for (const call of setItem.mock.calls) {
      expect(String(call[1]), 'localStorage.setItem was given the secret').not.toContain(
        THE_SECRET,
      );
    }
  });
});

/**
 * The register form and the rotate dialog write the same value to the same
 * column, so they have to ask for it the same way.
 *
 * They did not. Register offered no example at all and said only "Not secret.
 * This is what gets fingerprinted" — which explains what happens to the value,
 * not why it is required. Rotate had a worked example and a proper explanation,
 * but hard-coded the PROXMOX one, so an AWS credential was shown
 * `root@pam!apiary` as its example.
 *
 * This is not cosmetic. A Proxmox token id is `user@realm!tokenname`, a shape
 * nobody guesses, and a credential stored without it authenticates against
 * nothing: the capacity sync fails with "the API token id is unknown and the
 * secret alone cannot authenticate", a long way from the form that accepted it.
 */
describe('EstatesPage — the identifying half is asked for the same way everywhere', () => {
  beforeEach(() => {
    window.localStorage.setItem('access_token', 'a-bearer-token');
    stubFetch({ list: { status: 200, body: { estates: [] } } });
  });

  it('shows a worked example for the selected provider, not a generic box', async () => {
    render(<EstatesPage />);
    await screen.findByText(/no credentials yet/i);

    // aws is the default provider in this form.
    const identity = screen.getByLabelText(/access key id/i);
    expect(identity).toHaveAttribute('placeholder', 'AKIAIOSFODNN7EXAMPLE');
  });

  it('explains why the field is required, not only what is done with it', async () => {
    render(<EstatesPage />);
    await screen.findByText(/no credentials yet/i);

    // The old copy said only that the value is fingerprinted. A field whose
    // stated purpose is bookkeeping is one people leave empty.
    expect(
      screen.getByText(/cannot authenticate at all, whatever the secret is/i),
    ).toBeInTheDocument();
  });

  it('offers the provider-specific example when the provider changes', async () => {
    const user = userEvent.setup();
    render(<EstatesPage />);
    await screen.findByText(/no credentials yet/i);

    // A Base UI combobox, not a native <select>: open the listbox, then pick.
    await user.click(screen.getByLabelText(/^provider$/i));
    await user.click(await screen.findByRole('option', { name: /proxmox/i }));

    // The label changes with the provider, and so must the example: a Proxmox
    // token id looks nothing like an AWS access key.
    const identity = await screen.findByLabelText(/token id/i);
    expect(identity).toHaveAttribute('placeholder', 'root@pam!apiary');
  });
});
