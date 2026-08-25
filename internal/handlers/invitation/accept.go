package invitation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
)

// The accept surface is PUBLIC, and that is a design decision rather than an
// oversight worth re-examining every time someone reads this file.
//
// The person redeeming an invitation usually has no account here yet. Putting
// these endpoints behind the bearer middleware would mean they had to register
// first — and registering mints an organization of their own, which is exactly
// the thing an invitee should not be doing. Possession of the emailed token is
// the authorization, and the token is bound to one address, one organization
// and one role.

type PreviewRequest struct {
	Token string `query:"token" required:"true" maxLength:"128"`
}

type PreviewResponse struct {
	Body struct {
		Email        string `json:"email"`
		Organization string `json:"organization"`
		Role         string `json:"role"`
		NeedsAccount bool   `json:"needs_account" doc:"True when this address has no account yet, so the page must ask for a name and password. False means sign in to accept."`
	}
}

type AcceptRequest struct {
	Body struct {
		Token     string `json:"token" required:"true" maxLength:"128"`
		Password  string `json:"password,omitempty" minLength:"8" maxLength:"128" doc:"Only for an address that has no account yet. Supplying one for an existing account is refused, not ignored."`
		FirstName string `json:"first_name,omitempty" maxLength:"100"`
		LastName  string `json:"last_name,omitempty" maxLength:"100"`
	}
}

type AcceptResponse struct {
	Body struct {
		Accepted       bool   `json:"accepted"`
		Organization   string `json:"organization"`
		Email          string `json:"email"`
		AccountCreated bool   `json:"account_created"`
		Message        string `json:"message"`
	}
}

func mapAcceptErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, service.ErrInvitationInvalid):
		return huma.Error404NotFound("this invitation is invalid, has expired, or has already been used")
	case errors.Is(err, service.ErrPasswordRequired):
		return huma.Error422UnprocessableEntity("choose a password to finish setting up your account")
	case errors.Is(err, service.ErrPasswordNotAccepted):
		// Its own status and wording. This is the branch that stops an
		// invitation to a known address being an account takeover, and the
		// invitee needs to be told to sign in rather than left guessing.
		return huma.Error409Conflict("this address already has an account — sign in to accept the invitation")
	case errors.Is(err, service.ErrWeakPassword):
		return huma.Error422UnprocessableEntity(err.Error())
	case errors.Is(err, service.ErrRegistrationName):
		return huma.Error422UnprocessableEntity("a name is required")
	case errors.Is(err, repo.ErrAccessDenied):
		return huma.Error403Forbidden("this invitation cannot be accepted")
	default:
		slog.ErrorContext(ctx, "accepting an invitation failed", "error", err)
		return huma.Error500InternalServerError("could not accept this invitation")
	}
}

// RegisterPublic mounts the unauthenticated accept surface. It must be handed
// the PUBLIC huma.API — see the package comment.
func RegisterPublic(hapi huma.API, deps *app.Deps) {
	svc := newService(deps)

	huma.Register(hapi, huma.Operation{
		OperationID: "invitations-preview",
		Method:      http.MethodGet,
		Path:        "/invitations/preview",
		Summary:     "Describe an invitation",
		Description: "Lets the accept page render before the invitee commits to anything: which organization, which address, and whether they need to choose a password or sign in. " +
			"Holding the token is the authorization — it was mailed to the address it names.",
		Tags: []string{"Invitations"},
	}, func(ctx context.Context, in *PreviewRequest) (*PreviewResponse, error) {
		preview, err := svc.Preview(ctx, in.Token)
		if err != nil {
			return nil, mapAcceptErr(ctx, err)
		}
		out := &PreviewResponse{}
		out.Body.Email = preview.Email
		out.Body.Organization = preview.Organization
		out.Body.Role = string(preview.Role)
		out.Body.NeedsAccount = preview.NeedsAccount
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "invitations-accept",
		Method:        http.MethodPost,
		Path:          "/invitations/accept",
		Summary:       "Accept an invitation",
		DefaultStatus: http.StatusOK,
		Description: "Issues NO session. A new invitee chooses a password here and then signs in through /auth/login; an existing account gains the membership and signs in as it already could. " +
			"Keeping session minting on the login path alone means lockout, MFA and the email-confirmation gate cannot be skipped by arriving through this door.",
		Tags: []string{"Invitations"},
	}, func(ctx context.Context, in *AcceptRequest) (*AcceptResponse, error) {
		result, err := svc.Accept(ctx, service.AcceptInvitationInput{
			Token:     in.Body.Token,
			Password:  in.Body.Password,
			FirstName: in.Body.FirstName,
			LastName:  in.Body.LastName,
		})
		if err != nil {
			return nil, mapAcceptErr(ctx, err)
		}

		out := &AcceptResponse{}
		out.Body.Accepted = true
		out.Body.Organization = result.Organization
		out.Body.Email = result.Email
		out.Body.AccountCreated = result.AccountCreated
		if result.AccountCreated {
			out.Body.Message = "Your account is ready. Sign in to continue."
		} else {
			out.Body.Message = "You have joined " + result.Organization + ". Sign in to continue."
		}
		return out, nil
	})
}
