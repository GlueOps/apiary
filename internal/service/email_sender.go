package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/glueops/apiary/internal/utils"
	"github.com/glueops/apiary/internal/workflow"
)

// EmailSender is what the services depend on. They render a message and hand it
// over; whether it goes straight to SMTP or onto a queue is not their business.
//
// The seam exists because every caller previously spawned its own goroutine,
// logged whatever came back, and dropped the message — four copies of the same
// fire-and-forget mistake. Behind an interface there is one delivery policy, and
// changing it does not mean editing four services.
type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// DirectEmailSender sends synchronously over SMTP, with no retry.
//
// Kept for development and tests, where a queue and a worker are more
// ceremony than the situation deserves. Production wiring should use
// QueuedEmailSender: a synchronous send inside a request means a slow mail
// server becomes a slow API, and a failed one is simply lost.
type DirectEmailSender struct{ svc *EmailService }

func NewDirectEmailSender(svc *EmailService) *DirectEmailSender {
	return &DirectEmailSender{svc: svc}
}

func (d *DirectEmailSender) Send(_ context.Context, to, subject, htmlBody string) error {
	if d == nil || d.svc == nil {
		return errors.New("email: no mail service configured")
	}
	return d.svc.Send(to, subject, htmlBody)
}

// QueuedEmailSender seals the message body and puts it on the job queue.
//
// Sealing is not optional decoration. The body carries a live token — a
// confirmation link, a password reset, an invitation — and River stores job args
// as JSONB in the control database and keeps them after the job completes. In
// clear, that is a working credential sitting beside the table that deliberately
// stores only its SHA-256; anyone who could read the jobs table could redeem
// every pending link. Sealed, that table holds ciphertext.
type QueuedEmailSender struct {
	jobs     workflow.Enqueuer
	key      []byte
	fallback EmailSender
}

// NewQueuedEmailSender derives its own key from the master key, under a purpose
// used for nothing else — so a queue payload cannot be opened with a tenant key
// and vice versa.
//
// fallback may be nil. When set it is used only if enqueueing fails, which keeps
// a broken queue from silently swallowing a registration confirmation.
func NewQueuedEmailSender(jobs workflow.Enqueuer, masterKey []byte, fallback EmailSender) (*QueuedEmailSender, error) {
	if jobs == nil {
		return nil, errors.New("email: queued sender requires an enqueuer")
	}
	key, err := utils.DeriveKey(masterKey, utils.KeyPurposeQueuePayload)
	if err != nil {
		return nil, fmt.Errorf("email: derive queue key: %w", err)
	}
	return &QueuedEmailSender{jobs: jobs, key: key, fallback: fallback}, nil
}

func (q *QueuedEmailSender) Send(ctx context.Context, to, subject, htmlBody string) error {
	args, err := workflow.SealEmailBody(q.key, to, subject, htmlBody)
	if err != nil {
		return err
	}
	if err := q.jobs.EnqueueSendEmail(ctx, args); err == nil {
		return nil
	} else if q.fallback == nil {
		return err
	}
	// Falling back to an inline send only when the ENQUEUE failed. Losing a
	// registration confirmation because the queue is unavailable is worse than
	// sending it synchronously and without retries; a delivery failure, by
	// contrast, is exactly what the queue is there to retry and must not be
	// short-circuited here.
	return q.fallback.Send(ctx, to, subject, htmlBody)
}

var _ EmailSender = (*QueuedEmailSender)(nil)
var _ EmailSender = (*DirectEmailSender)(nil)
