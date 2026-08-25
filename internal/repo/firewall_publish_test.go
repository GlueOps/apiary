package repo

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
)

var publishTestSchema = testsupport.SchemaName("apiary_repo_firewall_publish_test")

func publishDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, publishTestSchema,
		&tenant.Hive{}, &tenant.FirewallSnapshot{})
}

func seedPublishHive(t *testing.T, db *gorm.DB) tenant.Hive {
	t.Helper()
	h := tenant.Hive{ID: uuid.New(), Name: "lab", Provider: "proxmox"}
	if err := db.Create(&h).Error; err != nil {
		t.Fatalf("seed hive: %v", err)
	}
	return h
}

// rules returns a resolver yielding a fixed policy, so the tests are about
// publishing rather than about rule resolution.
func rules(names ...string) func(context.Context, *gorm.DB, uuid.UUID) (tenant.RenderInput, error) {
	return func(context.Context, *gorm.DB, uuid.UUID) (tenant.RenderInput, error) {
		in := tenant.RenderInput{}
		for _, n := range names {
			in.Rules = append(in.Rules, tenant.FirewallRule{
				Name: n, Enabled: true, Family: "inet", TableName: "filter",
				ChainName: "input", Protocol: tenant.TCP, Action: tenant.Accept, DstPort: 22,
			})
		}
		return in, nil
	}
}

// TestPublishingWritesTheBodyAndMovesThePointerTogether.
//
// The pointer naming a snapshot that does not exist is the failure this
// transaction exists to prevent: every bee in the hive would ask for a body
// nobody could serve.
func TestPublishingWritesTheBodyAndMovesThePointerTogether(t *testing.T) {
	db := publishDB(t)
	ctx := context.Background()
	h := seedPublishHive(t, db)

	res, err := PublishHiveFirewall(ctx, db, h.ID, nil, "first", rules("ssh"))
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if res.Revision != 1 {
		t.Fatalf("revision = %d, want 1", res.Revision)
	}

	var after tenant.Hive
	if err := db.First(&after, "id = ?", h.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.FirewallRevision != 1 || after.FirewallDigest != res.Digest {
		t.Fatalf("hive points at revision %d digest %q, want 1 and %q",
			after.FirewallRevision, after.FirewallDigest, res.Digest)
	}
	if after.FirewallPublishedAt == nil {
		t.Fatal("nothing recorded when the policy was published")
	}

	var snap tenant.FirewallSnapshot
	if err := db.First(&snap, "hive_id = ? AND revision = 1", h.ID).Error; err != nil {
		t.Fatalf("the pointer names a snapshot that does not exist: %v", err)
	}
	if !strings.Contains(snap.Content, "flush ruleset") {
		t.Fatalf("the stored body is not a renderable ruleset:\n%s", snap.Content)
	}
}

// TestRepublishingUnchangedRulesDoesNotMoveTheRevision.
//
// Every bump makes the whole fleet re-apply, and re-applying an identical
// ruleset flushes and reloads every host's firewall to reach the state it was
// already in. Clicking Publish twice is the ordinary way to hit this.
func TestRepublishingUnchangedRulesDoesNotMoveTheRevision(t *testing.T) {
	db := publishDB(t)
	ctx := context.Background()
	h := seedPublishHive(t, db)

	if _, err := PublishHiveFirewall(ctx, db, h.ID, nil, "first", rules("ssh")); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	_, err := PublishHiveFirewall(ctx, db, h.ID, nil, "again", rules("ssh"))
	if !errors.Is(err, ErrNothingToPublish) {
		t.Fatalf("second publish returned %v, want ErrNothingToPublish", err)
	}

	var after tenant.Hive
	_ = db.First(&after, "id = ?", h.ID).Error
	if after.FirewallRevision != 1 {
		t.Fatalf("revision moved to %d for an unchanged policy", after.FirewallRevision)
	}
}

// TestAChangedPolicyPublishesTheNextRevision.
func TestAChangedPolicyPublishesTheNextRevision(t *testing.T) {
	db := publishDB(t)
	ctx := context.Background()
	h := seedPublishHive(t, db)

	first, err := PublishHiveFirewall(ctx, db, h.ID, nil, "", rules("ssh"))
	if err != nil {
		t.Fatalf("publish 1: %v", err)
	}
	second, err := PublishHiveFirewall(ctx, db, h.ID, nil, "", rules("ssh", "https"))
	if err != nil {
		t.Fatalf("publish 2: %v", err)
	}
	if second.Revision != 2 {
		t.Fatalf("revision = %d, want 2", second.Revision)
	}
	if second.Digest == first.Digest {
		t.Fatal("a changed policy produced the same digest, so no bee would re-apply it")
	}
}

// TestARevertGoesFORWARDSCarryingTheOldBody.
//
// The direction is the whole point. A bee compares what it applied against the
// hive's revision, and the two obvious comparisons disagree when the number goes
// down: `applied != desired` re-applies, `applied < desired` does nothing and
// leaves the host pinned to the ruleset being backed out — the one case where
// the backout mattered.
func TestARevertGoesFORWARDSCarryingTheOldBody(t *testing.T) {
	db := publishDB(t)
	ctx := context.Background()
	h := seedPublishHive(t, db)

	first, err := PublishHiveFirewall(ctx, db, h.ID, nil, "", rules("ssh"))
	if err != nil {
		t.Fatalf("publish 1: %v", err)
	}
	if _, err := PublishHiveFirewall(ctx, db, h.ID, nil, "", rules("ssh", "oops")); err != nil {
		t.Fatalf("publish 2: %v", err)
	}

	back, err := RevertHiveFirewall(ctx, db, h.ID, 1, nil, "rolling back")
	if err != nil {
		t.Fatalf("revert: %v", err)
	}
	if back.Revision != 3 {
		t.Fatalf("revert produced revision %d; it must go FORWARDS, or an agent comparing "+
			"applied < desired stays on the ruleset being backed out", back.Revision)
	}
	if back.Digest != first.Digest {
		t.Fatal("the revert did not reproduce revision 1's body")
	}

	var snap tenant.FirewallSnapshot
	_ = db.First(&snap, "hive_id = ? AND revision = 3", h.ID).Error
	if snap.RevertOf == nil || *snap.RevertOf != 1 {
		t.Fatal("the snapshot does not record which revision it reproduces, so the history " +
			"leaves someone to diff bodies and infer it")
	}
}

// TestNothingPublishedIsNotAnEmptyRuleset.
//
// A hive at revision 0 has published nothing. Rendering that as an empty body
// would flush whatever firewall a host already had the moment it joined a fleet
// nobody had published for.
func TestNothingPublishedIsNotAnEmptyRuleset(t *testing.T) {
	db := publishDB(t)
	ctx := context.Background()
	h := seedPublishHive(t, db)

	got, err := DesiredFirewall(ctx, db, h.ID)
	if err != nil {
		t.Fatalf("desired: %v", err)
	}
	if got != nil {
		t.Fatalf("an unpublished hive returned a body (%d bytes); an agent would apply it "+
			"and flush the firewall the host already had", len(got.Content))
	}

	if _, err := PublishHiveFirewall(ctx, db, h.ID, nil, "", rules("ssh")); err != nil {
		t.Fatalf("publish: %v", err)
	}
	got, err = DesiredFirewall(ctx, db, h.ID)
	if err != nil {
		t.Fatalf("desired after publish: %v", err)
	}
	if got == nil || got.Revision != 1 {
		t.Fatalf("desired = %+v, want revision 1", got)
	}
}

// TestConcurrentPublishesDoNotCollideOnARevision. Two operators publishing at
// once would otherwise read the same current revision and compute the same next
// one.
func TestConcurrentPublishesDoNotCollideOnARevision(t *testing.T) {
	db := publishDB(t)
	ctx := context.Background()
	h := seedPublishHive(t, db)

	type result struct {
		rev int
		err error
	}
	done := make(chan result, 2)
	for i, policy := range [][]string{{"a"}, {"b"}} {
		go func(names []string) {
			res, err := PublishHiveFirewall(ctx, db, h.ID, nil, "", rules(names...))
			done <- result{res.Revision, err}
		}(policy)
		_ = i
	}

	revs := map[int]bool{}
	for i := 0; i < 2; i++ {
		r := <-done
		if r.err != nil {
			t.Fatalf("concurrent publish failed: %v", r.err)
		}
		if revs[r.rev] {
			t.Fatalf("two publishes both claimed revision %d", r.rev)
		}
		revs[r.rev] = true
	}
	if !revs[1] || !revs[2] {
		t.Fatalf("revisions were %v, want 1 and 2", revs)
	}
}
