package repo

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/datatypes"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
)

var membershipTestSchema = testsupport.SchemaName("apiary_repo_membership_test")

func membershipDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, membershipTestSchema,
		&tenant.Hive{}, &tenant.Bee{}, &tenant.Hypervisor{}, &tenant.MembershipSnapshot{},
		&tenant.FleetTask{})
}

func seedMembershipHive(t *testing.T, db *gorm.DB) tenant.Hive {
	t.Helper()
	h := tenant.Hive{ID: uuid.New(), Name: "lab-" + uuid.NewString()[:8], Provider: "proxmox"}
	if err := db.Create(&h).Error; err != nil {
		t.Fatalf("seed hive: %v", err)
	}
	return h
}

func seedNode(t *testing.T, db *gorm.DB, hive *tenant.Hive, node string) tenant.Hypervisor {
	t.Helper()
	h := tenant.Hypervisor{ID: uuid.New(), Address: "10.0.0." + node}
	if hive != nil {
		h.HiveID = &hive.ID
	}
	if node != "" {
		n := node
		h.Name = &n
	}
	if err := db.Create(&h).Error; err != nil {
		t.Fatalf("seed hypervisor %q: %v", node, err)
	}
	return h
}

// A republish of an unchanged composition must not move the revision. Every
// bump makes each member of the hive re-evaluate its cluster state, and
// re-running a cluster operation is not the cheap no-op that re-applying a
// firewall ruleset is.
func TestPublishingAnUnchangedCompositionDoesNotBumpTheRevision(t *testing.T) {
	db := membershipDB(t)
	ctx := context.Background()
	hive := seedMembershipHive(t, db)
	seedNode(t, db, &hive, "pve1")
	seedNode(t, db, &hive, "pve2")

	first, err := PublishHiveMembership(ctx, db, hive.ID)
	if err != nil {
		t.Fatalf("first publish: %v", err)
	}
	if first.Revision != 1 || first.Members != 2 {
		t.Fatalf("first publish = rev %d, %d members; want rev 1, 2", first.Revision, first.Members)
	}

	second, err := PublishHiveMembership(ctx, db, hive.ID)
	if !errors.Is(err, ErrMembershipUnchanged) {
		t.Fatalf("republish err = %v, want ErrMembershipUnchanged", err)
	}
	if second.Revision != first.Revision || second.Digest != first.Digest {
		t.Fatalf("republish moved the pointer: %d/%s -> %d/%s",
			first.Revision, first.Digest, second.Revision, second.Digest)
	}
}

// A departure is a hypervisor leaving the hive, and the composition it produces
// is what every remaining member reconciles against. If this did not bump, no
// node would ever learn to run delnode.
func TestADepartureBumpsTheRevisionAndDropsTheMember(t *testing.T) {
	db := membershipDB(t)
	ctx := context.Background()
	hive := seedMembershipHive(t, db)
	seedNode(t, db, &hive, "pve1")
	leaving := seedNode(t, db, &hive, "pve2")

	first, err := PublishHiveMembership(ctx, db, hive.ID)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", leaving.ID).
		Update("hive_id", nil).Error; err != nil {
		t.Fatalf("depart: %v", err)
	}

	after, err := PublishHiveMembership(ctx, db, hive.ID)
	if err != nil {
		t.Fatalf("republish after departure: %v", err)
	}
	if after.Revision != first.Revision+1 {
		t.Fatalf("revision = %d, want %d", after.Revision, first.Revision+1)
	}
	if after.Members != 1 {
		t.Fatalf("members = %d, want 1", after.Members)
	}

	_, members, _, err := ResolveMembership(ctx, db, hive.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	for _, m := range members {
		if m.Node == "pve2" {
			t.Fatal("the departed node is still in the composition")
		}
	}
}

// A hypervisor whose agent has never reported must still appear. Dropping it
// would tell every other member the node is desired-absent, and a desired-absent
// member gets EVICTED — turning "this agent is not running yet" into "remove it
// from the cluster".
func TestAHypervisorWithNoAgentIsStillAMember(t *testing.T) {
	db := membershipDB(t)
	ctx := context.Background()
	hive := seedMembershipHive(t, db)
	seedNode(t, db, &hive, "pve1")

	_, members, _, err := ResolveMembership(ctx, db, hive.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("members = %d, want 1", len(members))
	}
	if members[0].Fingerprint != "" {
		t.Fatalf("fingerprint = %q, want empty for a node with no agent", members[0].Fingerprint)
	}
	// And it is correctly excluded as a join TARGET, because a join through a
	// peer with no fingerprint prompts and hangs.
	doc, _ := tenant.RenderMembership("lab", members, nil)
	if got := doc.JoinTargets(tenant.Self{Node: "pve9"}); len(got) != 0 {
		t.Fatalf("join targets = %d, want 0 for a member with no fingerprint", len(got))
	}
}

// The FIRST publish has to happen somewhere, and this is where.
//
// The resolve used to return early on revision 0, which put the repair below
// behind a check that bailed before reaching it — so a hive that had never
// published never would. Both lab hives sat at revision 0 through a full restart
// with five agents heartbeating at them, and no agent could converge on anything
// because nothing was ever advertised.
func TestAHiveThatHasNeverPublishedPublishesOnFirstResolve(t *testing.T) {
	db := membershipDB(t)
	ctx := context.Background()
	hive := seedMembershipHive(t, db)
	node := seedNode(t, db, &hive, "pve1")

	bee := tenant.Bee{ID: uuid.New(), APIKeyID: uuid.New()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatalf("seed bee: %v", err)
	}
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", node.ID).
		Update("bee_id", bee.ID).Error; err != nil {
		t.Fatalf("attach bee: %v", err)
	}

	var before tenant.Hive
	if err := db.Where("id = ?", hive.ID).First(&before).Error; err != nil {
		t.Fatalf("read hive: %v", err)
	}
	if before.MembershipRevision != 0 {
		t.Fatalf("precondition: revision = %d, want 0", before.MembershipRevision)
	}

	snap, err := DesiredMembershipForBee(ctx, db, bee.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if snap == nil {
		t.Fatal("an unpublished hive resolved to HOLD; it should have published its first revision")
	}
	if snap.Revision != 1 {
		t.Fatalf("revision = %d, want 1", snap.Revision)
	}
	if snap.Digest == "" || snap.Membership.Cluster != hive.Name {
		t.Fatalf("snapshot = %+v, want the hive's own composition", snap)
	}

	// And it is durable, not just returned: the next heartbeat must advertise it.
	var after tenant.Hive
	if err := db.Where("id = ?", hive.ID).First(&after).Error; err != nil {
		t.Fatalf("re-read hive: %v", err)
	}
	if after.MembershipRevision != 1 || after.MembershipDigest != snap.Digest {
		t.Fatalf("hive = rev %d/%s, want rev 1/%s",
			after.MembershipRevision, after.MembershipDigest, snap.Digest)
	}
}

// A machine in no hive resolves to HOLD, which is the one case that must stay a
// nil snapshot: it is not "you should be in no cluster".
func TestAFleetlessMachineResolvesToHold(t *testing.T) {
	db := membershipDB(t)
	ctx := context.Background()
	node := seedNode(t, db, nil, "")

	bee := tenant.Bee{ID: uuid.New(), APIKeyID: uuid.New()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatalf("seed bee: %v", err)
	}
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", node.ID).
		Update("bee_id", bee.ID).Error; err != nil {
		t.Fatalf("attach bee: %v", err)
	}

	snap, err := DesiredMembershipForBee(ctx, db, bee.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if snap != nil {
		t.Fatalf("a fleetless machine resolved to %+v, want HOLD", snap)
	}
}

// ONE STEP AT A TIME: issue N+1, wait for the acknowledgement, issue N+2.
//
// Acknowledged means REACHED, not received — the agent re-observes and records a
// revision as applied only when observed state equals desired state at it.
func TestRevisionsAreIssuedOneStepAtATime(t *testing.T) {
	db := membershipDB(t)
	ctx := context.Background()
	hive := seedMembershipHive(t, db)
	node := seedNode(t, db, &hive, "pve1")

	bee := tenant.Bee{ID: uuid.New(), APIKeyID: uuid.New()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", node.ID).
		Update("bee_id", bee.ID).Error; err != nil {
		t.Fatal(err)
	}

	// Revision 1.
	first, err := DesiredMembershipForBee(ctx, db, bee.ID)
	if err != nil || first == nil {
		t.Fatalf("first resolve: %v (%v)", err, first)
	}
	if first.Revision != 1 {
		t.Fatalf("first issue = %d, want 1", first.Revision)
	}

	// Two further publications stack up while the agent has acknowledged nothing.
	seedNode(t, db, &hive, "pve2")
	if _, err := PublishHiveMembership(ctx, db, hive.ID); err != nil {
		t.Fatal(err)
	}
	seedNode(t, db, &hive, "pve3")
	if _, err := PublishHiveMembership(ctx, db, hive.ID); err != nil {
		t.Fatal(err)
	}

	// Still 1: nothing advances without an acknowledgement.
	held, err := DesiredMembershipForBee(ctx, db, bee.ID)
	if err != nil || held == nil {
		t.Fatalf("resolve in flight: %v (%v)", err, held)
	}
	if held.Revision != first.Revision || held.Digest != first.Digest {
		t.Fatalf("advanced to %d while %d was in flight", held.Revision, first.Revision)
	}

	// Acknowledge 1 -> issued 2, NOT the latest 3. And it carries revision 2's
	// own body: one member, as published at that step.
	if err := RecordAppliedMembership(ctx, db, bee.ID, first.Revision, first.Digest); err != nil {
		t.Fatal(err)
	}
	second, err := DesiredMembershipForBee(ctx, db, bee.ID)
	if err != nil || second == nil {
		t.Fatalf("resolve after ack: %v (%v)", err, second)
	}
	if second.Revision != 2 {
		t.Fatalf("issued %d after acknowledging 1, want 2 — steps must not be skipped",
			second.Revision)
	}
	if len(second.Membership.Members) != 2 {
		t.Fatalf("revision 2 served %d members, want the 2 it was published with — serving "+
			"today's members under an older number makes the agent report a digest nobody "+
			"issued", len(second.Membership.Members))
	}

	// Acknowledge 2 -> issued 3.
	if err := RecordAppliedMembership(ctx, db, bee.ID, second.Revision, second.Digest); err != nil {
		t.Fatal(err)
	}
	third, err := DesiredMembershipForBee(ctx, db, bee.ID)
	if err != nil || third == nil {
		t.Fatalf("resolve after second ack: %v (%v)", err, third)
	}
	if third.Revision != 3 {
		t.Fatalf("issued %d after acknowledging 2, want 3", third.Revision)
	}
	if len(third.Membership.Members) != 3 {
		t.Fatalf("revision 3 served %d members, want 3", len(third.Membership.Members))
	}

	// Converged: the last step stays put.
	if err := RecordAppliedMembership(ctx, db, bee.ID, third.Revision, third.Digest); err != nil {
		t.Fatal(err)
	}
	settled, err := DesiredMembershipForBee(ctx, db, bee.ID)
	if err != nil || settled == nil {
		t.Fatalf("resolve when converged: %v (%v)", err, settled)
	}
	if settled.Revision != 3 {
		t.Fatalf("issued %d when already at the latest, want 3", settled.Revision)
	}
}

// A STEP THE HOST CANNOT REACH IS SKIPPED, not re-served forever.
//
// Observed live: lab-b went {pve3,pve4,pve5} at 17, {pve3,pve4} at 18, and back
// to {pve3,pve4,pve5} by 21. pve5 applied 17, was issued 18 — where it is
// desired-absent while sitting in the cluster with peers — and Decide answered
// VerbWait, because a departing node cannot remove itself. Not VerbNone, so it
// never acknowledged, and 18 was re-served indefinitely. The agent was correct
// and the step was impossible.
func TestAnUnreachableStepIsSkippedRatherThanDeadlocking(t *testing.T) {
	db := membershipDB(t)
	ctx := context.Background()
	hive := seedMembershipHive(t, db)

	a := seedNode(t, db, &hive, "pve3")
	b := seedNode(t, db, &hive, "pve4")
	me := seedNode(t, db, &hive, "pve5")

	bee := tenant.Bee{ID: uuid.New(), APIKeyID: uuid.New()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", me.ID).
		Update("bee_id", bee.ID).Error; err != nil {
		t.Fatal(err)
	}
	// It reports what it actually sees: in lab-b, quorate, all three present.
	now := time.Now()
	if err := db.Model(&tenant.Bee{}).Where("id = ?", bee.ID).Updates(map[string]any{
		"observed_cluster":       hive.Name,
		"observed_members":       datatypes.NewJSONType([]string{"pve3", "pve4", "pve5"}),
		"observed_quorate":       true,
		"observed_membership_at": now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	_ = a
	_ = b

	// Revision 1: all three.
	first, err := DesiredMembershipForBee(ctx, db, bee.ID)
	if err != nil || first == nil {
		t.Fatalf("first resolve: %v (%v)", err, first)
	}
	if err := RecordAppliedMembership(ctx, db, bee.ID, first.Revision, first.Digest); err != nil {
		t.Fatal(err)
	}

	// Revision 2 drops pve5 — the withdrawn state it can never reach.
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", me.ID).
		Update("hive_id", nil).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := PublishHiveMembership(ctx, db, hive.ID); err != nil {
		t.Fatal(err)
	}
	// Revision 3 puts it back.
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", me.ID).
		Update("hive_id", hive.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := PublishHiveMembership(ctx, db, hive.ID); err != nil {
		t.Fatal(err)
	}

	got, err := DesiredMembershipForBee(ctx, db, bee.ID)
	if err != nil || got == nil {
		t.Fatalf("resolve after the withdrawn revision: %v (%v)", err, got)
	}
	if got.Revision == first.Revision+1 {
		t.Fatalf("issued the withdrawn revision %d, which this host can never acknowledge — "+
			"it is desired-absent there while sitting in the cluster with peers, so Decide "+
			"answers VerbWait and the step is re-served forever", got.Revision)
	}
	// It landed on a step it CAN act on.
	verb, _, _ := got.Membership.Decide(
		tenant.Self{ID: me.ID.String(), Node: "pve5"},
		tenant.Observed{Cluster: hive.Name, Members: []string{"pve3", "pve4", "pve5"}, Quorate: true})
	if verb == tenant.VerbWait || verb == tenant.VerbBlocked {
		t.Fatalf("issued revision %d yields %q, which only another actor can resolve",
			got.Revision, verb)
	}
}

// ARMING EVICTION: a departing machine keeps its hive and name so the
// composition can NAME it, which is the only thing that authorises a peer to
// run `pvecm delnode`.
//
// Clearing the row on the operator's click left nothing to name, which is why
// convergent eviction was built and never armed — Departing was always empty and
// absence from the member list is deliberately not enough.
func TestADepartingMachineIsNamedSoAPeerCanEvictIt(t *testing.T) {
	db := membershipDB(t)
	ctx := context.Background()
	hive := seedMembershipHive(t, db)
	stay := seedNode(t, db, &hive, "pve1")
	leave := seedNode(t, db, &hive, "pve2")

	if err := MarkDeparting(ctx, db, leave.ID, "accept-exposure"); err != nil {
		t.Fatalf("mark departing: %v", err)
	}

	name, members, departing, err := ResolveMembership(ctx, db, hive.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(members) != 1 || members[0].Node != "pve1" {
		t.Fatalf("members = %+v, want only pve1", members)
	}
	if len(departing) != 1 || departing[0] != "pve2" {
		t.Fatalf("departing = %v, want [pve2] — a peer cannot evict a node nobody published",
			departing)
	}

	// And the remaining member now decides to evict it, which it would not do on
	// absence alone.
	doc, _ := tenant.RenderMembership(name, members, departing)
	verb, target, _ := doc.Decide(
		tenant.Self{ID: stay.ID.String(), Node: "pve1"},
		tenant.Observed{Cluster: name, Members: []string{"pve1", "pve2"}, Quorate: true})
	if verb != tenant.VerbEvict || target != "pve2" {
		t.Fatalf("verb = %q target = %q, want evict pve2", verb, target)
	}
}

// The record is cleared on evidence from a REMAINING member, never from the
// departing node — after delnode it still believes it is a member, which is why
// it must not be powered back on.
func TestADepartureIsFinalisedOnlyOnAPeersObservation(t *testing.T) {
	db := membershipDB(t)
	ctx := context.Background()
	hive := seedMembershipHive(t, db)
	stay := seedNode(t, db, &hive, "pve1")
	leave := seedNode(t, db, &hive, "pve2")

	bee := tenant.Bee{ID: uuid.New(), APIKeyID: uuid.New()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", stay.ID).
		Update("bee_id", bee.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := MarkDeparting(ctx, db, leave.ID, "accept-exposure"); err != nil {
		t.Fatal(err)
	}

	// The peer still sees it: nothing is finalised.
	obs := func(members []string) {
		t.Helper()
		if err := db.Model(&tenant.Bee{}).Where("id = ?", bee.ID).Updates(map[string]any{
			"observed_members":       datatypes.NewJSONType(members),
			"observed_quorate":       true,
			"observed_membership_at": time.Now(),
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	obs([]string{"pve1", "pve2"})
	if n, err := FinaliseDepartures(ctx, db, hive.ID); err != nil || n != 0 {
		t.Fatalf("finalised %d (%v) while the peer still lists it", n, err)
	}
	var mid tenant.Hypervisor
	if err := db.First(&mid, "id = ?", leave.ID).Error; err != nil {
		t.Fatal(err)
	}
	if mid.HiveID == nil {
		t.Fatal("membership was cleared before the cluster dropped the node")
	}

	// The peer no longer lists it: now it is finalised.
	obs([]string{"pve1"})
	if n, err := FinaliseDepartures(ctx, db, hive.ID); err != nil || n != 1 {
		t.Fatalf("finalised %d (%v), want 1", n, err)
	}
	var done tenant.Hypervisor
	if err := db.First(&done, "id = ?", leave.ID).Error; err != nil {
		t.Fatal(err)
	}
	if done.HiveID != nil || done.Name != nil || done.DepartingAt != nil {
		t.Fatalf("row not cleared: hive=%v name=%v departing=%v",
			done.HiveID, done.Name, done.DepartingAt)
	}
}

// THE DISARM IS QUEUED WHEN THE CLUSTER HAS ACTUALLY DROPPED THE NODE, carrying
// the disposition the operator chose and the peers whose configuration the host
// is still holding.
//
// Not before: disarming a machine that is still a member destroys secrets it is
// still using. And not over SSH — writing files and restarting services on a
// host is local work, and doing it from the control plane made its own
// reachability a precondition of closing a credential exposure.
func TestFinalisingADepartureQueuesTheDisarm(t *testing.T) {
	db := membershipDB(t)
	ctx := context.Background()
	hive := seedMembershipHive(t, db)
	stay := seedNode(t, db, &hive, "pve1")
	leave := seedNode(t, db, &hive, "pve2")

	witness := tenant.Bee{ID: uuid.New(), APIKeyID: uuid.New()}
	departing := tenant.Bee{ID: uuid.New(), APIKeyID: uuid.New()}
	for _, b := range []tenant.Bee{witness, departing} {
		if err := db.Create(&b).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", stay.ID).
		Update("bee_id", witness.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", leave.ID).
		Update("bee_id", departing.ID).Error; err != nil {
		t.Fatal(err)
	}

	if err := MarkDeparting(ctx, db, leave.ID, "rotate"); err != nil {
		t.Fatal(err)
	}
	// The peer no longer lists it: the cluster has dropped it.
	if err := db.Model(&tenant.Bee{}).Where("id = ?", witness.ID).Updates(map[string]any{
		"observed_members":       datatypes.NewJSONType([]string{"pve1"}),
		"observed_quorate":       true,
		"observed_membership_at": time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if n, err := FinaliseDepartures(ctx, db, hive.ID); err != nil || n != 1 {
		t.Fatalf("finalised %d (%v), want 1", n, err)
	}

	var task tenant.FleetTask
	if err := db.Where("bee_id = ? AND kind = ?", departing.ID, KindDisarm).
		First(&task).Error; err != nil {
		t.Fatalf("no disarm was queued for the departed host, so it keeps the hive's "+
			"secrets: %v", err)
	}
	// Decoded rather than string-matched: jsonb normalises whitespace, so the
	// text form is not the contract.
	var spec struct {
		Disposition string   `json:"disposition"`
		Peers       []string `json:"peers"`
	}
	if err := json.Unmarshal(task.Spec, &spec); err != nil {
		t.Fatalf("spec %s: %v", task.Spec, err)
	}
	if spec.Disposition != "rotate" {
		t.Fatalf("disposition = %q, want the operator's chosen one", spec.Disposition)
	}
	if len(spec.Peers) != 1 || spec.Peers[0] != "pve1" {
		t.Fatalf("peers = %v, want the peer whose configuration this host still carries",
			spec.Peers)
	}
}
