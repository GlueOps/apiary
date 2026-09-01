package main

import (
	"fmt"
	"io"
	"os"

	"ariga.io/atlas-provider-gorm/gormschema"
	"github.com/glueops/apiary/internal/models/control"
)

func main() {

	stmts, err := gormschema.New("postgres").Load(

		&control.User{},
		&control.Organization{},
		&control.Account{},
		&control.Membership{},

		&control.TokenSession{},
		&control.AuthFailure{},
		&control.AuthChallenge{},
		&control.AuthAccessRule{},
		&control.AuthAuditEvent{},

		&control.UserTOTP{},
		&control.UserPasskey{},

		&control.OAuthState{},

		&control.PasswordResetToken{},
		&control.EmailChangeRequest{},
		&control.EmailVerificationToken{},
		&control.Invitation{},
		&control.APIKey{},

		// The only fleet row left in control. It is redeemed by an
		// unauthenticated caller who has no organization yet, so it cannot live in
		// a tenant database — everything else about a fleet does.
		&control.BeeEnrolmentToken{},

		&control.PlatformPolicy{},
		&control.OrgPolicy{},

		// Feature flags are a SIBLING of the policy tables, not an extension.
		// Policies merge with OR and only tighten; a flag's narrower level
		// overrides in either direction. Two tables rather than one nullable
		// organization_id, because the unique index that shape needs is partial
		// and cannot be expressed in GORM tags — so Atlas would diff it away.
		&control.PlatformFeatureFlag{},
		&control.OrgFeatureFlag{},
		&control.ImpersonationGrant{},

		&control.DomainCohortFinding{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}
	if _, err := io.WriteString(os.Stdout, stmts); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write schema: %v\n", err)
		os.Exit(1)
	}
}
