package tenant

import (
	"strings"
	"testing"
)

func member(node, id, addr string) MembershipMember {
	return MembershipMember{Node: node, HypervisorID: id, Address: addr, Fingerprint: "AA:BB"}
}

func labA(nodes ...string) Membership {
	m := Membership{Cluster: "lab-a"}
	for i, n := range nodes {
		m.Members = append(m.Members, member(n, string(rune('a'+i)), "10.0.0."+n))
	}
	return m
}

// departing marks a node as one the control plane positively removed, which is
// the only thing that authorises an eviction.
func departing(m Membership, nodes ...string) Membership {
	m.Departing = append(m.Departing, nodes...)
	return m
}

// A node in corosync that the control plane has no record of must NOT be
// evicted. It is as likely to be a host somebody joined by hand and the capacity
// sync has not adopted yet as it is one that was removed — and the two need
// opposite treatment, so inference is not good enough for a destructive act.
func TestAnUnrecordedNodeIsReportedNotEvicted(t *testing.T) {
	m := labA("pve1", "pve2")
	obs := Observed{Cluster: "lab-a", Members: []string{"pve1", "pve2", "pve9"}, Quorate: true}

	verb, target, why := m.Decide(Self{Node: "pve1"}, obs)
	if verb != VerbNone {
		t.Fatalf("verb = %q (target %q), want none for an unrecorded node", verb, target)
	}
	if why == "" {
		t.Fatal("an unrecorded node was ignored silently; it should be reported as a discrepancy")
	}
}

func TestDecide(t *testing.T) {
	cases := []struct {
		name   string
		desire Membership
		self   Self
		obs    Observed
		verb   MembershipVerb
		target string
	}{
		{
			name:   "desired in and standalone joins",
			desire: labA("pve1", "pve2"),
			self:   Self{Node: "pve2"},
			obs:    Observed{},
			verb:   VerbJoin,
		},
		{
			name:   "converged does nothing",
			desire: labA("pve1", "pve2"),
			self:   Self{Node: "pve2"},
			obs:    Observed{Cluster: "lab-a", Members: []string{"pve1", "pve2"}, Quorate: true},
			verb:   VerbNone,
		},
		{
			// The departure case. pve3 is recorded as departed and still voting;
			// the lowest-named remaining member owns the removal.
			name:   "lowest remaining member evicts a recorded departure",
			desire: departing(labA("pve1", "pve2"), "pve3"),
			self:   Self{Node: "pve1"},
			obs:    Observed{Cluster: "lab-a", Members: []string{"pve1", "pve2", "pve3"}, Quorate: true},
			verb:   VerbEvict,
			target: "pve3",
		},
		{
			// Same document, different reader. Without this every remaining node
			// issues its own delnode and all but one report a failure against a
			// node that is already gone.
			name:   "a non-owner stays out of the way",
			desire: departing(labA("pve1", "pve2"), "pve3"),
			self:   Self{Node: "pve2"},
			obs:    Observed{Cluster: "lab-a", Members: []string{"pve1", "pve2", "pve3"}, Quorate: true},
			verb:   VerbNone,
		},
		{
			name:   "the departing node waits for a peer to remove it",
			desire: labA("pve1"),
			self:   Self{Node: "pve2"},
			obs:    Observed{Cluster: "lab-a", Members: []string{"pve1", "pve2"}, Quorate: true},
			verb:   VerbWait,
		},
		{
			// 1 -> 0. Nobody remains to run delnode, so the node dissolves itself.
			name:   "the last node dissolves its own cluster",
			desire: Membership{Cluster: "lab-a"},
			self:   Self{Node: "pve1"},
			obs:    Observed{Cluster: "lab-a", Members: []string{"pve1"}, Quorate: true},
			verb:   VerbDissolve,
		},
		{
			name:   "departed and standalone is converged",
			desire: Membership{Cluster: "lab-a"},
			self:   Self{Node: "pve1"},
			obs:    Observed{},
			verb:   VerbNone,
		},
		{
			// Moving between clusters destroys guest configuration, so it is an
			// operator's deliberate departure, never a convergence.
			// BLOCKED, not converged. A move between clusters destroys guest
			// configuration, so it needs an operator's departure — and until
			// then this host must not report the gap as closed.
			name:   "a host in the wrong cluster is blocked, not converged",
			desire: labA("pve1"),
			self:   Self{Node: "pve1"},
			obs:    Observed{Cluster: "lab-b", Members: []string{"pve1"}, Quorate: true},
			verb:   VerbBlocked,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verb, target, why := tc.desire.Decide(tc.self, tc.obs)
			if verb != tc.verb {
				t.Fatalf("verb = %q, want %q (why: %s)", verb, tc.verb, why)
			}
			if target != tc.target {
				t.Fatalf("target = %q, want %q", target, tc.target)
			}
			if verb != VerbNone && why == "" {
				t.Fatalf("verb %q carries no explanation", verb)
			}
		})
	}
}

// The digest gates every agent's decision to act, so it must depend on the
// composition and on nothing else. A row order change from Postgres must not
// move it.
func TestRenderMembershipIsOrderIndependent(t *testing.T) {
	a := []MembershipMember{member("pve3", "c", "10.0.0.3"), member("pve1", "a", "10.0.0.1")}
	b := []MembershipMember{member("pve1", "a", "10.0.0.1"), member("pve3", "c", "10.0.0.3")}

	_, da := RenderMembership("lab-a", a, nil)
	_, db := RenderMembership("lab-a", b, nil)
	if da != db {
		t.Fatalf("digest depends on input order: %s vs %s", da, db)
	}
	if !strings.HasPrefix(da, "sha256:") {
		t.Fatalf("digest %q is not prefixed", da)
	}

	_, dc := RenderMembership("lab-a", a[:1], nil)
	if dc == da {
		t.Fatal("dropping a member did not change the digest")
	}
}

func TestJoinTargetsExcludesSelfAndUnusablePeers(t *testing.T) {
	m := labA("pve1", "pve2")
	m.Members = append(m.Members,
		MembershipMember{Node: "pve4", HypervisorID: "d", Address: "10.0.0.4"},  // no fingerprint
		MembershipMember{Node: "pve5", HypervisorID: "e", Fingerprint: "AA:BB"}, // no address
	)

	got := m.JoinTargets(Self{Node: "pve1"})
	if len(got) != 1 || got[0].Node != "pve2" {
		names := make([]string, 0, len(got))
		for _, g := range got {
			names = append(names, g.Node)
		}
		t.Fatalf("join targets = %v, want [pve2]", names)
	}
}

// A machine that has never been in a cluster has no node name, so its entry in
// the composition carries an empty Node. Matching on names alone would make it
// conclude it is not wanted and never join — which is the state EVERY arrival
// starts from, so the join path would never have run at all.
func TestAMachineWithNoNodeNameStillRecognisesItself(t *testing.T) {
	m := Membership{Cluster: "lab-b", Members: []MembershipMember{
		{Node: "pve3", HypervisorID: "h3", Address: "10.0.0.3", Fingerprint: "AA"},
		{Node: "", HypervisorID: "h9", Address: "10.0.0.9"}, // never been in a cluster
	}}

	verb, _, why := m.Decide(Self{ID: "h9"}, Observed{})
	if verb != VerbJoin {
		t.Fatalf("verb = %q, want join (why: %s)", verb, why)
	}

	// And it must not offer itself as a peer to join through.
	for _, tgt := range m.JoinTargets(Self{ID: "h9"}) {
		if tgt.HypervisorID == "h9" {
			t.Fatal("the joining host was offered as its own join target")
		}
	}
}

// A DISARM IS JUDGED FROM REPORTED STATE, not from the task's exit status.
//
// A task reporting success is the agent's own account of its own work, and the
// whole purpose of a disarm is that certain files stop existing — a claim about
// state is exactly the kind of thing that should be checked against the state.
func TestVerifyDisarm(t *testing.T) {
	rotated := func(v DisarmVerdict) string {
		if v.Rotated == nil {
			return "n/a"
		}
		if *v.Rotated {
			return "yes"
		}
		return "no"
	}

	// Silence is not evidence.
	if v := VerifyDisarm(DisarmInputs{Disposition: "rotate"}); v.Disarmed {
		t.Fatal("a host that has never reported was judged disarmed")
	}

	// Still holding the hive's secrets: not disarmed, whatever the task said.
	v := VerifyDisarm(DisarmInputs{Reported: true, HeldSecrets: 3, Disposition: "accept-exposure"})
	if v.Disarmed {
		t.Fatal("a host still holding the cluster's secrets was judged disarmed")
	}
	if !strings.Contains(v.Detail, "authenticates") {
		t.Fatalf("detail = %q, want it to say what the exposure actually is", v.Detail)
	}

	// Secrets gone, no rotation asked for.
	v = VerifyDisarm(DisarmInputs{Reported: true, Disposition: "accept-exposure",
		ObservedRootKey: "ssh-ed25519 SAME", DepartingKey: "ssh-ed25519 SAME"})
	if !v.Disarmed || v.Rotated != nil {
		t.Fatalf("disarmed=%v rotated=%s, want disarmed with rotation not applicable",
			v.Disarmed, rotated(v))
	}

	// Rotation asked for and the key changed.
	v = VerifyDisarm(DisarmInputs{Reported: true, Disposition: "rotate",
		ObservedRootKey: "ssh-ed25519 NEW", DepartingKey: "ssh-ed25519 OLD"})
	if !v.Disarmed || v.Rotated == nil || !*v.Rotated {
		t.Fatalf("disarmed=%v rotated=%s, want both", v.Disarmed, rotated(v))
	}

	// Rotation asked for and the key did NOT change — the case worth catching.
	v = VerifyDisarm(DisarmInputs{Reported: true, Disposition: "rotate",
		ObservedRootKey: "ssh-ed25519 SAME", DepartingKey: "ssh-ed25519 SAME"})
	if v.Rotated == nil || *v.Rotated {
		t.Fatalf("rotated=%s, want no — the host kept the key it left with", rotated(v))
	}
	if !strings.Contains(v.Detail, "did not happen") {
		t.Fatalf("detail = %q, want it to say the rotation did not happen", v.Detail)
	}
}

// SECRETS GONE IS NOT THE WHOLE OF IT.
//
// A node still carrying a corosync configuration is detached but not converted,
// and it will keep trying to be a member of something. Judging on the secret
// count alone showed a host as "disarmed · rotated" in green while its own
// disarm task had failed saying it was not standalone — the two measuring
// different things, and the operator seeing only the flattering one.
func TestVerifyDisarmAccountsForAnUnfinishedConversion(t *testing.T) {
	v := VerifyDisarm(DisarmInputs{
		Reported: true, HeldSecrets: 0, CarriesClusterConfig: true,
		Disposition: "rotate", ObservedRootKey: "new", DepartingKey: "old",
	})
	if v.Disarmed {
		t.Fatal("a host still carrying a corosync configuration was judged disarmed")
	}
	if !strings.Contains(v.Detail, "corosync") {
		t.Fatalf("detail = %q, want it to name what is left behind", v.Detail)
	}
}

// A FAILED ATTEMPT IS NOT OVERRIDDEN BY LOOKING FINE. The agent's account of what
// stopped it is more specific than anything derivable from state.
func TestVerifyDisarmSurfacesAFailedAttempt(t *testing.T) {
	v := VerifyDisarm(DisarmInputs{
		Reported: true, HeldSecrets: 0,
		LastAttemptFailed: true, LastAttemptError: "corosync.conf still exists",
		Disposition: "accept-exposure",
	})
	if v.Disarmed {
		t.Fatal("a host whose last disarm failed was judged disarmed")
	}
	if !strings.Contains(v.Detail, "corosync.conf still exists") {
		t.Fatalf("detail = %q, want the agent's own reason", v.Detail)
	}

	// And a subsequent success supersedes it: the flag is about the LAST attempt.
	v = VerifyDisarm(DisarmInputs{Reported: true, HeldSecrets: 0, Disposition: "accept-exposure"})
	if !v.Disarmed {
		t.Fatalf("a clean host was not judged disarmed: %q", v.Detail)
	}
}

// THE SECRET COUNT STOPS MEANING WHAT IT APPEARS TO ONCE A DISARM SUCCEEDS.
//
// A disarm ends with `pvecm updatecerts --force`, which regenerates this node's
// OWN certificate authority and ticket key. So the count drops to zero and comes
// straight back up — observed live as a host that "comes back green, then picks
// up 4 credentials again", flapping between disarmed and a false exposure.
//
// The distinction cannot be recovered afterwards: only the disarm sees both the
// before and after bytes, which is why it fingerprints them and refuses on a
// match. That comparison is the proof. Reporting digests instead was rejected
// deliberately — a digest is a stable identifier for the very bytes the disarm
// exists to destroy.
func TestSecretCountIsNotReadAfterASuccessfulDisarm(t *testing.T) {
	// Before any disarm, the files ARE the ones the node left with.
	v := VerifyDisarm(DisarmInputs{
		Reported: true, HeldSecrets: 4, Disposition: "rotate",
	})
	if v.Disarmed {
		t.Fatal("a host that has not been disarmed was judged disarmed")
	}
	if !strings.Contains(v.Detail, "still authenticates") {
		t.Fatalf("detail = %q, want the exposure named", v.Detail)
	}

	// After one succeeds, the same count is this node's own regenerated material.
	v = VerifyDisarm(DisarmInputs{
		Reported: true, HeldSecrets: 4, LastAttemptSucceeded: true,
		Disposition: "rotate", ObservedRootKey: "new", DepartingKey: "old",
	})
	if !v.Disarmed {
		t.Fatalf("a disarmed host was reported as exposed because updatecerts regenerated its "+
			"own credentials: %q", v.Detail)
	}
	if v.Rotated == nil || !*v.Rotated {
		t.Fatalf("rotation was not recognised: %+v", v)
	}

	// And a leftover cluster config still outranks a successful attempt: that is
	// a state fact, not a count that changed meaning.
	v = VerifyDisarm(DisarmInputs{
		Reported: true, HeldSecrets: 4, LastAttemptSucceeded: true,
		CarriesClusterConfig: true, Disposition: "rotate",
	})
	if v.Disarmed {
		t.Fatal("a host still carrying a corosync configuration was judged disarmed")
	}
}
