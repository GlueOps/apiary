package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var (
	ErrUnsupportedProvider = errors.New("unsupported provider")
	ErrProviderNotConfig   = errors.New("provider not configured")
	ErrMissingCode         = errors.New("missing code")
	ErrEmailRequired       = errors.New("verified email required")
)

type oauthProvider struct {
	Name     string
	Issuer   string
	Scopes   []string
	ClientID string
	Secret   string
}

func providerConfig(cfg *config.Config, name string) (oauthProvider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "google":
		if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
			return oauthProvider{}, ErrProviderNotConfig
		}
		return oauthProvider{
			Name:     "google",
			Issuer:   "https://accounts.google.com",
			Scopes:   []string{oidc.ScopeOpenID, "email", "profile"},
			ClientID: cfg.GoogleClientID,
			Secret:   cfg.GoogleClientSecret,
		}, nil

	case "keycloak":
		if cfg.KeycloakClientID == "" || cfg.KeycloakClientSecret == "" || cfg.KeycloakIssuerURL == "" {
			return oauthProvider{}, ErrProviderNotConfig
		}
		return oauthProvider{
			Name: "keycloak",
			// Trailing slash trimmed: go-oidc requires the issuer it discovers
			// to match this string byte for byte, and Keycloak reports its own
			// issuer without one, so ".../realms/apiary/" fails verification
			// with a confusing issuer-mismatch error rather than a 404.
			Issuer:   strings.TrimRight(strings.TrimSpace(cfg.KeycloakIssuerURL), "/"),
			Scopes:   []string{oidc.ScopeOpenID, "email", "profile"},
			ClientID: cfg.KeycloakClientID,
			Secret:   cfg.KeycloakClientSecret,
		}, nil

	case "github":
		if cfg.GithubClientID == "" || cfg.GithubClientSecret == "" {
			return oauthProvider{}, ErrProviderNotConfig
		}
		return oauthProvider{
			Name:     "github",
			Issuer:   "github",
			Scopes:   []string{"read:user", "user:email"},
			ClientID: cfg.GithubClientID,
			Secret:   cfg.GithubClientSecret,
		}, nil
	}

	return oauthProvider{}, ErrUnsupportedProvider
}

type SocialAuthInput struct {
	Provider string
	Code     string
	IP       string
	UA       string
}

type SocialAuthResult struct {
	User           *control.User
	OrganizationID string
	Organization   string
	Tokens         *IssuedTokens
	MFARequired    bool
	MFAToken       string

	Mode   string
	Origin string
}

type SocialAuthService struct {
	cfg         *config.Config
	socialRepo  repo.SocialAuthRepo
	accessRules repo.AuthAccessRuleRepo
	memberships repo.MembershipRepo
	totps       repo.UserTOTPRepo
	tokens      *TokenService
	sessions    *TokenSessionService
	httpClient  *http.Client
	bootstrap   *OrgBootstrapService
	policies    *PolicyService
	states      repo.OAuthStateRepo
}

func NewSocialAuthService(
	cfg *config.Config,
	socialRepo repo.SocialAuthRepo,
	accessRules repo.AuthAccessRuleRepo,
	memberships repo.MembershipRepo,
	totps repo.UserTOTPRepo,
	tokens *TokenService,
	bootstrap *OrgBootstrapService,
	sessions *TokenSessionService,
	policies *PolicyService,
	states repo.OAuthStateRepo,
) *SocialAuthService {
	return &SocialAuthService{
		cfg:         cfg,
		socialRepo:  socialRepo,
		accessRules: accessRules,
		memberships: memberships,
		totps:       totps,
		bootstrap:   bootstrap,
		tokens:      tokens,
		sessions:    sessions,
		policies:    policies,
		states:      states,
		httpClient:  http.DefaultClient,
	}
}

func effectiveMFARequirement(
	ctx context.Context,
	policies *PolicyService,
	user *control.User,
	orgID uuid.UUID,
) (bool, error) {
	if user != nil && user.MFAEnabled {
		return true, nil
	}
	if policies == nil {
		return false, nil
	}

	ep, err := policies.EffectivePolicy(ctx, orgID, control.PolicyKindRequireMFA)
	if err != nil {
		return false, err
	}
	return ep != nil && ep.Enabled, nil
}

func (s *SocialAuthService) StartURL(ctx context.Context, providerName, mode, origin string) (authURL, cookieValue string, err error) {
	p, err := providerConfig(s.cfg, providerName)
	if err != nil {
		return "", "", err
	}

	redirect := strings.TrimRight(s.cfg.OAuthRedirectBase, "/") + "/api/v1/auth/" + p.Name + "/callback"

	state, err := randomToken(32)
	if err != nil {
		return "", "", err
	}
	cookieValue, err = randomToken(32)
	if err != nil {
		return "", "", err
	}

	if err := s.states.Create(ctx, &control.OAuthState{
		StateHash:  hashToken(state),
		Provider:   p.Name,
		Mode:       strings.ToLower(strings.TrimSpace(mode)),
		Origin:     strings.TrimSpace(origin),
		CookieHash: hashToken(cookieValue),
		ExpiresAt:  time.Now().UTC().Add(oauthStateTTL),
	}); err != nil {
		return "", "", err
	}

	if p.Issuer == "github" {
		o := &oauth2.Config{
			ClientID:     p.ClientID,
			ClientSecret: p.Secret,
			RedirectURL:  redirect,
			Scopes:       p.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://github.com/login/oauth/authorize",
				TokenURL: "https://github.com/login/oauth/access_token",
			},
		}
		return o.AuthCodeURL(state, oauth2.AccessTypeOffline), cookieValue, nil
	}

	prov, err := oidc.NewProvider(ctx, p.Issuer)
	if err != nil {
		return "", "", err
	}

	o := &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.Secret,
		RedirectURL:  redirect,
		Endpoint:     prov.Endpoint(),
		Scopes:       p.Scopes,
	}
	return o.AuthCodeURL(state, oauth2.AccessTypeOffline), cookieValue, nil
}

func (s *SocialAuthService) Callback(ctx context.Context, providerName, code, state, stateCookie, ip, ua string) (*SocialAuthResult, error) {
	if strings.TrimSpace(code) == "" {
		return nil, ErrMissingCode
	}

	p, err := providerConfig(s.cfg, providerName)
	if err != nil {
		return nil, err
	}

	stateRec, err := s.consumeState(ctx, p.Name, state, stateCookie)
	if err != nil {
		return nil, err
	}

	redirect := strings.TrimRight(s.cfg.OAuthRedirectBase, "/") + "/api/v1/auth/" + p.Name + "/callback"

	identity, err := s.exchangeIdentity(ctx, p, redirect, code)
	if err != nil {
		return nil, err
	}

	if identity.Email == "" || !identity.EmailVerified {
		return nil, ErrEmailRequired
	}

	// The ban list only. Org creation on this path is gated later, inside the
	// bootstrap transaction, against purpose=create_org — so a user whose domain
	// may not mint new tenants can still sign in to one they were added to.
	if err := s.accessRules.CheckAuthentication(ctx, identity.Email); err != nil {
		return nil, err
	}

	user, err := s.socialRepo.UpsertFromIdentity(ctx, repo.UpsertSocialIdentityInput{
		Provider:      identity.Provider,
		Subject:       identity.Subject,
		Email:         identity.Email,
		EmailVerified: identity.EmailVerified,
		DisplayName:   identity.DisplayName,
		AvatarURL:     identity.AvatarURL,
		Profile:       identity.Profile,

		LinkByEmail: s.cfg.AuthSocialLinkByEmail,
	})
	if err != nil {
		return nil, err
	}

	if err := s.socialRepo.EnsureAutoMembership(ctx, user.ID, identity.Email); err != nil {
		return nil, err
	}

	memberships, err := s.memberships.ListByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		m, err := s.bootstrap.EnsureUserHasWorkspace(ctx, user)
		if err != nil {
			return nil, err
		}
		memberships = []control.Membership{*m}
	}

	selected := memberships[0]
	now := time.Now().UTC()
	orgIDStr := selected.OrganizationID.String()

	mfaRequired, err := effectiveMFARequirement(ctx, s.policies, user, selected.OrganizationID)
	if err != nil {
		return nil, err
	}

	if mfaRequired {
		totp, err := s.totps.GetByUserID(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		if totp != nil && totp.Verified {
			mfaToken, err := s.tokens.IssueMFAPending(user.ID.String(), orgIDStr, now)
			if err != nil {
				return nil, err
			}
			return &SocialAuthResult{
				User:           user,
				OrganizationID: orgIDStr,
				Organization:   selected.Organization.Name,
				MFARequired:    true,
				MFAToken:       mfaToken,
				Mode:           stateRec.Mode,
				Origin:         stateRec.Origin,
			}, nil
		}
	}

	issued, err := s.sessions.IssueTracked(ctx, IssueTrackedInput{
		SubjectID:   user.ID.String(),
		SubjectKind: "user",
		OrgID:       orgIDStr,
		Opts: IssueOpts{
			AMR: []string{"social"},
			MFA: false,

			MFAEnabled: mfaRequired,
		},
		IPAddress: ip,
		UserAgent: ua,
		EventType: "login",
	})
	if err != nil {
		return nil, err
	}

	return &SocialAuthResult{
		User:           user,
		OrganizationID: orgIDStr,
		Organization:   selected.Organization.Name,
		Tokens:         issued,
		Mode:           stateRec.Mode,
		Origin:         stateRec.Origin,
	}, nil
}

type socialIdentity struct {
	Provider      string
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
	AvatarURL     string
	Profile       map[string]any
}

func (s *SocialAuthService) exchangeIdentity(ctx context.Context, p oauthProvider, redirect, code string) (*socialIdentity, error) {
	if p.Issuer == "github" {
		return s.exchangeGithub(ctx, p, redirect, code)
	}
	return s.exchangeOIDC(ctx, p, redirect, code)
}

func (s *SocialAuthService) exchangeGithub(ctx context.Context, p oauthProvider, redirect, code string) (*socialIdentity, error) {
	o := &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.Secret,
		RedirectURL:  redirect,
		Scopes:       p.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}

	tok, err := o.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(tok))

	var ghUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := getJSON(client, "https://api.github.com/user", &ghUser); err != nil {
		return nil, err
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := getJSON(client, "https://api.github.com/user/emails", &emails); err != nil {
		return nil, err
	}

	email := ""
	emailVerified := false
	for _, e := range emails {
		if e.Primary && e.Verified {
			email = strings.ToLower(strings.TrimSpace(e.Email))
			emailVerified = true
			break
		}
	}
	if email == "" {
		for _, e := range emails {
			if e.Verified {
				email = strings.ToLower(strings.TrimSpace(e.Email))
				emailVerified = true
				break
			}
		}
	}

	display := strings.TrimSpace(ghUser.Name)
	if display == "" {
		display = strings.TrimSpace(ghUser.Login)
	}

	return &socialIdentity{
		Provider:      "github",
		Subject:       strconv.FormatInt(ghUser.ID, 10),
		Email:         email,
		EmailVerified: emailVerified,
		DisplayName:   display,
		AvatarURL:     strings.TrimSpace(ghUser.AvatarURL),
		Profile: map[string]any{
			"id":         ghUser.ID,
			"login":      ghUser.Login,
			"name":       ghUser.Name,
			"avatar_url": ghUser.AvatarURL,
		},
	}, nil
}

// exchangeOIDC serves every provider whose Issuer is a real discovery URL. The
// identity is stamped with p.Name, not a literal: two providers writing the same
// provider string would collide in social_identities, where (provider, subject)
// is what links an account.
func (s *SocialAuthService) exchangeOIDC(ctx context.Context, p oauthProvider, redirect, code string) (*socialIdentity, error) {
	oidcProv, err := oidc.NewProvider(ctx, p.Issuer)
	if err != nil {
		return nil, err
	}

	o := &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.Secret,
		RedirectURL:  redirect,
		Endpoint:     oidcProv.Endpoint(),
		Scopes:       p.Scopes,
	}

	tok, err := o.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("missing id_token")
	}

	verifier := oidcProv.Verifier(&oidc.Config{ClientID: p.ClientID})
	idt, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}

	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := idt.Claims(&claims); err != nil {
		return nil, err
	}

	return &socialIdentity{
		Provider:      p.Name,
		Subject:       strings.TrimSpace(claims.Sub),
		Email:         strings.ToLower(strings.TrimSpace(claims.Email)),
		EmailVerified: claims.EmailVerified,
		DisplayName:   strings.TrimSpace(claims.Name),
		AvatarURL:     strings.TrimSpace(claims.Picture),
		Profile: map[string]any{
			"sub":            claims.Sub,
			"email":          claims.Email,
			"email_verified": claims.EmailVerified,
			"name":           claims.Name,
			"picture":        claims.Picture,
		},
	}, nil
}

func getJSON(client *http.Client, url string, out any) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

const OAuthStateCookieName = "__Host-oauth_state"

const oauthStateTTL = 10 * time.Minute

var ErrInvalidOAuthState = errors.New("invalid or expired oauth state")

func (s *SocialAuthService) consumeState(ctx context.Context, provider, state, stateCookie string) (*control.OAuthState, error) {
	state = strings.TrimSpace(state)
	stateCookie = strings.TrimSpace(stateCookie)
	if state == "" || stateCookie == "" {
		return nil, ErrInvalidOAuthState
	}

	rec, err := s.states.Consume(ctx, hashToken(state), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, ErrInvalidOAuthState
	}

	if subtle.ConstantTimeCompare([]byte(rec.CookieHash), []byte(hashToken(stateCookie))) != 1 {
		return nil, ErrInvalidOAuthState
	}

	if !strings.EqualFold(rec.Provider, provider) {
		return nil, ErrInvalidOAuthState
	}

	return rec, nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
