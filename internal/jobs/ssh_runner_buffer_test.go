package jobs

import (
	"strings"
	"sync"
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
