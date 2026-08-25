package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
)

type loginFakeUserRepo struct {
	repo.UserRepo
	byEmail map[string]*control.User
	lookups int
}

func (f *loginFakeUserRepo) GetByEmail(_ context.Context, email string) (*control.User, error) {
	f.lookups++
	u, ok := f.byEmail[email]
	if !ok {
		return nil, nil
	}
	return u, nil
}

type loginFailureRow struct {
	count       int
	windowStart time.Time
	lastFailed  time.Time
	lockedUntil *time.Time
}

type loginFakeFailureRepo struct {
	rows      map[string]*loginFailureRow
	lockedErr error
}

func newLoginFakeFailureRepo() *loginFakeFailureRepo {
	return &loginFakeFailureRepo{rows: map[string]*loginFailureRow{}}
}

func (f *loginFakeFailureRepo) key(identifier, kind string) string { return kind + "|" + identifier }

func (f *loginFakeFailureRepo) IsLocked(_ context.Context, identifier, kind string, now time.Time) (bool, error) {
	if f.lockedErr != nil {
		return false, f.lockedErr
	}
	row, ok := f.rows[f.key(identifier, kind)]
	if !ok {
		return false, nil
	}
	return row.lockedUntil != nil && row.lockedUntil.After(now), nil
}

func (f *loginFakeFailureRepo) RecordFailure(
	_ context.Context, identifier, kind string, threshold int, lockFor time.Duration, now time.Time,
) (int, bool, error) {
	k := f.key(identifier, kind)
	row, ok := f.rows[k]
	if !ok {
		row = &loginFailureRow{windowStart: now}
		f.rows[k] = row
	}
	if now.Sub(row.windowStart) > failureWindowForTest && (row.lockedUntil == nil || !row.lockedUntil.After(now)) {
		row.count = 0
		row.windowStart = now
		row.lockedUntil = nil
	}
	row.count++
	row.lastFailed = now
	locked := false
	if threshold > 0 && row.count >= threshold {
		until := now.Add(lockFor)
		row.lockedUntil = &until
		locked = true
	}
	return row.count, locked, nil
}

func (f *loginFakeFailureRepo) Clear(_ context.Context, identifier, kind string) error {
	delete(f.rows, f.key(identifier, kind))
	return nil
}

func (f *loginFakeFailureRepo) DeleteStale(_ context.Context, before time.Time) (int64, error) {
	var n int64
	for k, row := range f.rows {
		if row.lastFailed.Before(before) && (row.lockedUntil == nil || row.lockedUntil.Before(before)) {
			delete(f.rows, k)
			n++
		}
	}
	return n, nil
}

const failureWindowForTest = 15 * time.Minute

func newLoginTestService(users repo.UserRepo, failures repo.AuthFailureRepo) *LoginService {
	return NewLoginService(users, nil, nil, nil, nil, nil, nil, failures)
}

func loginAttempt(t *testing.T, svc *LoginService, email, ip string) error {
	t.Helper()
	_, err := svc.Login(context.Background(), LoginInput{
		Email:     email,
		Password:  "wrong-password",
		IPAddress: ip,
	})
	return err
}

func TestLoginLockoutIsScopedToTheSourceAddress(t *testing.T) {
	users := &loginFakeUserRepo{byEmail: map[string]*control.User{}}
	failures := newLoginFakeFailureRepo()
	svc := newLoginTestService(users, failures)

	const victim = "victim@example.test"
	const attackerIP = "203.0.113.7"

	for i := 0; i < LoginMaxFailures; i++ {
		if err := loginAttempt(t, svc, victim, attackerIP); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: got %v, want ErrInvalidCredentials", i+1, err)
		}
	}

	if err := loginAttempt(t, svc, victim, attackerIP); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("attacker's own source not locked after %d failures: %v", LoginMaxFailures, err)
	}

	if err := loginAttempt(t, svc, victim, "198.51.100.42"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("victim locked out from an unrelated address: %v", err)
	}
}

func TestPerEmailCounterAlertsButNeverLocks(t *testing.T) {
	users := &loginFakeUserRepo{byEmail: map[string]*control.User{}}
	failures := newLoginFakeFailureRepo()
	svc := newLoginTestService(users, failures)

	const victim = "victim@example.test"
	for i := 0; i < LoginEmailAlertThreshold+5; i++ {

		ip := fmt.Sprintf("203.0.113.%d", i%200)
		if i >= 200 {
			ip = fmt.Sprintf("198.51.100.%d", i%200)
		}
		if err := loginAttempt(t, svc, victim, ip); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d from %s: got %v", i+1, ip, err)
		}
	}

	row := failures.rows[failures.key(victim, loginFailureKindEmail)]
	if row == nil {
		t.Fatal("per-email counter was never written; the alert signal does not exist")
	}
	if row.count < LoginEmailAlertThreshold {
		t.Fatalf("per-email count %d, want at least %d", row.count, LoginEmailAlertThreshold)
	}
	if row.lockedUntil != nil {
		t.Fatal("the per-email counter locked; a lock keyed on the email alone is a remote denial of service")
	}

	if err := loginAttempt(t, svc, victim, "192.0.2.99"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("a fresh source was blocked by the per-email counter: %v", err)
	}
}

func TestLoginDeniesWhenTheLockoutStoreIsUnreadable(t *testing.T) {
	users := &loginFakeUserRepo{byEmail: map[string]*control.User{}}
	failures := newLoginFakeFailureRepo()
	failures.lockedErr = errors.New("connection reset by peer")
	svc := newLoginTestService(users, failures)

	if err := loginAttempt(t, svc, "victim@example.test", "203.0.113.7"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("got %v, want ErrAccountLocked when the lockout store errors", err)
	}
	if users.lookups != 0 {
		t.Fatal("the password path ran anyway: the lockout check fails open")
	}
}

func TestLoginWithNoClientIPDoesNotLockTheAccountItself(t *testing.T) {
	users := &loginFakeUserRepo{byEmail: map[string]*control.User{}}
	failures := newLoginFakeFailureRepo()
	svc := newLoginTestService(users, failures)

	const victim = "victim@example.test"
	for i := 0; i < LoginMaxFailures; i++ {
		if err := loginAttempt(t, svc, victim, ""); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}
	if _, ok := failures.rows[failures.key(victim, loginFailureKindScoped)]; ok {
		t.Fatal("the blocking counter is keyed on the bare email address")
	}

	if err := loginAttempt(t, svc, victim, "203.0.113.7"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("a named source inherited the unnamed bucket's lock: %v", err)
	}
}

func TestNewLoginServiceRefusesToStartWithoutAFailureRepo(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewLoginService accepted a nil AuthFailureRepo, silently disabling the login lockout")
		}
	}()
	_ = newLoginTestService(&loginFakeUserRepo{}, nil)
}

func TestLoginScopeKeyIsUnambiguous(t *testing.T) {
	if loginScopeKey("a@example.test", "203.0.113.7") == loginScopeKey("a@example.test", "203.0.113.8") {
		t.Fatal("two source addresses share a lockout bucket")
	}
	if loginScopeKey("a@example.test", "203.0.113.7") == loginScopeKey("b@example.test", "203.0.113.7") {
		t.Fatal("two accounts share a lockout bucket")
	}

	if loginScopeKey("ab", "c") == loginScopeKey("a", "bc") {
		t.Fatal("the scope key is ambiguous across the email/address boundary")
	}
	if got := len(loginScopeKey("a@example.test", "203.0.113.7")); got != 64 {
		t.Fatalf("scope key is %d chars; auth_failures.identifier is varchar(320)", got)
	}
}

func TestSuccessfulLoginClearsBothCounters(t *testing.T) {
	user := &control.User{ID: uuid.New(), Email: "user@example.test", IsActive: true}
	if err := user.SetPassword("correct-horse-battery-staple"); err != nil {
		t.Skipf("control.User has no SetPassword helper in this tree: %v", err)
	}

	users := &loginFakeUserRepo{byEmail: map[string]*control.User{user.Email: user}}
	failures := newLoginFakeFailureRepo()
	svc := newLoginTestService(users, failures)

	if err := loginAttempt(t, svc, user.Email, "203.0.113.7"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("seed failure: %v", err)
	}
	if len(failures.rows) != 2 {
		t.Fatalf("expected a scoped and a per-email counter, got %d rows", len(failures.rows))
	}

	func() {
		defer func() { _ = recover() }()
		_, _ = svc.Login(context.Background(), LoginInput{
			Email:     user.Email,
			Password:  "correct-horse-battery-staple",
			IPAddress: "203.0.113.7",
		})
	}()

	if len(failures.rows) != 0 {
		t.Fatalf("a correct password left %d failure counters behind", len(failures.rows))
	}
}
