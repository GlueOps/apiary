import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { FleetlessHostView } from '@/sdk';

import { DisarmBadge } from './hypervisors';

type Disarm = FleetlessHostView['disarm'];

/**
 * A machine that LEFT a cluster and still holds its secrets authenticates
 * against a fleet it is no longer part of. That is the exposure the whole
 * departure sequence exists to close, and it was visible only by querying the
 * database.
 *
 * The verdict is judged from what the HOST reports, never from the disarm task's
 * exit status: a task reporting success is the agent's account of its own work,
 * and the point of a disarm is that certain files stop existing.
 */
describe('DisarmBadge', () => {
  it('names the exposure while the secrets are still there', () => {
    render(
      <DisarmBadge
        disarm={
          {
            disarmed: false,
            held_secrets: 4,
            disposition: 'rotate',
            detail: 'this host still holds 4 of the cluster secret files',
          } as Disarm
        }
      />,
    );
    expect(screen.getByText(/holds 4 cluster secrets/i)).toBeInTheDocument();
    expect(screen.getByText(/still holds 4 of the cluster secret files/i)).toBeInTheDocument();
  });

  /**
   * Distinct from "still armed", because the remedy is different: the secrets
   * went, and the identity the operator chose to replace did not.
   */
  it('separates a rotation that did not happen from one never asked for', () => {
    const { unmount } = render(
      <DisarmBadge
        disarm={
          {
            disarmed: true,
            held_secrets: 0,
            disposition: 'rotate',
            rotated: false,
            detail: 'the rotation it was asked for did not happen',
          } as Disarm
        }
      />,
    );
    expect(screen.getByText(/not rotated/i)).toBeInTheDocument();
    unmount();

    render(
      <DisarmBadge
        disarm={{ disarmed: true, held_secrets: 0, disposition: 'accept-exposure' } as Disarm}
      />,
    );
    expect(screen.getByText(/^disarmed$/i)).toBeInTheDocument();
  });

  it('shows a rotation that did happen', () => {
    render(
      <DisarmBadge
        disarm={{ disarmed: true, held_secrets: 0, disposition: 'rotate', rotated: true } as Disarm}
      />,
    );
    expect(screen.getByText(/disarmed · rotated/i)).toBeInTheDocument();
  });

  /**
   * A machine that was never in a cluster has nothing to have disarmed, and
   * badging every freshly onboarded host would train an operator to ignore the
   * one that matters.
   */
  it('says nothing about a machine that never left a cluster', () => {
    const { container } = render(<DisarmBadge disarm={undefined} />);
    expect(container).toBeEmptyDOMElement();
  });
});
