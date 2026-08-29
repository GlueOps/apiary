package fleetops

import (
	"testing"
	"time"
)

// swapSettle shortens the join settle window for one test and restores it.
func swapSettle(t *testing.T, timeout, interval time.Duration) func() {
	t.Helper()
	ot, oi := joinSettleTimeout, joinPollInterval
	joinSettleTimeout, joinPollInterval = timeout, interval
	return func() { joinSettleTimeout, joinPollInterval = ot, oi }
}

// departing marks a node as one the control plane positively removed, which is
// the only thing that authorises an eviction.
func departing(d Desired, nodes ...string) Desired {
	d.Departing = append(d.Departing, nodes...)
	return d
}

// A node in corosync the control plane has no record of must NOT be evicted.
func TestAnUnrecordedNodeIsReportedNotEvicted(t *testing.T) {
	d := want("lab-a", "pve1", "pve1", "pve2")
	obs := Observation{Cluster: "lab-a", Members: []string{"pve1", "pve2", "pve9"}, Quorate: true}
	verb, target, why := d.Decide(obs)
	if verb != VerbNone {
		t.Fatalf("verb = %q (target %q), want none for an unrecorded node", verb, target)
	}
	if why == "" {
		t.Fatal("an unrecorded node was ignored silently; it should be reported")
	}
}

func want(cluster, self string, nodes ...string) Desired {
	d := Desired{Cluster: cluster, Self: self}
	for _, n := range nodes {
		d.Members = append(d.Members, DesiredMember{
			Node: n, HypervisorID: n, Address: "10.0.0." + n, Fingerprint: "AA:BB",
		})
	}
	return d
}

// The SAME table the control plane's tenant.Membership.Decide is tested against.
//
// The two implementations are deliberately separate — the agent module does not
// import the control plane — so nothing but this makes them agree. A case added
// to one and not the other is the drift this exists to catch: the control plane
// would gate on one set of rules while the host acted on another, and the
// disagreement would only ever show up on live metal.
func TestDecideMatchesTheControlPlaneTable(t *testing.T) {
	cases := []struct {
		name   string
		desire Desired
		obs    Observation
		verb   Verb
		target string
	}{
		{
			name:   "desired in and standalone joins",
			desire: want("lab-a", "pve2", "pve1", "pve2"),
			obs:    Observation{},
			verb:   VerbJoin,
		},
		{
			name:   "converged does nothing",
			desire: want("lab-a", "pve2", "pve1", "pve2"),
			obs:    Observation{Cluster: "lab-a", Members: []string{"pve1", "pve2"}, Quorate: true},
			verb:   VerbNone,
		},
		{
			name:   "lowest remaining member evicts a recorded departure",
			desire: departing(want("lab-a", "pve1", "pve1", "pve2"), "pve3"),
			obs:    Observation{Cluster: "lab-a", Members: []string{"pve1", "pve2", "pve3"}, Quorate: true},
			verb:   VerbEvict,
			target: "pve3",
		},
		{
			name:   "a non-owner stays out of the way",
			desire: departing(want("lab-a", "pve2", "pve1", "pve2"), "pve3"),
			obs:    Observation{Cluster: "lab-a", Members: []string{"pve1", "pve2", "pve3"}, Quorate: true},
			verb:   VerbNone,
		},
		{
			name:   "the departing node waits for a peer to remove it",
			desire: want("lab-a", "pve2", "pve1"),
			obs:    Observation{Cluster: "lab-a", Members: []string{"pve1", "pve2"}, Quorate: true},
			verb:   VerbWait,
		},
		{
			name:   "the last node needs a departure, not a convergence",
			desire: want("lab-a", "pve1"),
			obs:    Observation{Cluster: "lab-a", Members: []string{"pve1"}, Quorate: true},
			verb:   VerbDissolve,
		},
		{
			name:   "departed and standalone is converged",
			desire: want("lab-a", "pve1"),
			obs:    Observation{},
			verb:   VerbNone,
		},
		{
			name:   "a host in the wrong cluster is not converged automatically",
			desire: want("lab-a", "pve1", "pve1"),
			obs:    Observation{Cluster: "lab-b", Members: []string{"pve1"}, Quorate: true},
			verb:   VerbNone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verb, target, why := tc.desire.Decide(tc.obs)
			if verb != tc.verb {
				t.Fatalf("verb = %q, want %q (why: %s)", verb, tc.verb, why)
			}
			if target != tc.target {
				t.Fatalf("target = %q, want %q", target, tc.target)
			}
		})
	}
}

// A standalone host cannot see whether its cluster exists, so it must ASK every
// published peer before forming one. Creating first would split a cluster in two
// the moment two hosts converged at once — both finding no peer, both creating.
func TestJoinIsTriedAgainstEveryPeerBeforeCreating(t *testing.T) {
	// The fake never becomes a member, so the post-create verification would
	// otherwise poll for its full four-minute settle window. These vars exist to
	// be shortened; the wait is the point of the real one, not of this one.
	defer swapSettle(t, 200*time.Millisecond, 10*time.Millisecond)()

	cl := &fakeCluster{joinErr: errRefused{}}
	d := Deps{
		Cluster: cl, Node: "pve3",
		JoinPass: func() (string, error) { return "pw", nil },
	}
	var phases []string
	e := testEnv(&phases, nil, nil)

	w := want("lab-a", "pve3", "pve1", "pve2", "pve3")
	verb, err := joinOrCreate(t.Context(), d, e, w)

	if cl.joinAttempts != 2 {
		t.Fatalf("join attempts = %d, want 2 (one per published peer)", cl.joinAttempts)
	}
	if verb != VerbCreate {
		t.Fatalf("verb = %q, want create once every peer refused", verb)
	}
	if cl.created == nil {
		t.Fatal("no cluster was created after every peer refused")
	}
	if cl.created.ClusterName != "lab-a" {
		t.Fatalf("created %q, want lab-a", cl.created.ClusterName)
	}
	_ = err
}

// A peer with no fingerprint is not a usable join target: pvecm accepts a
// missing one by PROMPTING, which non-interactively hangs until something kills
// it and reports a timeout that says nothing about the cause.
func TestPeersWithoutAFingerprintAreNotJoinTargets(t *testing.T) {
	d := want("lab-a", "pve1", "pve1", "pve2")
	d.Members = append(d.Members,
		DesiredMember{Node: "pve4", Address: "10.0.0.4"},
		DesiredMember{Node: "pve5", Fingerprint: "AA:BB"},
	)
	got := d.joinTargets()
	if len(got) != 1 || got[0].Node != "pve2" {
		t.Fatalf("join targets = %+v, want just pve2", got)
	}
}

type errRefused struct{}

func (errRefused) Error() string { return "peer refused the join" }
