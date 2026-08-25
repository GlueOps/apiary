package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/utils"
	"github.com/glueops/apiary/internal/workflow"
)

type esFakeEnqueuer struct {
	// Inherits no-op implementations of the job kinds this fake has no opinion
	// about — the fleet lifecycle ones. Overridden methods below still win.
	workflow.MockEnqueuer

	sent []*workflow.SendEmailArgs
	fail error
}

func (f *esFakeEnqueuer) EnqueueProvisionTenant(context.Context, *workflow.ProvisionTenantArgs) error {
	return nil
}
func (f *esFakeEnqueuer) EnqueueDestroyTenant(context.Context, *workflow.DestroyTenantArgs) error {
	return nil
}
func (f *esFakeEnqueuer) EnqueueSendEmail(_ context.Context, a *workflow.SendEmailArgs) error {
	if f.fail != nil {
		return f.fail
	}
	f.sent = append(f.sent, a)
	return nil
}

type esRecordingSender struct{ calls int }

func (r *esRecordingSender) Send(context.Context, string, string, string) error {
	r.calls++
	return nil
}

func esMasterKey() []byte {
	k := make([]byte, utils.MasterKeyLen)
	for i := range k {
		k[i] = byte(i + 1)
	}
	return k
}

// This is the assertion the whole sealed-payload design exists for.
//
// River persists job args as JSONB in the control database and keeps them after
// the job completes. An email body carries a live token — a confirmation link, a
// password reset, an invitation — so enqueueing it in clear would put a working
// credential next to the table that deliberately stores only its SHA-256, which
// is the entire reason that table hashes. Anyone able to read the jobs table
// could then redeem every pending link.
//
// The check is over the SERIALISED job, not the struct, because that is what
// actually reaches the database.
func TestQueuedEmailNeverPutsTheTokenInTheJobInClear(t *testing.T) {
	const secretToken = "5f3c9d2ae1b74c08a6d1e9f4b7c2a0d3"
	body := `<a href="https://apiary.test/verify-email?token=` + secretToken + `">Confirm</a>`

	q := &esFakeEnqueuer{}
	sender, err := NewQueuedEmailSender(q, esMasterKey(), nil)
	if err != nil {
		t.Fatalf("new queued sender: %v", err)
	}
	if err := sender.Send(context.Background(), "invitee@example.com", "Confirm", body); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(q.sent) != 1 {
		t.Fatalf("enqueued %d jobs, want 1", len(q.sent))
	}

	raw, err := json.Marshal(q.sent[0])
	if err != nil {
		t.Fatalf("marshal job args: %v", err)
	}
	if strings.Contains(string(raw), secretToken) {
		t.Fatalf("the raw token appears in the serialised job args, which River stores as JSONB "+
			"and retains after completion — a reader of the jobs table could redeem it. args=%s", raw)
	}
	if strings.Contains(string(raw), "verify-email") {
		t.Fatalf("the message body is not sealed; the job carries the link in clear. args=%s", raw)
	}
}

// Sealing is worthless if the worker cannot open it. Round-trips through the
// exact pair the two processes use, so an AAD or key-purpose change on one side
// fails here rather than at 3am in the worker.
func TestQueuedEmailBodyRoundTripsThroughTheWorkerHalf(t *testing.T) {
	const body = "<p>hello</p>"
	q := &esFakeEnqueuer{}
	sender, err := NewQueuedEmailSender(q, esMasterKey(), nil)
	if err != nil {
		t.Fatalf("new queued sender: %v", err)
	}
	if err := sender.Send(context.Background(), "someone@example.com", "Subject", body); err != nil {
		t.Fatalf("send: %v", err)
	}

	key, err := utils.DeriveKey(esMasterKey(), utils.KeyPurposeQueuePayload)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	got, err := workflow.OpenEmailBody(key, q.sent[0])
	if err != nil {
		t.Fatalf("worker could not open the body it was sent: %v", err)
	}
	if got != body {
		t.Fatalf("body = %q, want %q", got, body)
	}
}

// The body is bound to its recipient, so a job row rewritten to point at another
// address cannot be delivered. Without this an attacker with write access to the
// queue could redirect somebody else's confirmation link to a mailbox they own.
func TestQueuedEmailBodyIsBoundToItsRecipient(t *testing.T) {
	q := &esFakeEnqueuer{}
	sender, _ := NewQueuedEmailSender(q, esMasterKey(), nil)
	if err := sender.Send(context.Background(), "victim@example.com", "s", "<p>secret link</p>"); err != nil {
		t.Fatalf("send: %v", err)
	}

	tampered := *q.sent[0]
	tampered.To = "attacker@example.com"

	key, _ := utils.DeriveKey(esMasterKey(), utils.KeyPurposeQueuePayload)
	if _, err := workflow.OpenEmailBody(key, &tampered); err == nil {
		t.Fatal("a body resealed for one recipient opened after the address was changed — the AAD " +
			"binding is not in place, so the queue can be used to redirect somebody's link")
	}
}

// A queue that is down must not silently swallow a registration confirmation:
// without the email the account exists, cannot log in past the confirmation
// gate, and its organization is never provisioned.
func TestQueuedEmailFallsBackWhenEnqueueingFails(t *testing.T) {
	fallback := &esRecordingSender{}
	sender, _ := NewQueuedEmailSender(&esFakeEnqueuer{fail: errors.New("queue is down")}, esMasterKey(), fallback)

	if err := sender.Send(context.Background(), "a@example.com", "s", "b"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if fallback.calls != 1 {
		t.Fatalf("fallback called %d times, want 1 — the message was dropped", fallback.calls)
	}
}

func TestQueuedEmailReportsFailureWhenThereIsNoFallback(t *testing.T) {
	sender, _ := NewQueuedEmailSender(&esFakeEnqueuer{fail: errors.New("queue is down")}, esMasterKey(), nil)
	if err := sender.Send(context.Background(), "a@example.com", "s", "b"); err == nil {
		t.Fatal("enqueue failed with no fallback configured, yet Send reported success")
	}
}
