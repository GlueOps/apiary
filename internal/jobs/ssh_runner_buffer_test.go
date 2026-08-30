package jobs

import (
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"
	"testing"
)

// TestLockedBufferSurvivesConcurrentWriters is the regression test for a data
// race in SSHRunner.Run.
//
// x/crypto/ssh starts one copy goroutine per stream, so setting Stdout and
// Stderr to the same plain bytes.Buffer had two goroutines writing it. The
// symptom was not garbled output but MISSING output: a command that worked over
// plain ssh returned an empty string through the runner, intermittently. Every
// caller in this package reads output to decide something, so an empty reply
// reads as "no keys to prune" or "no nodes in corosync.conf" — a silent false
// negative on the checks that exist to prevent damage.
//
// Run with -race for this to mean anything; without it, it only checks that no
// bytes are lost.
func TestLockedBufferSurvivesConcurrentWriters(t *testing.T) {
	const writers, perWriter = 8, 200

	var lb lockedBuffer
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perWriter; j++ {
				if _, err := lb.Write([]byte("x")); err != nil {
					t.Errorf("write: %v", err)
					return
				}
			}
		}()
	}
	// Concurrent readers too: the ctx.Done() path in Run reads the buffer while
	// the copy goroutines are still writing, because nothing has waited for them.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perWriter; j++ {
				_ = lb.String()
			}
		}()
	}
	wg.Wait()

	got := lb.String()
	if want := writers * perWriter; len(got) != want {
		t.Fatalf("lost bytes under concurrency: got %d, want %d", len(got), want)
	}
	if strings.Trim(got, "x") != "" {
		t.Fatal("buffer contents corrupted under concurrency")
	}
}

// A REFUSED CONNECTION IS NOT A REFUSED KEY.
//
// A lab host failed onboarding here while its sshd was reachable before and
// after. The reload that runs immediately before was the natural suspect and was
// ruled out — the port stayed open across it and socket activation was disabled
// — so the transient is unexplained. The retry stands on its own reasoning: a
// refusal between two reachable moments is transient, and the alternative is
// failing an onboarding on a single dial.
func TestConnectionRefusalIsRetryableAndAuthFailureIsNot(t *testing.T) {
	refused := &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}
	if !isConnectionRefused(refused) {
		t.Fatal("a refused dial was not treated as retryable")
	}
	if !isConnectionRefused(io.EOF) {
		t.Fatal("a listener disappearing mid-handshake was not treated as retryable")
	}

	// An authentication failure is the thing this reconnect exists to PROVE.
	// Retrying it would turn a real finding into a slow one.
	authErr := errors.New("ssh: handshake failed: ssh: unable to authenticate")
	if isConnectionRefused(authErr) {
		t.Fatal("an authentication failure was treated as a transport problem; the reconnect " +
			"exists to catch exactly this and must fail fast")
	}
	if isConnectionRefused(nil) {
		t.Fatal("nil reported as refused")
	}
}
