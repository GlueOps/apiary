package repo

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
)

var membershipTestSchema = testsupport.SchemaName("apiary_repo_membership_test")

func membershipDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, membershipTestSchema,
		&tenant.Hive{}, &tenant.Bee{}, &tenant.Hypervisor{})
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

	_, members, err := ResolveMembership(ctx, db, hive.ID)
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

	_, members, err := ResolveMembership(ctx, db, hive.ID)
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
	doc, _ := tenant.RenderMembership("lab", members)
	if got := doc.JoinTargets("pve9"); len(got) != 0 {
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
