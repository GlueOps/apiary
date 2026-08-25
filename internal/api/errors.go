package api

import (
	"errors"

	"github.com/danielgtaylor/huma/v2"
)

// stripErrorValues removes the submitted value from every validation error
// detail, globally, for every operation this service exposes.
//
// huma echoes the offending value back to help a developer see what they sent
// (ErrorDetail.Value, "the value at the given location"). For a body that
// failed a REQUIRED-property check the value it echoes is the whole decoded
// body — so a request that omits one field gets the rest of itself back,
// including anything secret in it.
//
// That is not theoretical here. Before this, a 422 from:
//
//	POST /auth/login          echoed the password
//	POST /auth/register       echoed the password
//	POST /auth/reset-password echoed the new password
//	POST /estates             echoed the provider secret access key
//
// A 4xx body is precisely the thing client-side error trackers, proxies and
// support tickets capture and retain, so the blast radius of an echoed
// credential is much wider than the request that caused it.
//
// The fix is global rather than per-handler on purpose. Suppressing it only
// where a secret is known to live means the next endpoint that accepts one
// leaks by default, and whoever adds it will have no reason to think about
// this. The message and location are preserved, so an operator still learns
// WHICH field was wrong — they simply are not handed back what they sent.
func stripErrorValues() {
	base := huma.NewError

	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		stripValues(errs)
		return base(status, msg, errs...)
	}

	baseCtx := huma.NewErrorWithContext
	huma.NewErrorWithContext = func(c huma.Context, status int, msg string, errs ...error) huma.StatusError {
		stripValues(errs)
		return baseCtx(c, status, msg, errs...)
	}
}

// stripValues clears ErrorDetail.Value on every detail, unwrapping as it goes.
//
// errors.As rather than a type assertion: a detail that has been wrapped —
// which any future error-annotating middleware would do — would sail past a
// bare assertion and leak the value it carries. The check that removes a secret
// must not be defeated by someone adding context to an error.
func stripValues(errs []error) {
	for _, err := range errs {
		var d *huma.ErrorDetail
		if errors.As(err, &d) {
			d.Value = nil
		}
	}
}
