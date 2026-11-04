import { describe, expect, it } from 'vitest';

import { apiErrorMessage } from './api-error';

describe('apiErrorMessage', () => {
  it("names the field and the rule from Huma's errors, not just 'validation failed'", () => {
    const msg = apiErrorMessage(
      {
        title: 'Unprocessable Entity',
        status: 422,
        detail: 'validation failed',
        errors: [
          { message: 'expected number <= 720', location: 'body.duration_minutes', value: 900 },
        ],
      },
      'fallback',
    );
    expect(msg).toContain('duration_minutes');
    expect(msg).toContain('expected number <= 720');
  });

  it('uses the detail when there are no per-field errors', () => {
    expect(apiErrorMessage({ detail: 'a run is already in progress' }, 'fallback')).toBe(
      'a run is already in progress',
    );
  });

  it('falls back when the error carries nothing', () => {
    expect(apiErrorMessage(undefined, 'fallback')).toBe('fallback');
  });
});
