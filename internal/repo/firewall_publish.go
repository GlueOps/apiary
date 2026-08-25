package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// ErrNothingToPublish reports that the rendered policy is identical to the one
// already published.
//
// Not an error condition so much as a refusal to move the revision for nothing.
// Every bump makes the whole fleet re-apply, and re-applying an identical
// ruleset is a flush and reload of every host's firewall to reach the state it
// was already in — brief, real, and completely pointless. Republishing unchanged
// rules is also the ordinary consequence of clicking Publish twice.
var ErrNothingToPublish = errors.New("firewall: the rendered policy is identical to the published one")

// PublishResult is what a publish did.
type PublishResult struct {
	Revision int
	Digest   string
	// Bytes of the rendered body, so a caller can say how large a policy it just
	// committed the fleet to without holding the body.
	Bytes int
}

// PublishHiveFirewall renders one hive's policy and commits it as the next
// revision.
//
// ONE TRANSACTION, which the model requires rather than prefers: publishing
// writes the snapshot AND moves the hive's pointer at it, and a failure between
// the two would leave the pointer naming a snapshot that does not exist — every
// bee in the hive asking for a body nobody can serve. Both rows live in this
// database precisely so one transaction can cover them.
//
// The revision is read INSIDE the transaction with a row lock. Two operators
// publishing at once would otherwise read the same current revision, compute the
// same next one, and collide on the unique index — or, worse, if the index were
// ever relaxed, produce two snapshots claiming to be the same revision with
// different bodies.
//
// Rendering happens inside too. The rules could change between a render outside
// and the write, and the snapshot would then record a body that no set of rows
// ever produced.
func PublishHiveFirewall(
	ctx context.Context,
	db *gorm.DB,
	hiveID uuid.UUID,
	byUser *uuid.UUID,
	reason string,
	resolve func(ctx context.Context, tx *gorm.DB, hiveID uuid.UUID) (tenant.RenderInput, error),
) (PublishResult, error) {
	var out PublishResult

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var hive tenant.Hive
		// FOR UPDATE: the revision is derived from this row, so reading it without
		// the lock lets two publishes derive the same next revision.
		if err := tx.Raw(`SELECT * FROM hives WHERE id = ? FOR UPDATE`, hiveID).
			Scan(&hive).Error; err != nil {
			return fmt.Errorf("publish: lock the hive: %w", err)
		}
		if hive.ID == uuid.Nil {
			return gorm.ErrRecordNotFound
		}

		in, err := resolve(ctx, tx, hiveID)
		if err != nil {
			return err
		}
		rendered, err := tenant.Render(in)
		if err != nil {
			return err
		}

		// Compared against the DIGEST, not the revision. A republish of unchanged
		// rules is the ordinary result of clicking twice, and bumping for it makes
		// the whole fleet flush and reload its firewall to reach the state it was
		// already in.
		if hive.FirewallRevision > 0 && hive.FirewallDigest == rendered.Digest {
			return ErrNothingToPublish
		}

		next := hive.FirewallRevision + 1
		snap := tenant.FirewallSnapshot{
			HiveID:      hiveID,
			Revision:    next,
			Content:     rendered.Body,
			Digest:      rendered.Digest,
			CreatedByID: byUser,
			Reason:      reason,
		}
		if err := tx.Create(&snap).Error; err != nil {
			return fmt.Errorf("publish: write the snapshot: %w", err)
		}

		// The pointer moves only after the body it names exists.
		if err := tx.Model(&tenant.Hive{}).Where("id = ?", hiveID).
			Updates(map[string]any{
				"firewall_revision":     next,
				"firewall_digest":       rendered.Digest,
				"firewall_published_at": gorm.Expr("now()"),
			}).Error; err != nil {
			return fmt.Errorf("publish: move the hive's pointer: %w", err)
		}

		out = PublishResult{Revision: next, Digest: rendered.Digest, Bytes: len(rendered.Body)}
		return nil
	})
	return out, err
}

// RevertHiveFirewall publishes an OLD revision's body as a NEW revision.
//
// Never a move backwards. A bee decides whether to converge by comparing what it
// applied against the hive's revision, and the two obvious comparisons disagree
// when the number goes down: `applied != desired` re-applies, while
// `applied < desired` does nothing and leaves the host pinned to the very
// ruleset being backed out. Only ever moving up makes both readings behave
// identically, so no agent — including an older one already deployed — can hold
// a wrong opinion about it.
//
// RevertOf records which revision this reproduces, so the history says "this was
// a backout" rather than leaving someone to diff bodies and infer it.
func RevertHiveFirewall(
	ctx context.Context, db *gorm.DB, hiveID uuid.UUID, to int, byUser *uuid.UUID, reason string,
) (PublishResult, error) {
	var out PublishResult

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var hive tenant.Hive
		if err := tx.Raw(`SELECT * FROM hives WHERE id = ? FOR UPDATE`, hiveID).
			Scan(&hive).Error; err != nil {
			return fmt.Errorf("revert: lock the hive: %w", err)
		}
		if hive.ID == uuid.Nil {
			return gorm.ErrRecordNotFound
		}

		var old tenant.FirewallSnapshot
		if err := tx.First(&old, "hive_id = ? AND revision = ?", hiveID, to).Error; err != nil {
			return err
		}

		next := hive.FirewallRevision + 1
		revertOf := to
		snap := tenant.FirewallSnapshot{
			HiveID: hiveID, Revision: next,
			// The old BODY verbatim, not a re-render of the old rules. The rules have
			// moved on — that is why this is a backout — and re-rendering them would
			// reproduce the state being backed out.
			Content:     old.Content,
			Digest:      old.Digest,
			RevertOf:    &revertOf,
			CreatedByID: byUser,
			Reason:      reason,
		}
		if err := tx.Create(&snap).Error; err != nil {
			return fmt.Errorf("revert: write the snapshot: %w", err)
		}
		if err := tx.Model(&tenant.Hive{}).Where("id = ?", hiveID).
			Updates(map[string]any{
				"firewall_revision":     next,
				"firewall_digest":       old.Digest,
				"firewall_published_at": gorm.Expr("now()"),
			}).Error; err != nil {
			return fmt.Errorf("revert: move the hive's pointer: %w", err)
		}

		out = PublishResult{Revision: next, Digest: old.Digest, Bytes: len(old.Content)}
		return nil
	})
	return out, err
}

// DesiredFirewall returns what a bee in this hive should be running.
//
// A hive with revision 0 has published NOTHING, which is not the same as an
// empty ruleset — see Bee.AppliedFirewallRevision. The caller must render that
// as "hold what you have", never as "apply this empty body", or enrolling a host
// into a fleet nobody has published for would flush whatever firewall it already
// had.
func DesiredFirewall(
	ctx context.Context, db *gorm.DB, hiveID uuid.UUID,
) (*tenant.FirewallSnapshot, error) {
	var hive tenant.Hive
	if err := db.WithContext(ctx).First(&hive, "id = ?", hiveID).Error; err != nil {
		return nil, err
	}
	if hive.FirewallRevision == 0 {
		return nil, nil
	}
	var snap tenant.FirewallSnapshot
	if err := db.WithContext(ctx).
		First(&snap, "hive_id = ? AND revision = ?", hiveID, hive.FirewallRevision).Error; err != nil {
		return nil, fmt.Errorf("desired firewall: the hive names revision %d and no snapshot "+
			"holds it: %w", hive.FirewallRevision, err)
	}
	return &snap, nil
}
