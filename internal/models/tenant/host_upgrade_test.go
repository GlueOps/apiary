package tenant

import (
	"testing"
	"time"
)

func upgradeReq(mut func(*UpgradeRequest)) UpgradeRequest {
	r := UpgradeRequest{
		Hypervisor:    Hypervisor{Schedulable: false, ObservedGuests: 0},
		HiveNodeCount: 3,
		MaxAge:        10 * time.Minute,
	}
	if mut != nil {
		mut(&r)
	}
	return r
}

func upgradeCodes(rs []DepartureRefusal) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Code)
	}
	return out
}

func hasUpgradeCode(rs []DepartureRefusal, code string) bool {
	for _, r := range rs {
		if r.Code == code {
			return true
		}
	}
	return false
}

// TestAnUpgradeThatWillNotRebootIsNeverRefused.
//
// THE CASE THAT WAS WRONGLY BLOCKED. The agent ran a REBOOT preflight
// unconditionally, so `reboot: never` with guests running was refused by a check
// for a reboot that never happened. Installing packages does not interrupt a
// running guest, cost quorum, or care whether the host is cordoned.
func TestAnUpgradeThatWillNotRebootIsNeverRefused(t *testing.T) {
	t.Parallel()

	rs := CheckHostUpgrade(upgradeReq(func(r *UpgradeRequest) {
		r.Reboot = RebootNever
		// Every other condition at its worst.
		r.Hypervisor.Schedulable = true
		r.Hypervisor.ObservedGuests = 7
		r.HiveNodeCount = 2
		r.ObservationAge = 3 * time.Hour
	}))
	if len(rs) != 0 {
		t.Fatalf("an upgrade that reboots nothing was refused: %v", upgradeCodes(rs))
	}
}

// TestAnUnstatedPolicyIsTreatedAsAPossibleReboot.
//
// "if-needed" is the default, and whether it reboots is decided on the host
// after the packages arrive. The gate cannot know and must not assume the
// favourable answer: an operator who never chose a policy has still asked for
// something that can reboot.
func TestAnUnstatedPolicyIsTreatedAsAPossibleReboot(t *testing.T) {
	t.Parallel()

	rs := CheckHostUpgrade(upgradeReq(func(r *UpgradeRequest) {
		r.Reboot = "" // nothing stated
		r.Hypervisor.ObservedGuests = 1
	}))
	if !hasUpgradeCode(rs, RefuseUpgradeGuestsPresent) {
		t.Fatalf("an unstated policy was treated as safe; got %v", upgradeCodes(rs))
	}
}

// TestQuorumIsTheConstraint, not the departure minimum.
//
// A reboot is temporary — the node comes back — so the question is whether
// corosync keeps quorum while it is away, not whether the cluster could run
// permanently without it. Conflating them is why a small cluster looks
// un-upgradeable.
func TestQuorumIsTheConstraint(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		nodes   int
		refused bool
		why     string
	}{
		{0, false, "a host in no cluster has no quorum to lose"},
		{1, false, "a single-node cluster: losing it costs nothing anyone else depends on"},
		{2, true, "two nodes cannot survive losing one — quorum is 2"},
		{3, false, "three survives losing one"},
		{4, false, "four needs 3 for quorum, and losing one leaves 3 — survives"},
		{5, false, "five survives losing one"},
	} {
		rs := CheckHostUpgrade(upgradeReq(func(r *UpgradeRequest) { r.HiveNodeCount = tc.nodes }))
		if got := hasUpgradeCode(rs, RefuseUpgradeQuorum); got != tc.refused {
			t.Errorf("%d nodes: refused=%v, want %v (%s)", tc.nodes, got, tc.refused, tc.why)
		}
	}
}

// TestQuorumSurvivesLosingMatchesCorosync, spelled out because every warning
// about two-node clusters is this arithmetic.
func TestQuorumSurvivesLosingMatchesCorosync(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		members, lost int
		want          bool
	}{
		{1, 1, false}, {2, 1, false}, {3, 1, true}, {4, 1, true},
		{5, 1, true}, {5, 2, true}, {5, 3, false},
		{0, 1, false},
	} {
		if got := QuorumSurvivesLosing(tc.members, tc.lost); got != tc.want {
			t.Errorf("QuorumSurvivesLosing(%d, %d) = %v, want %v",
				tc.members, tc.lost, got, tc.want)
		}
	}
}

// TestAStaleObservationRefusesRatherThanGuessing, the same rule the departure
// gate applies: a host that looks empty in a stale observation may not be.
func TestAStaleObservationRefusesRatherThanGuessing(t *testing.T) {
	t.Parallel()

	rs := CheckHostUpgrade(upgradeReq(func(r *UpgradeRequest) {
		r.ObservationAge = 2 * time.Hour
	}))
	if !hasUpgradeCode(rs, RefuseUpgradeStale) {
		t.Fatalf("a stale observation was treated as evidence; got %v", upgradeCodes(rs))
	}
}

// TestACordonedEmptyHostIsUpgradeable, which is the ordinary case and has to
// pass cleanly or the gate is just an obstacle.
func TestACordonedEmptyHostIsUpgradeable(t *testing.T) {
	t.Parallel()

	if rs := CheckHostUpgrade(upgradeReq(nil)); len(rs) != 0 {
		t.Fatalf("a cordoned, empty host in a three-node cluster was refused: %v",
			upgradeCodes(rs))
	}
}

// TestAFleetlessCordonedHostIsUpgradeable. There is no cluster to lose quorum in
// and nothing running on it, so the only question left is answered.
func TestAFleetlessCordonedHostIsUpgradeable(t *testing.T) {
	t.Parallel()

	rs := CheckHostUpgrade(upgradeReq(func(r *UpgradeRequest) { r.HiveNodeCount = 0 }))
	if len(rs) != 0 {
		t.Fatalf("a fleetless cordoned host was refused: %v", upgradeCodes(rs))
	}
}

// TestEveryRefusalCarriesASummaryAndARemedy, so the card can group them and the
// operator is told what to do — the shape the whole refusal model settled on.
func TestEveryRefusalCarriesASummaryAndARemedy(t *testing.T) {
	t.Parallel()

	rs := CheckHostUpgrade(upgradeReq(func(r *UpgradeRequest) {
		r.Hypervisor.Schedulable = true
		r.Hypervisor.ObservedGuests = 2
		r.HiveNodeCount = 2
		r.ObservationAge = 3 * time.Hour
	}))
	if len(rs) != 4 {
		t.Fatalf("expected all four refusals at once, got %v", upgradeCodes(rs))
	}
	for _, r := range rs {
		if r.Summary == "" {
			t.Errorf("%s has no summary, so the card would show one host's figures for a group",
				r.Code)
		}
		if r.Remedy == "" {
			t.Errorf("%s says what is wrong and not what to do", r.Code)
		}
	}
}
