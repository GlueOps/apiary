package apimiddleware

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/service"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// accessTestSchema is private to this test binary invocation. Opening the DSN
// unscoped put these tables in public, which against a developer's own
// DATABASE_URL means the suite migrates the real control database.
var accessTestSchema = testsupport.SchemaName("apiary_apimiddleware_access_test")

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, accessTestSchema, &control.User{}, &control.TokenSession{})
}

func seedRevokedSession(t *testing.T, db *gorm.DB, revoked bool) (uuid.UUID, string) {
	t.Helper()

	user := control.User{
		Email:     "revocation-" + uuid.NewString() + "@example.test",
		Password:  "x",
		FirstName: "Test",
		LastName:  "User",
		Role:      control.RolePlatformCustomer,
		IsActive:  true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&control.User{}, "id = ?", user.ID) })

	now := time.Now().UTC()
	session := control.TokenSession{
		SubjectID:        user.ID,
		SubjectKind:      "user",
		OrganizationID:   uuid.New(),
		FamilyID:         uuid.New(),
		AccessTokenJTI:   uuid.NewString(),
		RefreshTokenJTI:  uuid.NewString(),
		AccessTokenHash:  "h",
		RefreshTokenHash: "h",
		AccessExpiresAt:  now.Add(15 * time.Minute),
		RefreshExpiresAt: now.Add(168 * time.Hour),
	}
	if revoked {
		revokedAt := now.Add(-time.Minute)
		session.RevokedAt = &revokedAt
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&control.TokenSession{}, "id = ?", session.ID) })

	return user.ID, session.AccessTokenJTI
}

func claimsFor(userID uuid.UUID, jti string) *service.AuthClaims {
	now := time.Now().UTC()
	return &service.AuthClaims{
		OrgID: uuid.NewString(),
		Type:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	}
}

func depsFor(db *gorm.DB) *app.Deps {
	return &app.Deps{Config: &config.Config{}, ControlDB: db}
}

func TestCheckSessionLiveness_RejectsRevokedSession(t *testing.T) {
	db := testDB(t)
	userID, jti := seedRevokedSession(t, db, true)

	err := checkSessionLiveness(context.Background(), depsFor(db), claimsFor(userID, jti), userID)

	if !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("a revoked session must be denied: got err=%v, want ErrAccessDenied", err)
	}
}

func TestCheckSessionLiveness_AllowsLiveSession(t *testing.T) {
	db := testDB(t)
	userID, jti := seedRevokedSession(t, db, false)

	if err := checkSessionLiveness(context.Background(), depsFor(db), claimsFor(userID, jti), userID); err != nil {
		t.Fatalf("a live session must be allowed: got err=%v, want nil", err)
	}
}

func TestCheckSessionLiveness_RejectsUnknownJTI(t *testing.T) {
	db := testDB(t)
	userID, _ := seedRevokedSession(t, db, false)

	err := checkSessionLiveness(context.Background(), depsFor(db), claimsFor(userID, uuid.NewString()), userID)

	if !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("a JTI with no session row must be denied: got err=%v, want ErrAccessDenied", err)
	}
}

func TestTokenSessionAccessJTIColumnName(t *testing.T) {
	parsed, err := schema.Parse(&control.TokenSession{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse TokenSession schema: %v", err)
	}

	if _, ok := parsed.FieldsByDBName[accessJTIColumn]; !ok {
		have := make([]string, 0, len(parsed.FieldsByDBName))
		for name := range parsed.FieldsByDBName {
			have = append(have, name)
		}
		sort.Strings(have)
		t.Fatalf("checkSessionLiveness queries column %q, which TokenSession does not have; columns are %v",
			accessJTIColumn, have)
	}
}
