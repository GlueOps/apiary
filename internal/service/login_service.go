package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user inactive")
	ErrNoOrganization     = errors.New("user has no organization membership")
)

var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("apiary-dummy-password"), bcrypt.DefaultCost)

type LoginInput struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

type LoginResult struct {
	User           *control.User
	OrganizationID string
	Organization   string
	Tokens         *IssuedTokens
	MFARequired    bool
	MFAToken       string
}

type LoginService struct {
	users       repo.UserRepo
	memberships repo.MembershipRepo
	totps       repo.UserTOTPRepo
	tokens      *TokenService
	sessions    *TokenSessionService
	policies    *PolicyService
	challenges  repo.AuthChallengeRepo
	failures    repo.AuthFailureRepo
}

const (
	LoginMaxFailures = 10
	LoginLockoutFor  = 15 * time.Minute

	LoginEmailAlertThreshold = 50
)

const (
	loginFailureKindScoped = "login_ip"

	loginFailureKindEmail = "login_email"
)

var ErrAccountLocked = errors.New("account temporarily locked")

const MFAPendingMaxAttempts = 5

func loginScopeKey(email, ip string) string {
	sum := sha256.Sum256([]byte(email + "\x00" + ip))
	return hex.EncodeToString(sum[:])
}

func NewLoginService(users repo.UserRepo, memberships repo.MembershipRepo, totps repo.UserTOTPRepo, tokens *TokenService, sessions *TokenSessionService, policies *PolicyService, challenges repo.AuthChallengeRepo, failures repo.AuthFailureRepo) *LoginService {

	if failures == nil {
		panic("NewLoginService: auth failure repo is required — without it there is no login lockout")
	}
	return &LoginService{
		users:       users,
		memberships: memberships,
		totps:       totps,
		tokens:      tokens,
		sessions:    sessions,
		policies:    policies,
		challenges:  challenges,
		failures:    failures,
	}
}

func (s *LoginService) Login(ctx context.Context, in LoginInput) (*LoginResult, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" || in.Password == "" {
		return nil, ErrInvalidCredentials
	}

	ip := strings.TrimSpace(in.IPAddress)
	if ip == "" {

		slog.ErrorContext(ctx, "login: client IP unavailable; failures share one lockout bucket")
		ip = "unknown"
	}

	now0 := time.Now().UTC()
	scope := loginScopeKey(email, ip)

	locked, ferr := s.failures.IsLocked(ctx, scope, loginFailureKindScoped, now0)
	if ferr != nil {

		slog.ErrorContext(ctx, "login lockout check failed", "error", ferr)
		return nil, ErrAccountLocked
	}
	if locked {

		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(in.Password))
		return nil, ErrAccountLocked
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {

		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(in.Password))
		s.recordLoginFailure(ctx, email, scope, now0)
		return nil, ErrInvalidCredentials
	}
	if !user.CheckPassword(in.Password) {
		s.recordLoginFailure(ctx, email, scope, now0)
		return nil, ErrInvalidCredentials
	}

	if cerr := s.failures.Clear(ctx, scope, loginFailureKindScoped); cerr != nil {
		slog.ErrorContext(ctx, "failed to clear scoped login failures", "error", cerr)
	}
	if cerr := s.failures.Clear(ctx, email, loginFailureKindEmail); cerr != nil {
		slog.ErrorContext(ctx, "failed to clear per-email login failures", "error", cerr)
	}
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// The confirmation gate, and it sits HERE deliberately: after the password
	// has been checked and after the failure counters have been cleared.
	//
	// After the password check, because ErrEmailNotVerified is a distinguishable
	// answer. Placed before it, this endpoint would tell an anonymous caller
	// which addresses are registered but unconfirmed. Placed after, the only
	// people who can observe it are those who already supplied the correct
	// password for the account — they are entitled to know why they are being
	// turned away, and telling them is what lets the UI offer a resend rather
	// than "wrong password".
	//
	// Before the session is issued, because a token_session is precisely what
	// this gate exists to withhold: sessions on distinct days are what the
	// proving sweep counts, so an unconfirmed account earning them would make
	// the admission ceiling meaningless.
	//
	// Social sign-in does not pass through here. That path stamps
	// EmailVerifiedAt from the provider's own email_verified claim and refuses
	// an unverified one outright, so a social user arrives already confirmed.
	if !user.EmailVerified() {
		return nil, ErrEmailNotVerified
	}

	memberships, err := s.memberships.ListByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return nil, ErrNoOrganization
	}

	selected := memberships[0]

	now := time.Now().UTC()
	if err := s.users.UpdateLastLogin(ctx, user.ID, now); err != nil {
		return nil, err
	}

	orgIDStr := selected.OrganizationID.String()

	mfaRequired := user.MFAEnabled
	if s.policies != nil {
		orgUUID, _ := uuid.Parse(orgIDStr)

		ep, perr := s.policies.EffectivePolicy(ctx, orgUUID, control.PolicyKindRequireMFA)
		if perr != nil {
			return nil, perr
		}
		if ep != nil && ep.Enabled {
			mfaRequired = true
		}
	}

	if mfaRequired {
		totp, err := s.totps.GetByUserID(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		if totp != nil && totp.Verified {
			mfaToken, jti, err := s.tokens.IssueMFAPendingWithJTI(user.ID.String(), orgIDStr, now)
			if err != nil {
				return nil, err
			}

			if s.challenges != nil {
				if err := s.challenges.Create(ctx, &control.AuthChallenge{
					JTI:         jti,
					UserID:      user.ID,
					Kind:        "mfa_pending",
					MaxAttempts: MFAPendingMaxAttempts,
					ExpiresAt:   now.Add(5 * time.Minute),
				}); err != nil {
					return nil, err
				}
			}
			return &LoginResult{
				User:           user,
				OrganizationID: orgIDStr,
				Organization:   selected.Organization.Name,
				MFARequired:    true,
				MFAToken:       mfaToken,
			}, nil
		}
	}

	effectiveMFAEnabled := mfaRequired

	issued, err := s.sessions.IssueTracked(ctx, IssueTrackedInput{
		SubjectID:   user.ID.String(),
		SubjectKind: "user",
		OrgID:       orgIDStr,
		Opts: IssueOpts{
			AMR:        []string{"pwd"},
			MFA:        false,
			MFAEnabled: effectiveMFAEnabled,
		},
		IPAddress: in.IPAddress,
		UserAgent: in.UserAgent,
		EventType: "login",
	})
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:           user,
		OrganizationID: orgIDStr,
		Organization:   selected.Organization.Name,
		Tokens:         issued,
	}, nil
}

func (s *LoginService) recordLoginFailure(ctx context.Context, email, scope string, now time.Time) {

	if _, _, err := s.failures.RecordFailure(
		ctx, scope, loginFailureKindScoped, LoginMaxFailures, LoginLockoutFor, now,
	); err != nil {
		slog.ErrorContext(ctx, "failed to record scoped login failure", "error", err)
	}

	count, _, err := s.failures.RecordFailure(ctx, email, loginFailureKindEmail, 0, 0, now)
	if err != nil {
		slog.ErrorContext(ctx, "failed to record per-email login failure", "error", err)
		return
	}
	if count >= LoginEmailAlertThreshold {

		slog.WarnContext(ctx, "login failure burst against a single account",
			"email", email, "failures", count)
	}
}
