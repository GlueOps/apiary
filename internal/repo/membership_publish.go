package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// ErrMembershipUnchanged reports that the rendered composition matches the one
// already published.
//
// Returned rather than silently ignored so a caller can tell "published nothing
// because nothing changed" from "published". It matters more here than for the
// firewall: a bumped revision makes every member of the hive re-evaluate its
// cluster state, and re-running a cluster operation is not the cheap no-op that
// re-applying a firewall ruleset is.
var ErrMembershipUnchanged = errors.New("membership: the composition is already published")

// MembershipResult is what a publish did.
type MembershipResult struct {
	Revision int
	Digest   string
	Members  int
}

// ResolveMembership builds one hive's desired composition from the hypervisors
// assigned to it.
//
// DERIVED, never stored as its own list. A second list of who should be in a
// cluster is a second thing to keep in step with hypervisors.hive_id, and the
// two disagreeing is exactly the class of bug that made hv5 belong to hive A and
// datacenter B at once. Assignment IS the desired state; this only renders it.
//
// The fingerprint and the address come from different rows on purpose: the
// address is a fact about the machine and lives on the hypervisor, while the
// fingerprint is what the host's own agent last reported about the certificate
// it is actually serving. A stale fingerprint from a node whose certificate was
// reissued would fail a join with a mismatch, which is the correct outcome and a
// legible one.
func ResolveMembership(ctx context.Context, tx *gorm.DB, hiveID uuid.UUID) (string, []tenant.MembershipMember, error) {
	var hive tenant.Hive
	if err := tx.WithContext(ctx).Where("id = ?", hiveID).First(&hive).Error; err != nil {
		return "", nil, fmt.Errorf("membership: read the hive: %w", err)
	}

	type row struct {
		ID          uuid.UUID
		Name        *string
		Address     string
		Fingerprint string
		RootKey     string
		SSHHostKeys datatypes.JSONType[[]string]
	}
	var rows []row
	// LEFT JOIN, so a hypervisor whose agent has never reported still appears in
	// the composition. Dropping it would tell every other member that the node is
	// desired-absent, and a member that is desired-absent gets EVICTED — turning
	// "this host's agent is not running yet" into "remove it from the cluster".
	if err := tx.WithContext(ctx).Raw(`
		SELECT h.id, h.name, h.address,
		       -- The agent's live report wins over what onboarding captured: host
		       -- keys survive a rebuild only by accident, and a stale pin fails a
		       -- join with a message that names nothing. The stored column stays
		       -- as the fallback for a host whose agent has not reported yet.
		       COALESCE(b.observed_host_keys, h.ssh_host_keys) AS ssh_host_keys,
		       COALESCE(b.observed_fingerprint, '') AS fingerprint,
		       COALESCE(b.observed_root_key, '')    AS root_key
		  FROM hypervisors h
		  LEFT JOIN bees b ON b.id = h.bee_id
		 WHERE h.hive_id = ?`, hiveID).Scan(&rows).Error; err != nil {
		return "", nil, fmt.Errorf("membership: read the hive's hypervisors: %w", err)
	}

	members := make([]tenant.MembershipMember, 0, len(rows))
	for _, r := range rows {
		name := ""
		if r.Name != nil {
			name = *r.Name
		}
		members = append(members, tenant.MembershipMember{
			Node:         name,
			HypervisorID: r.ID.String(),
			Address:      r.Address,
			Fingerprint:  r.Fingerprint,
			HostKeys:     r.SSHHostKeys.Data(),
			RootKey:      r.RootKey,
		})
	}
	return hive.Name, members, nil
}

// PublishHiveMembership renders one hive's composition and commits it as the
// next revision if it has changed.
//
// IDEMPOTENT, and that is what makes the whole loop safe to call from anywhere.
// Every write that changes membership intent should call this, and one that
// forgets is repaired by the next caller rather than leaving the fleet pinned to
// a stale composition forever. Publishing what is already published costs one
// locked read and no revision.
//
// The revision is read INSIDE the transaction with a row lock, for the same
// reason the firewall's is: two concurrent arrivals would otherwise derive the
// same next revision, and two different compositions would then claim to be the
// same number — which every agent uses as its whole basis for deciding whether
// it has already converged.
func PublishHiveMembership(ctx context.Context, db *gorm.DB, hiveID uuid.UUID) (MembershipResult, error) {
	var out MembershipResult

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var hive tenant.Hive
		if err := tx.Raw(`SELECT * FROM hives WHERE id = ? FOR UPDATE`, hiveID).
			Scan(&hive).Error; err != nil {
			return fmt.Errorf("membership: lock the hive: %w", err)
		}
		if hive.ID == uuid.Nil {
			return gorm.ErrRecordNotFound
		}

		// Resolved inside the transaction. Rendering outside it would let a
		// hypervisor's assignment change between the render and the write, and
		// the stored digest would then describe a composition no set of rows ever
		// produced — with every agent comparing against it.
		name, members, err := ResolveMembership(ctx, tx, hiveID)
		if err != nil {
			return err
		}
		doc, digest := tenant.RenderMembership(name, members)
		if digest == "" {
			return errors.New("membership: the composition could not be rendered")
		}

		if hive.MembershipRevision > 0 && hive.MembershipDigest == digest {
			out = MembershipResult{Revision: hive.MembershipRevision, Digest: digest, Members: len(members)}
			return ErrMembershipUnchanged
		}

		next := hive.MembershipRevision + 1

		// The BODY first, then the pointer at it — one transaction, the same
		// discipline the firewall publish states: a pointer naming a snapshot that
		// does not exist is every agent in the hive asking for a composition
		// nobody can serve.
		if err := tx.Create(&tenant.MembershipSnapshot{
			HiveID:   hiveID,
			Revision: next,
			Content:  datatypes.NewJSONType(doc),
			Digest:   digest,
		}).Error; err != nil {
			return fmt.Errorf("membership: write the composition snapshot: %w", err)
		}

		if err := tx.Model(&tenant.Hive{}).Where("id = ?", hiveID).
			Updates(map[string]any{
				"membership_revision":     next,
				"membership_digest":       digest,
				"membership_published_at": gorm.Expr("now()"),
			}).Error; err != nil {
			return fmt.Errorf("membership: move the hive's pointer: %w", err)
		}
		out = MembershipResult{Revision: next, Digest: digest, Members: len(members)}
		return nil
	})
	if errors.Is(err, ErrMembershipUnchanged) {
		return out, err
	}
	return out, err
}
