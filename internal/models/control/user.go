package control

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserRole string

// PLATFORM roles, held on the user record. This is one of two independent
// ladders and they are not interchangeable:
//
//	platform (here)          who you are to the platform
//	tenant (MemberRole)      what you may do inside one organization
//
// Both used to contain a bare "admin", in code and in the database, which made
// the two indistinguishable at a glance and invited exactly the mistake of
// answering "may they drain this hypervisor" with the wrong ladder. The prefix
// is on the VALUE as well as the identifier so that a row, a log line and an API
// payload each say which ladder they mean.
const (
	// RolePlatformAdmin operates the platform. Everything, including policy that
	// binds every tenant.
	RolePlatformAdmin UserRole = "platform_admin"

	// RolePlatformSupport may impersonate a member who has consented, and that
	// is its whole standing capability — it reaches a tenant's resources AS that
	// member, never by holding a grant over them.
	RolePlatformSupport UserRole = "platform_support"

	// RolePlatformSales carries nothing today.
	RolePlatformSales UserRole = "platform_sales"

	// RolePlatformCustomer is everyone else: tenant-bound operations only,
	// authorized entirely through membership.
	RolePlatformCustomer UserRole = "platform_customer"
)

type User struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email      string    `gorm:"uniqueIndex;not null;type:varchar(255)" json:"email"`
	Password   string    `gorm:"not null;type:varchar(255)" json:"-"`
	FirstName  string    `gorm:"not null;type:varchar(100)" json:"first_name"`
	LastName   string    `gorm:"not null;type:varchar(100)" json:"last_name"`
	Role       UserRole  `gorm:"not null;default:'platform_customer';type:varchar(20)" json:"role"`
	MFAEnabled bool      `gorm:"default:false" json:"mfa_enabled"`
	// EmailVerifiedAt records when this user proved control of Email. It is a
	// timestamp rather than a bool to match ProvenAt and DomainVerifiedAt, and
	// because "when" is what an audit of a disputed registration needs.
	//
	// It lives on User, not on Account: Account is the social-identity table and
	// a password registration has no row there at all, so Account.EmailVerified
	// can only ever speak for an address an IdP asserted.
	EmailVerifiedAt *time.Time `gorm:"type:timestamptz;column:email_verified_at" json:"email_verified_at,omitempty"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`

	// DeletedAt soft-deletes the account.
	//
	// The row is kept because other records point at it and must stay readable:
	// audit events name the actor, fleet tasks name who created them, and
	// memberships record who invited whom. Removing the row would either break
	// those references or cascade them away — and an audit trail that disappears
	// when the account does is an audit trail that can be erased by the person
	// being audited.
	//
	// Distinct from IsActive, which is a suspension: reversible, and the account
	// still exists. This is the account being gone.
	DeletedAt       *time.Time `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
	TokensRevokedAt *time.Time `gorm:"type:timestamptz" json:"-"`
	LastLoginAt     *time.Time `gorm:"autoUpdateTime;column:last_login_at;not null;default:now()" json:"last_login_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at,omitempty" gorm:"type:timestamptz;column:created_at;not null;default:now()"`
	UpdatedAt       time.Time  `json:"updated_at,omitempty" gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()"`
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

func (u *User) SetPassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) == nil
}

// EmailVerified reports whether this user has proven control of their address.
//
// Callers ask through this method rather than comparing the pointer themselves,
// so that the meaning of "verified" has one definition. A nil check written out
// at each call site is how one path ends up treating a legacy NULL as verified.
func (u *User) EmailVerified() bool {
	return u != nil && u.EmailVerifiedAt != nil
}
