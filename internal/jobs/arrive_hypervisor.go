package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
)

// ErrArrivalRefused means a gate said no. Like its departure counterpart it is a
// decision, not a failure, and must not surface as a 500.
var ErrArrivalRefused = errors.New("arrive: refused by a precondition")

// ArriveDeps is what arrival needs from the rest of the system.
type ArriveDeps struct {
	FleetKey  func(ctx context.Context, orgID uuid.UUID) (pub string, priv []byte, err error)
	Snapshot  func(ctx context.Context, orgID, hypervisorID, hiveID uuid.UUID) (ArrivalSnapshot, error)
	NewRunner func(addr, user string, priv []byte) (Runner, error)

	// MarkJoined records the machine's new membership: hive_id AND the node name
	// the cluster now knows it by, in one update.
	//
	// THE NODE NAME IS NOT COSMETIC. It is the capacity sync's half of this
	// machine's identity — Proxmox's /cluster/resources reports no machine id, so
	// (hive_id, name) is the only thing the sync can match on. Writing hive_id
	// alone would leave a clustered row the sync cannot recognise, and the very
	// next tick would create a SECOND record for the machine that just joined,
	// carrying none of its onboarding history. This is the deterministic half of
	// the merge; tenant.MatchMachineByNodeName is the repair path for a machine
	// that joined a cluster without going through here.
	//
	// Written only after the node is a verified quorate member, so a failure
	// part-way leaves the machine fleetless — which is true, visible, and the
	// state an operator can retry from.
	MarkJoined func(ctx context.Context, orgID, hypervisorID, hiveID uuid.UUID, nodeName string) error

	// EnqueueCapacitySync asks for a fresh observation of one hive. Departure's
	// counterpart, and a dependency for the same reason: the fleet-cycle test in
	// internal/service calls Arrive directly, with no River client to reach.
	EnqueueCapacitySync func(ctx context.Context, orgID, hiveID uuid.UUID) error

	// EnqueueCapacitySyncAfter asks for a fresh observation of one hive, LATER.
	//
	// Arrival uses this instead of the immediate variant, and the delay is the
	// entire point: Proxmox does not populate /cluster/resources stats for a node
	// the moment it joins, so a sync run immediately after a join reports zero
	// cores and zero memory for it — and writes them with last_synced_at set, so
	// the figures look freshly observed rather than absent. Measured on real
	// clusters: the sync that ran at the instant of the join recorded cpu=0
	// ram=0GiB, and the same node reported cpu=4 ram=3GiB moments later.
	//
	// It replaces the immediate sync rather than joining it. SyncHiveCapacityArgs
	// is unique ByArgs across five states INCLUDING scheduled, so enqueuing both
	// would silently drop the delayed one as a duplicate — the trap being that
	// River reports a unique collision as a successful insert, so nothing would
	// look wrong. Leaving the hive briefly without capacity figures is safe
	// because MarkJoined leaves the node unschedulable anyway.
	EnqueueCapacitySyncAfter func(ctx context.Context, orgID, hiveID uuid.UUID, after time.Duration) error
}

// ArrivalSnapshot is one consistent read of everything the arrival gate needs.
type ArrivalSnapshot struct {
	TargetHive tenant.Hive

	// CurrentHiveID is what the joining host reports today. Nil is expected.
	CurrentHiveID *uuid.UUID

	// JoiningAddress and JoiningNodeName identify the host being added.
	JoiningAddress  string
	JoiningNodeName string

	// ObservedGuests on the JOINING host, from the capacity sync.
	ObservedGuests int

	// TargetAddress is an existing member of the target hive — the node
	// `pvecm add` is pointed at, and the node the join key is authorized on.
	TargetAddress string

	// TargetNodeNames are the target hive's node names, used to seed known_hosts
	// under every name the join may resolve.
	TargetNodeNames []string

	TargetNodeCount int

	// Links are corosync rings recorded against this host, if any. The request's
	// own links take precedence; this is the fallback for a re-run that carries
	// none.
	Links []tenant.CorosyncLink

	SSHUser string
}

// ArriveInput identifies one machine to add to one hive.
//
// Keyed on the hypervisor, like departure. The asymmetry used to be forced — a
// Hypervisor row was whatever a capacity sync had observed in a cluster, so a
// host outside one had only its bee to be named by — and it stopped being true
// when that row became the durable machine record.
type ArriveInput struct {
	OrganizationID uuid.UUID
	HypervisorID   uuid.UUID
	TargetHiveID   uuid.UUID

	// Links are the corosync rings for the joining node, supplied by the operator
	// rather than derived.
	//
	// Plural because a single ring makes the cluster network a single point of
	// failure for quorum, which is the failure that fences healthy nodes — so
	// redundant links are the normal production shape. Never defaulted to the
	// management address: cluster traffic routinely belongs on a different
	// network, and guessing works until that network is busy.
	Links []tenant.CorosyncLink
}

// Arrive adds one standalone host to a hive.
//
// The mirror of Depart, and deliberately a separate operation. A migration is a
// departure and then an arrival: the fleetless state between them is real, and
// modelling it means a failed join leaves a standalone host rather than a
// half-migrated one belonging to nothing and claiming something.
//
// The sequence:
//
//  1. Gate on state this system knows: already in a hive, guests present, target
//     empty.
//  2. Gate on state only the host knows — the four things `pvecm add` refuses on
//     — read and reported in terms of what to do about them, because pvecm's own
//     message for one of them names a cause that is not the cause.
//  3. Read the target's host keys over our OWN authenticated session, and seed
//     them into the joining node's known_hosts. Without this the join fails with
//     a message that mentions none of this.
//  4. Authorize the joining node's own root key on the target, restricted to its
//     address.
//  5. Join, and verify quorate membership rather than trusting pvecm's output.
//  6. REVOKE the authorization. The file it lives in survives joins and leaves by
//     design, so leaving the grant is a permanent credential, not a stale one.
//  7. Prove the control plane can still reach the host, because the join replaced
//     its pmxcfs and destroyed every key in it.
//  8. Record the membership.
//  9. Ask for a fresh capacity observation of the hive it joined, so placement
//     does not spend the rest of the sync interval unaware of the node.
//
// Step 6 runs even when step 5 or 7 fails. That is the one deviation from
// fail-fast in this file, and it is deliberate: a failed join that leaves a
// standing root grant on the target is a worse outcome than a failed join.
func Arrive(ctx context.Context, d ArriveDeps, in ArriveInput) (err error) {
	snap, snapErr := d.Snapshot(ctx, in.OrganizationID, in.HypervisorID, in.TargetHiveID)
	if snapErr != nil {
		return fmt.Errorf("arrive: read state: %w", snapErr)
	}

	refusals := tenant.CheckArrival(tenant.ArrivalRequest{
		TargetHive:      snap.TargetHive,
		CurrentHiveID:   snap.CurrentHiveID,
		ObservedGuests:  snap.ObservedGuests,
		TargetNodeCount: snap.TargetNodeCount,
	})
	if len(refusals) > 0 {
		return fmt.Errorf("%w: %w", ErrArrivalRefused, tenant.ArrivalRefusalsError(refusals))
	}

	if snap.JoiningAddress == "" || snap.TargetAddress == "" {
		return fmt.Errorf("arrive: both the joining node's address and an existing member's " +
			"address are required")
	}
	links := in.Links
	if len(links) == 0 {
		links = snap.Links
	}
	// Validated BEFORE anything touches a host. A malformed ring set discovered
	// after the join key is authorized would leave a standing grant on the target
	// for an operation that was never going to run.
	if err := tenant.ValidateCorosyncLinks(links); err != nil {
		return fmt.Errorf("%w: %w", ErrArrivalRefused, err)
	}

	_, priv, err := d.FleetKey(ctx, in.OrganizationID)
	if err != nil {
		return fmt.Errorf("arrive: fleet key: %w", err)
	}
	user := snap.SSHUser
	if user == "" {
		user = "root"
	}

	joiner, closeJoiner, err := connect(ctx, ArriveRunnerDeps(d), snap.JoiningAddress, user, priv)
	if err != nil {
		return fmt.Errorf("arrive: connect to the joining node: %w", err)
	}
	defer closeJoiner()

	target, closeTarget, err := connect(ctx, ArriveRunnerDeps(d), snap.TargetAddress, user, priv)
	if err != nil {
		return fmt.Errorf("arrive: connect to a member of the target hive: %w", err)
	}
	defer closeTarget()

	// Asked of the machine rather than derived from its address. The snapshot's
	// name is a fallback for a host already known to a cluster; a machine being
	// onboarded by IP has no name to derive, and an empty one makes the node's own
	// pmxcfs directory read as a former peer's.
	nodeName, err := ReadNodeName(ctx, joiner)
	if err != nil {
		return err
	}

	blockers, err := ReadJoinBlockers(ctx, joiner, nodeName)
	if err != nil {
		return err
	}
	if blockers.Blocked() {
		return fmt.Errorf("%w: the joining host is not in a state pvecm will accept: %s",
			ErrArrivalRefused, blockers.Describe())
	}

	hostKeys, err := ReadHostKeys(ctx, target)
	if err != nil {
		return err
	}
	names := append([]string{hostFromAddr(snap.TargetAddress)}, snap.TargetNodeNames...)
	if err := SeedKnownHosts(ctx, joiner, names, hostKeys); err != nil {
		return err
	}

	joinerKeys, err := ReadNodeClientKeys(ctx, joiner)
	if err != nil {
		return err
	}
	// The RSA key specifically: ssh-copy-id copies /root/.ssh/id_rsa.pub, so that
	// is the one that has to be authorized. Authorizing all of them would grant
	// more than the join needs.
	joinerKey := preferRSA(joinerKeys)

	if err := AuthorizeJoinKey(ctx, target, joinerKey, hostFromAddr(snap.JoiningAddress)); err != nil {
		return err
	}

	// Revocation is deferred so it runs whatever happens next. A failed join that
	// leaves a standing root grant on the target is worse than a failed join, and
	// the grant lives in a file specifically designed to survive cluster
	// operations — so nothing else will clean it up.
	defer func() {
		if _, rerr := RevokeJoinKeys(ctx, target, joinerKeys); rerr != nil {
			if err == nil {
				err = fmt.Errorf("arrive: the join succeeded but revoking the join key failed, "+
					"which leaves the joined host with a standing root credential on the target: %w", rerr)
			} else {
				err = errors.Join(err, fmt.Errorf("and revoking the join key also failed: %w", rerr))
			}
		}
	}()

	// What cluster is the target ACTUALLY in? Read from the target itself rather
	// than taken from the hive's name: nothing enforces that a hive is named after
	// the Proxmox cluster it represents, so asserting on the hive's name fails a
	// correct join whenever an operator named the hive something else — and still
	// would not catch the case that matters, which is joining a cluster that
	// merely answers on that address.
	targetCluster, err := ReadClusterName(ctx, target)
	if err != nil {
		return err
	}
	if targetCluster == "" {
		return fmt.Errorf("%w: the node at %s belongs to no cluster, so there is nothing to join",
			ErrArrivalRefused, snap.TargetAddress)
	}

	if err := JoinCluster(ctx, joiner, hostFromAddr(snap.TargetAddress), links,
		targetCluster); err != nil {
		return err
	}

	if err := AssertManagedKeySurvived(ctx, joiner); err != nil {
		return err
	}

	if d.MarkJoined != nil {
		if err := d.MarkJoined(ctx, in.OrganizationID, in.HypervisorID, snap.TargetHive.ID,
			nodeName); err != nil {
			return fmt.Errorf("arrive: the host joined %q but recording its membership failed, "+
				"so the control plane still shows it as fleetless: %w", snap.TargetHive.Name, err)
		}
	}

	// The hive that just gained a node is re-observed SHORTLY, not at the next
	// tick and not this instant.
	//
	// Not at the next tick, for the mirror of departure's reason: the sync is the
	// only writer of the observed columns and it runs on a timer, so until it does
	// this machine has membership recorded but no capacity figures — and a row
	// with zero observed capacity reads to the placement oracle as a FULL host,
	// not a new one.
	//
	// Not this instant either, which is what this used to do and what measuring it
	// disproved: Proxmox has not populated stats for a node at the moment it
	// joins, so the immediate sync recorded exactly the zeros the paragraph above
	// is worried about — and stamped last_synced_at, making them read as observed
	// rather than missing. A short delay is what makes the observation true.
	//
	// One hive, and the gate is what guarantees it: tenant.CheckArrival refuses
	// with already_in_hive whenever CurrentHiveID is set, so a host that reaches
	// this line was fleetless and there is no source hive to also refresh.
	//
	// Inside Arrive rather than in the worker, so it runs before the deferred
	// RevokeJoinKeys closure can rewrite err — and logged rather than returned,
	// for the reason spelled out at the end of Depart. A retry here is milder than
	// departure's unterminating loop but still wrong: it re-runs AuthorizeJoinKey,
	// re-creating a root grant on the target and revoking it again, and it shows
	// the operator a failed job for a migration that worked. Real side effects for
	// a bookkeeping miss the scheduled sync repairs on its own.
	if hiveIsSyncable(snap.TargetHive) {
		enqueue := func(ctx context.Context) error {
			if d.EnqueueCapacitySyncAfter != nil {
				return d.EnqueueCapacitySyncAfter(ctx, in.OrganizationID, snap.TargetHive.ID,
					ArrivalResyncDelay)
			}
			if d.EnqueueCapacitySync != nil {
				return d.EnqueueCapacitySync(ctx, in.OrganizationID, snap.TargetHive.ID)
			}
			return nil
		}
		if err := enqueue(ctx); err != nil {
			slog.WarnContext(ctx, "arrive: the host joined its hive but a capacity sync could not "+
				"be triggered; the fleet view of that hive may be up to one sync interval out of "+
				"date until the scheduled sync runs, and a manual sync clears it immediately",
				"organization_id", in.OrganizationID, "hive_id", snap.TargetHive.ID, "error", err)
		}
	}
	return nil
}

// ArrivalResyncDelay is how long arrival waits before asking for the hive to be
// re-observed.
//
// Long enough for pvestatd to have reported the new node — it runs on a ~10s
// cycle — and short enough that an operator watching a migration does not think
// it stalled. It is not a correctness boundary: if the observation is still
// missing when this fires, the scheduled sync repairs it on its own, and the node
// is unschedulable until an operator says otherwise either way.
const ArrivalResyncDelay = 30 * time.Second

// ArriveRunnerDeps adapts ArriveDeps to the shape `connect` takes, so both
// workers share one implementation of the optional Connect/Close dance.
func ArriveRunnerDeps(d ArriveDeps) DepartDeps {
	return DepartDeps{NewRunner: d.NewRunner}
}

// preferRSA picks the key ssh-copy-id will actually copy.
func preferRSA(keys []string) string {
	for _, k := range keys {
		if strings.HasPrefix(k, "ssh-rsa ") {
			return k
		}
	}
	if len(keys) > 0 {
		return keys[0]
	}
	return ""
}

// hostFromAddr strips a port, since these values are passed to pvecm and into
// from= clauses, neither of which wants one.
func hostFromAddr(addr string) string {
	if i := strings.LastIndex(addr, ":"); i > 0 && !strings.Contains(addr[i:], "]") {
		return addr[:i]
	}
	return addr
}
