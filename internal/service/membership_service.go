package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// ErrMembershipNotInOrg is returned when a membership id names a row that
	// belongs to a different organization. It is reported as 404, identical to a
	// row that does not exist, so the endpoint cannot be used to discover which
	// membership ids are real.
	ErrMembershipNotInOrg = errors.New("membership: no such membership in this organization")

	// ErrLastOwner guards the one state an organization can never be in.
	ErrLastOwner = errors.New("membership: an organization must keep at least one owner")

	// ErrOwnerRoleRequiresOwner: admins manage members and admins. Granting or
	// revoking ownership is an owner's decision. Without this an admin could
	// promote themselves to owner, which makes the owner role decorative.
	ErrOwnerRoleRequiresOwner = errors.New("membership: only an owner may grant or revoke the owner role")
)

type MembershipService struct {
	// db is required by UpdateRole and Remove, which must read and write inside
	// one transaction. List does not need it.
	db          *gorm.DB
	memberships repo.MembershipRepo
	audit       repo.AuthAuditRepo
}

func NewMembershipService(db *gorm.DB, memberships repo.MembershipRepo, audit repo.AuthAuditRepo) *MembershipService {
	return &MembershipService{db: db, memberships: memberships, audit: audit}
}

type MemberView struct {
	MembershipID   uuid.UUID
	UserID         uuid.UUID
	Email          string
	FirstName      string
	LastName       string
	Role           control.MemberRole
	SupportEnabled bool
}

func (s *MembershipService) List(ctx context.Context, orgID uuid.UUID) ([]MemberView, error) {
	rows, err := s.memberships.ListByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]MemberView, 0, len(rows))
	for _, m := range rows {
		out = append(out, MemberView{
			MembershipID:   m.ID,
			UserID:         m.UserID,
			Email:          m.User.Email,
			FirstName:      m.User.FirstName,
			LastName:       m.User.LastName,
			Role:           m.Role,
			SupportEnabled: m.SupportEnabled,
		})
	}
	return out, nil
}

// lockOrgMemberships reads every membership row in an organization with
// SELECT ... FOR UPDATE, and runs fn against that locked set inside the same
// transaction.
//
// The lock covers the WHOLE SET, not just the row being changed, because the
// invariant is over the set: "this organization keeps at least one owner".
// Locking only the target row does not serialise two requests demoting two
// DIFFERENT owners — they touch different rows, take different locks, and
// proceed in parallel.
//
// That was a real defect, not a theoretical one. The previous version counted
// owners with an unlocked SELECT and then wrote outside any transaction, so two
// concurrent demotions in a two-owner organization each observed n=2, each
// passed the `n <= 1` guard, and the organization was left with ZERO owners —
// exactly the state ErrLastOwner exists to make unreachable, and unrecoverable
// without platform intervention because nobody left inside can promote one. A
// probe racing two demotions reproduced it in 7 of 12 rounds.
func (s *MembershipService) lockOrgMemberships(
	ctx context.Context,
	orgID uuid.UUID,
	fn func(tx *gorm.DB, locked []control.Membership) error,
) error {
	if s.db == nil {
		// A guard that cannot take its lock must refuse rather than fall back to
		// the unsafe path.
		return errors.New("membership: no database handle; refusing to mutate membership without a lock")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked []control.Membership
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("organization_id = ?", orgID).
			Order("id").
			Find(&locked).Error; err != nil {
			return err
		}
		return fn(tx, locked)
	})
}

// findLocked resolves a membership from the locked set.
//
// Resolving from the SET rather than re-reading is what makes the decision and
// the write agree: a re-read outside the lock could see a different row than the
// one whose absence or role the guards were computed against.
//
// A membership id belonging to another organization is not in the set, so it is
// reported as not-found — identical to one that does not exist, which is what
// keeps this from being a cross-tenant probe.
func findLocked(locked []control.Membership, id uuid.UUID) (*control.Membership, error) {
	for i := range locked {
		if locked[i].ID == id {
			return &locked[i], nil
		}
	}
	return nil, ErrMembershipNotInOrg
}

func countOwners(locked []control.Membership) int {
	n := 0
	for _, m := range locked {
		if m.Role == control.MemberRoleOwner {
			n++
		}
	}
	return n
}

type UpdateRoleInput struct {
	OrgID        uuid.UUID
	MembershipID uuid.UUID
	NewRole      control.MemberRole
	ActorID      uuid.UUID
	ActorRole    control.MemberRole
}

func (s *MembershipService) UpdateRole(ctx context.Context, in UpdateRoleInput) error {
	if in.NewRole != control.MemberRoleOwner &&
		in.NewRole != control.MemberRoleAdmin &&
		in.NewRole != control.MemberRoleMember {
		return errors.New("membership: unknown role")
	}

	// Escapes the closure because a nil return means both "changed it" and
	// "nothing to change", and only the first may be audited. An audit log that
	// records changes which did not happen is not evidence of anything.
	noop := false

	err := s.lockOrgMemberships(ctx, in.OrgID, func(tx *gorm.DB, locked []control.Membership) error {
		m, err := findLocked(locked, in.MembershipID)
		if err != nil {
			return err
		}
		if m.Role == in.NewRole {
			noop = true
			return nil
		}

		// Either direction across the owner line is an owner's call.
		if (in.NewRole == control.MemberRoleOwner || m.Role == control.MemberRoleOwner) &&
			in.ActorRole != control.MemberRoleOwner {
			return ErrOwnerRoleRequiresOwner
		}

		// Demoting the last owner would leave the organization with nobody able
		// to promote one back — including the person who did it. Counted from
		// the locked set, so a concurrent demotion cannot slip between the count
		// and the write.
		if m.Role == control.MemberRoleOwner && countOwners(locked) <= 1 {
			return ErrLastOwner
		}

		return tx.Model(&control.Membership{}).
			Where("id = ?", m.ID).
			Update("role", in.NewRole).Error
	})
	if err != nil {
		return err
	}
	if noop {
		return nil
	}
	s.writeAudit(ctx, in.ActorID, in.OrgID, "membership_role_changed")
	return nil
}

type RemoveMemberInput struct {
	OrgID        uuid.UUID
	MembershipID uuid.UUID
	ActorID      uuid.UUID
	ActorRole    control.MemberRole
}

func (s *MembershipService) Remove(ctx context.Context, in RemoveMemberInput) error {
	err := s.lockOrgMemberships(ctx, in.OrgID, func(tx *gorm.DB, locked []control.Membership) error {
		m, err := findLocked(locked, in.MembershipID)
		if err != nil {
			return err
		}

		if m.Role == control.MemberRoleOwner {
			if in.ActorRole != control.MemberRoleOwner {
				return ErrOwnerRoleRequiresOwner
			}
			if countOwners(locked) <= 1 {
				return ErrLastOwner
			}
		}

		return tx.Delete(&control.Membership{}, "id = ?", m.ID).Error
	})
	if err != nil {
		return err
	}
	s.writeAudit(ctx, in.ActorID, in.OrgID, "membership_removed")
	return nil
}

func (s *MembershipService) writeAudit(ctx context.Context, actorID, orgID uuid.UUID, eventType string) {
	if s.audit == nil {
		return
	}
	if err := s.audit.Create(ctx, &control.AuthAuditEvent{
		SubjectID:      actorID,
		SubjectKind:    "user",
		OrganizationID: orgID,
		EventType:      eventType,
		Metadata:       datatypes.NewJSONType(map[string]any{}),
	}); err != nil {
		slog.ErrorContext(ctx, "failed to write auth audit event", "event_type", eventType, "error", err)
	}
}
