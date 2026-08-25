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
