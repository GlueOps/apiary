/**
 * The message to show for a failed API call.
 *
 * Huma answers a request that fails validation with `detail: "validation
 * failed"` and the useful part — which field, and what rule — in `errors[]`.
 * Showing only the detail told an operator their window was wrong without
 * saying how.
 */
export function apiErrorMessage(e: unknown, fallback: string): string {
  const err = e as
    { detail?: string; errors?: { message?: string; location?: string }[] } | null | undefined;
  const details = (err?.errors ?? [])
    .filter((d) => d?.message)
    .map((d) => (d.location ? `${d.location.replace(/^body\./, '')}: ${d.message}` : d.message));
  if (details.length) {
    return err?.detail ? `${err.detail}: ${details.join('; ')}` : details.join('; ');
  }
  return err?.detail ?? fallback;
}
