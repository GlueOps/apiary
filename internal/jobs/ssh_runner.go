package jobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHRunner executes commands on a hypervisor over SSH.
//
// It holds the credential rather than a connection, so Reconnect can prove a
// configuration change by dialling again with the same key — which is the only
// evidence that a change to sshd took effect, since the session that made the
// change survives a reload regardless.
type SSHRunner struct {
	addr   string
	user   string
	signer ssh.Signer

	// hostKey is what the host presented on the FIRST connection, remembered so
	// every later dial can insist on the same one.
	//
	// Trust on first use, and stated as such rather than dressed up: nothing here
	// knows the host key in advance, so the first connection cannot be
	// authenticated. What this does buy is that a machine-in-the-middle appearing
	// LATER — between onboarding and the reconnect that proves the change — is
	// refused, which is the window where a swapped host would otherwise be handed
	// the fleet key.
	hostKey ssh.PublicKey

	mu     sync.Mutex
	client *ssh.Client
}

func NewSSHRunner(addr, user string, privateKeyPEM []byte) (*SSHRunner, error) {
	signer, err := ssh.ParsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("onboard: parse fleet key: %w", err)
	}
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, strconv.Itoa(22))
	}
	return &SSHRunner{addr: addr, user: user, signer: signer}, nil
}

func (s *SSHRunner) config() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            s.user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(s.signer)},
		HostKeyCallback: s.checkHostKey,
		Timeout:         15 * time.Second,
	}
}

// checkHostKey pins whatever the host presented first.
//
// ssh.InsecureIgnoreHostKey would accept a different host on every dial, which
// would defeat the reconnect proof entirely: the "new connection" could be to
// something else.
func (s *SSHRunner) checkHostKey(_ string, _ net.Addr, key ssh.PublicKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hostKey == nil {
		s.hostKey = key
		return nil
	}
	if !bytes.Equal(s.hostKey.Marshal(), key.Marshal()) {
		return fmt.Errorf("onboard: host key changed mid-onboarding (was %s, now %s)",
			ssh.FingerprintSHA256(s.hostKey), ssh.FingerprintSHA256(key))
	}
	return nil
}

func (s *SSHRunner) dial(ctx context.Context) (*ssh.Client, error) {
	d := net.Dialer{Timeout: 15 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, s.addr, s.config())
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

// Connect opens the first session.
func (s *SSHRunner) Connect(ctx context.Context) error {
	c, err := s.dial(ctx)
	if err != nil {
		return fmt.Errorf("onboard: connect %s: %w", s.addr, err)
	}
	s.mu.Lock()
	s.client = c
	s.mu.Unlock()
	return nil
}

// Reconnect replaces the connection with a fresh one.
//
// The OLD connection is closed only after the new one succeeds. If the change
// under test broke authentication, the caller keeps a working session to report
// from and to undo with — closing first would strand the job on the far side of
// its own mistake.
// reconnectSettle bounds the wait for sshd to accept again after a reload.
//
// Vars so a test can shorten them.
var (
	reconnectSettleTimeout  = 20 * time.Second
	reconnectSettleInterval = time.Second
)

func (s *SSHRunner) Reconnect(ctx context.Context) error {
	// A REFUSED CONNECTION IS NOT A REFUSED KEY, and this dialled once.
	//
	// A lab host failed onboarding here with "connection refused" while its sshd
	// was reachable before and after, and stayed `failed` while four identical
	// hosts passed.
	//
	// THE OBVIOUS CAUSE WAS RULED OUT rather than assumed: the step before this
	// reloads sshd, so a reload window was the natural suspect — but polling the
	// port across `systemctl reload ssh` on that host showed it open throughout,
	// and socket activation was disabled there. Whatever the transient was, it
	// was not the reload.
	//
	// The retry is kept anyway, because it is cheap and correct for its own
	// reason: a connection refused between two reachable moments is transient by
	// definition, and the alternative is failing an onboarding on one dial.
	//
	// Only CONNECTION failures are retried. An authentication failure means the
	// key genuinely does not work, which is the thing this reconnect exists to
	// prove — retrying it would turn a real finding into a slow one, and the
	// whole point of the proof is that it can fail.
	deadline := time.Now().Add(reconnectSettleTimeout)
	var fresh *ssh.Client
	var err error
	for {
		fresh, err = s.dial(ctx)
		if err == nil || !isConnectionRefused(err) || time.Now().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(reconnectSettleInterval):
		}
	}
	if err != nil {
		return fmt.Errorf("onboard: reconnect %s: %w", s.addr, err)
	}
	s.mu.Lock()
	old := s.client
	s.client = fresh
	s.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return nil
}

func (s *SSHRunner) Run(ctx context.Context, cmd string) (string, error) {
	s.mu.Lock()
	c := s.client
	s.mu.Unlock()
	if c == nil {
		return "", fmt.Errorf("onboard: not connected")
	}
	sess, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer func() { _ = sess.Close() }()

	// ONE buffer, but a locked one. x/crypto/ssh starts a separate copy goroutine
	// for each of Stdout and Stderr, so handing the same bytes.Buffer to both is
	// a data race — and it does not merely garble output, it LOSES it: a racing
	// grow can drop a whole write. Caught by the race detector against a live
	// host, after a command that works perfectly over plain ssh returned an empty
	// string through this runner, intermittently.
	//
	// That failure mode is the worst available here, because every caller in this
	// package reads a command's output to decide something. An empty reply reads
	// as "no keys to prune", "no nodes in corosync.conf", "nothing to do" — a
	// silent false negative on exactly the checks that exist to prevent damage.
	var out lockedBuffer
	sess.Stdout = &out
	sess.Stderr = &out

	done := make(chan error, 1)
	go func() { done <- sess.Run(cmd) }()

	select {
	case <-ctx.Done():
		// Signal the remote process rather than only abandoning the local side; a
		// dist-upgrade left running after the job gave up is how a host ends up
		// mid-change with nothing watching it.
		_ = sess.Signal(ssh.SIGKILL)
		// Read under the lock: the copy goroutines are still running on this path,
		// since nothing has waited for them.
		return out.String(), ctx.Err()
	case err := <-done:
		return out.String(), err
	}
}

// lockedBuffer is an io.Writer safe for the two concurrent copy goroutines that
// x/crypto/ssh starts when both Stdout and Stderr are set.
//
// Combining the streams is deliberate — callers here want what the command said,
// interleaved as it was said — but combining them requires a writer that can
// actually take two writers.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

func (s *SSHRunner) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil {
		return nil
	}
	err := s.client.Close()
	s.client = nil
	return err
}

// HostKeyFingerprint is what the host presented, for recording against the bee
// so a later change is attributable.
func (s *SSHRunner) HostKeyFingerprint() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hostKey == nil {
		return ""
	}
	return ssh.FingerprintSHA256(s.hostKey)
}

// Upload writes content to a path atomically.
//
// Written to a temporary file beside the destination and then renamed, because
// a rename within one filesystem is atomic: a transfer that dies halfway leaves
// the temporary file rather than a truncated binary at the real path. systemd
// starting a half-copied executable fails in a way that looks like a broken
// agent rather than a broken copy, and that is a long afternoon.
//
// The temporary file is created with the FINAL mode before any content is
// written, so a binary is never briefly readable by more than it should be.
func (s *SSHRunner) Upload(ctx context.Context, path string, mode uint32, content []byte) error {
	tmp := path + ".apiary-tmp"

	s.mu.Lock()
	c := s.client
	s.mu.Unlock()
	if c == nil {
		return fmt.Errorf("onboard: not connected")
	}

	sess, err := c.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close() }()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return err
	}
	var out bytes.Buffer
	sess.Stdout = &out
	sess.Stderr = &out

	// umask 077 then an explicit chmod: the file must never exist, even briefly,
	// with wider permissions than it ends with.
	cmd := fmt.Sprintf("umask 077 && cat > %s && chmod %o %s && mv -f %s %s",
		shellQuote(tmp), mode, shellQuote(tmp), shellQuote(tmp), shellQuote(path))
	if err := sess.Start(cmd); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		_, werr := stdin.Write(content)
		if cerr := stdin.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			done <- werr
			return
		}
		done <- sess.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGKILL)
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("%w (%s)", err, strings.TrimSpace(out.String()))
		}
		return nil
	}
}

// isConnectionRefused reports whether an error is a transport-level refusal
// rather than a rejection of the credential.
//
// Matched on the syscall error, not on the message: the text is produced by the
// resolver and the kernel and differs between platforms, and matching prose is
// how isNotClustered ended up mistaking "standalone" for "unreadable".
func isConnectionRefused(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	// A listener that has gone away mid-handshake surfaces as EOF rather than as
	// a syscall error, and it means the same thing here.
	return errors.Is(err, io.EOF)
}
