package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/glueops/apiary/internal/utils"
	"github.com/glueops/apiary/internal/workflow"
	"github.com/riverqueue/river"
)

// SendEmailWorker delivers a queued email.
//
// It returns the send error rather than swallowing it, which is the entire
// reason mail moved onto the queue: River retries a failing job with backoff and
// leaves a visible row when it finally gives up. The previous arrangement —
// a goroutine that logged and dropped — lost a registration confirmation to any
// momentary SMTP fault, and the account was then stuck behind the confirmation
// gate with nothing to confirm.
// Mailer is the one thing this worker needs from the mail layer.
//
// Declared here rather than imported: internal/service already imports this
// package (the admission gate shares its counting query), so depending on
// service.EmailService directly would be an import cycle. A one-method interface
// at the consumer is the ordinary Go answer, and cmd/worker.go — which imports
// both — supplies the concrete sender.
type Mailer interface {
	Send(to, subject, htmlBody string) error
}

type SendEmailWorker struct {
	river.WorkerDefaults[workflow.SendEmailArgs]
	mail      Mailer
	masterKey []byte
}

func NewSendEmailWorker(mail Mailer, masterKey []byte) *SendEmailWorker {
	if mail == nil {
		panic("jobs: SendEmailWorker requires an EmailService")
	}
	if len(masterKey) == 0 {
		// Without the key the body cannot be opened, so every job would fail and
		// retry until it exhausted its attempts. Refusing at construction turns a
		// slow drip of failures into one loud startup error.
		panic("jobs: SendEmailWorker requires the encryption master key to open sealed bodies")
	}
	return &SendEmailWorker{mail: mail, masterKey: masterKey}
}

func (w *SendEmailWorker) Work(ctx context.Context, job *river.Job[workflow.SendEmailArgs]) error {
	key, err := utils.DeriveKey(w.masterKey, utils.KeyPurposeQueuePayload)
	if err != nil {
		return river.JobCancel(fmt.Errorf("derive queue key: %w", err))
	}

	body, err := workflow.OpenEmailBody(key, &job.Args)
	if err != nil {
		// Unopenable means the master key has changed, or the row was tampered
		// with. Neither is fixed by trying again, and each retry would re-log a
		// scary error, so cancel the job instead of burning its attempts.
		slog.ErrorContext(ctx, "cannot open a sealed email body; cancelling the job",
			"job_id", job.ID, "to", job.Args.To, "error", err)
		return river.JobCancel(fmt.Errorf("open sealed body: %w", err))
	}

	if err := w.mail.Send(job.Args.To, job.Args.Subject, body); err != nil {
		// Retryable: this is the transient case the queue exists for.
		return fmt.Errorf("send email to %s: %w", job.Args.To, err)
	}
	return nil
}
