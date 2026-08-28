import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { HypervisorView } from '@/sdk';

import { LastOperationBadge } from './hypervisors';

const base = {
  id: 'hv1',
  address: '192.168.240.12',
  schedulable: false,
  provisioning_state: 'provisioned',
} as unknown as HypervisorView;

/**
 * The outcome of a fleet OPERATION — depart, arrive, bootstrap — was recorded on
 * the machine row and rendered by nothing. A failed departure showed up as a
 * node that simply had not left, and diagnosing one meant querying the database.
 */
describe('LastOperationBadge', () => {
  it("shows the worker's own message, verbatim", () => {
    render(
      <LastOperationBadge
        hv={
          {
            ...base,
            last_operation: 'depart',
            last_operation_error:
              'depart: the cluster config lists 1 node(s), so there is no node to remove ' +
              'without destroying the cluster',
          } as HypervisorView
        }
      />,
    );

    expect(screen.getByText(/last departure failed/i)).toBeTruthy();
    // Verbatim: these messages are written to say what to do about the failure,
    // and paraphrasing them here would drop that half.
    expect(screen.getByText(/no node to remove without destroying the cluster/)).toBeTruthy();
  });

  it('names the operation it was, not a generic failure', () => {
    render(
      <LastOperationBadge
        hv={{ ...base, last_operation: 'join', last_operation_error: 'boom' } as HypervisorView}
      />,
    );
    expect(screen.getByText(/last join failed/i)).toBeTruthy();
  });

  it('renders nothing when the last operation succeeded', () => {
    const { container } = render(
      <LastOperationBadge hv={{ ...base, last_operation: 'depart' } as HypervisorView} />,
    );
    // A cleared error means the last attempt worked. Showing a stale failure
    // would send someone to investigate something already fixed.
    expect(container.textContent).toBe('');
  });
});
