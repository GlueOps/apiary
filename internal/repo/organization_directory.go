package repo

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationFilter narrows the directory. The zero filter is "every live
// organization, newest first".
type OrganizationFilter struct {
	// Query matches the name or the domain, case-insensitively.
	Query string

	// IncludeDeleted brings back soft-deleted organizations.
	//
	// Off by default, and that default is the useful one: a deleted
	// organization is not a thing to target with a feature flag, invite somebody
	// to, or count. It stays reachable behind an explicit flag because soft
	// deletion exists precisely so a mistaken one can be seen and undone.
	IncludeDeleted bool

	Limit  int
	Offset int
}

const (
	organizationsDefaultLimit = 50
	organizationsMaxLimit     = 200
)

// OrganizationSummary is what a platform operator may see about a tenant.
//
// AN EXPLICIT STRUCT, NEVER THE MODEL. control.Organization carries
// EncryptedTenantKey, the sealed ConnectionString and their IVs and tags — the
// only material that can decrypt that customer's database. Those fields are
// json:"-" today, and a struct that relies on a tag to stay secret is one
// refactor away from not being. The query below names its columns for the same
// reason: the secrets are never read out of the database at all, so no later
// change to a response shape can leak what was never loaded.
type OrganizationSummary struct {
	ID        uuid.UUID
	Name      string
	Domain    string
	Status    string
	CreatedAt time.Time
	DeletedAt *time.Time
}

// ListOrganizations returns the tenant directory.
//
// PLATFORM-ONLY BY CONSTRUCTION OF ITS CALLER, not by anything here. This
// function has no idea who is asking; it is mounted behind the admin group,
// which is where that decision belongs. It is worth saying out loud because a
// list of every tenant is exactly the thing a customer must never obtain — the
// names of the other organizations on the installation are not theirs to see.
func ListOrganizations(
	ctx context.Context, db *gorm.DB, f OrganizationFilter,
) ([]OrganizationSummary, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = organizationsDefaultLimit
	}
	if limit > organizationsMaxLimit {
		limit = organizationsMaxLimit
	}

	q := db.WithContext(ctx).
		Table("organizations").
		Select("id, name, coalesce(domain,'') AS domain, status, created_at, deleted_at").
		Order("name ASC").
		Limit(limit).Offset(f.Offset)

	if !f.IncludeDeleted {
		q = q.Where("deleted_at IS NULL")
	}
	if s := strings.TrimSpace(f.Query); s != "" {
		// ESCAPED before it reaches LIKE. Without this a search for "100%" is a
		// wildcard that matches every row, and an underscore quietly matches any
		// single character — which reads as a broken search rather than as the
		// pattern syntax leaking through.
		esc := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(s)
		pattern := "%" + esc + "%"
		q = q.Where("name ILIKE ? ESCAPE '\\' OR coalesce(domain,'') ILIKE ? ESCAPE '\\'",
			pattern, pattern)
	}

	var out []OrganizationSummary
	if err := q.Scan(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
