import { useQuery } from '@tanstack/react-query';

import { flagsPublicOptions } from '@/sdk/@tanstack/react-query.gen';

/**
 * The flags a page can read before anybody has signed in.
 *
 * Exists for one ordering problem: the sign-in page must decide whether to
 * render the password form, and it has to decide that with no credential to ask
 * with. Every other flag read requires a principal.
 */
export function usePublicFlags() {
  const query = useQuery({
    ...flagsPublicOptions(),
    // The sign-in page is the first thing a browser loads, and this decides what
    // it renders. Retrying keeps a cold start from painting the wrong form.
    retry: 2,
    staleTime: 60_000,
  });

  const flags = query.data?.flags ?? [];

  /**
   * ABSENT MEANS ON, and the default is load-bearing rather than lazy.
   *
   * A page that hid the password form because a request had not returned yet —
   * or because the endpoint was briefly unreachable — would lock every operator
   * out of a system they can still perfectly well sign in to, including the ones
   * who would fix it. The server decides for real: it refuses the request when
   * the flag is off. This only decides what to draw, so drawing the form when
   * unsure is the recoverable direction.
   */
  const enabled = (key: string) => flags.find((f) => f.key === key)?.value ?? true;

  return { enabled, isLoading: query.isLoading };
}
