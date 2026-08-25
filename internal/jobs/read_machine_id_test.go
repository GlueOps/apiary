package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// echoRunner answers every command with a fixed output, which is all
// ReadMachineID needs.
type echoRunner struct {
	out string
	err error
}

func (e *echoRunner) Run(context.Context, string) (string, error)          { return e.out, e.err }
func (e *echoRunner) Upload(context.Context, string, uint32, []byte) error { return nil }
func (e *echoRunner) Reconnect(context.Context) error                      { return nil }

// TestReadMachineIDAsksTheHost covers the case the ticker created: a machine
// record discovered by the capacity sync carries no machine id, and the digest
// it would carry is not the raw value this job needs anyway.
func TestReadMachineIDAsksTheHost(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		out  string
		want string
	}{
		{"a plain id", "8f3c1d2e4b5a6978", "8f3c1d2e4b5a6978"},
		{"a trailing newline, which cat always produces", "8f3c1d2e4b5a6978\n", "8f3c1d2e4b5a6978"},
		{"surrounding whitespace", "  8f3c1d2e4b5a6978  \n", "8f3c1d2e4b5a6978"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ReadMachineID(context.Background(), &echoRunner{out: tc.out})
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if got != tc.want {
				t.Fatalf("machine id = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestReadMachineIDRefusesToGuess. Inventing or picking a value decides which
// machine record a heartbeat belongs to, so a wrong answer merges two hosts into
// one row or waits forever for a heartbeat under an id nothing will send.
func TestReadMachineIDRefusesToGuess(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		r    *echoRunner
		want string
	}{
		{"an empty file", &echoRunner{out: "\n"}, "reports no machine id"},
		{"nothing at all", &echoRunner{out: ""}, "reports no machine id"},
		{"more than one value", &echoRunner{out: "aaaa\nbbbb\n"}, "not a single value"},
		{"the command failing", &echoRunner{err: errors.New("exit 1")}, "read machine id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ReadMachineID(context.Background(), tc.r)
			if err == nil {
				t.Fatalf("got %q, want a refusal", got)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

// TestReadMachineIDFallsBackToDbus, for hosts where /etc is read-only or the
// value was seeded by dbus.
func TestReadMachineIDLooksInBothPlaces(t *testing.T) {
	t.Parallel()
	r := &recordingRunner{out: "abc123"}
	if _, err := ReadMachineID(context.Background(), r); err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(r.cmd, "/etc/machine-id") {
		t.Fatalf("command %q does not read /etc/machine-id", r.cmd)
	}
	if !strings.Contains(r.cmd, "/var/lib/dbus/machine-id") {
		t.Fatalf("command %q has no fallback for a host where /etc is read-only", r.cmd)
	}
}

type recordingRunner struct {
	cmd string
	out string
}

func (r *recordingRunner) Run(_ context.Context, cmd string) (string, error) {
	r.cmd = cmd
	return r.out, nil
}
func (r *recordingRunner) Upload(context.Context, string, uint32, []byte) error { return nil }
func (r *recordingRunner) Reconnect(context.Context) error                      { return nil }
