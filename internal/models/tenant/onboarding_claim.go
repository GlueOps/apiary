package tenant

import "time"

// The onboarding ticker's rules, kept here because two of them are already
// written twice.
//
// A row moves pending -> provisioning -> provisioned|failed. Only the redrive
// endpoint and this ticker move it, and both decide with SQL rather than in Go:
// the ticker claims rows concurrently, so a verdict reached at one instant and
// written at another can hand one host to two workers, both installing keys and
// reloading sshd on it at once. The predicates below are therefore fragments
// meant to sit inside the same statement as the write, never conditions applied
// to a row already loaded.
//
// THE LEASE IS THE GUARD, NOT THE STATE. `provisioning` with an expired lease is
// available again, and that is the whole recovery path for a worker killed
// mid-onboard. A NULL lease reads as expired: the claim stamps state and lease in
// one statement so the combination cannot arise from correct code, and reading it
// as "held" would strand the row with SQL as the only way out.
//
// Postgres now(), never a Go timestamp. The lease is written by a worker and read
// by an API replica, and the database is the one clock they both agree on.
const (
	// ClaimablePredicate selects rows the ticker may take.
	//
	// Not simply `provisioning_state = 'pending'`. A row left in `provisioning`
	// by a worker that died is claimable once its lease lapses, and folding that
	// into the same predicate is what makes crash recovery automatic instead of
	// something an operator has to notice and redrive by hand.
	ClaimablePredicate = `(
	provisioning_state = 'pending'
	OR (
		provisioning_state = 'provisioning'
		AND (provisioning_lease_expires_at IS NULL OR provisioning_lease_expires_at <= now())
	)
)`

	// RedrivablePredicate selects rows an operator may push back to pending.
	//
	// Deliberately NOT the same set as ClaimablePredicate, which is why they are
	// two constants rather than one. Redrive admits terminal rows — retrying a
	// failure is the point, and re-running a provisioned host is "revive", safe
	// because onboarding skips enrolment for a host already heartbeating. It
	// excludes `pending`, because resetting a row already queued looks identical
	// to a successful redrive while changing nothing the operator asked for.
	RedrivablePredicate = `(
	provisioning_state IN ('failed', 'provisioned')
	OR (
		provisioning_state = 'provisioning'
		AND (provisioning_lease_expires_at IS NULL OR provisioning_lease_expires_at <= now())
	)
)`

	// LeaseExtension is how long a claim holds a row.
	//
	// Longer than an onboard takes, and short enough that a killed worker's rows
	// come back on their own within one shift. Onboarding waits for a heartbeat
	// after installing the agent, so the slow case is minutes, not seconds — a
	// lease shorter than that would let a second worker claim a host the first is
	// still working on, which is the exact collision the lease prevents.
	LeaseExtension = 15 * time.Minute

	// MaxOnboardAttempts caps automatic retries.
	//
	// A host that has failed this many times is failing for a reason no further
	// attempt will change — wrong address, no route, a key the host will not
	// accept — and a ticker that retries forever turns one misconfigured row into
	// a permanent SSH connection attempt every tick. The cap stops the ticker,
	// not the operator: redrive clears the count, which is why redrive resets
	// provisioning_attempt rather than preserving it.
	MaxOnboardAttempts = 5
)

// UnderAttemptCapPredicate keeps a permanently failing row from being retried
// forever. Separate from ClaimablePredicate so a caller that wants to see what
// the cap is excluding can ask without it.
const UnderAttemptCapPredicate = "provisioning_attempt < ?"
