//go:build unix

package firewall

import "syscall"

// detachedAttr puts the rollback in its own session and process group.
//
// Setsid rather than only Setpgid: a session leader has no controlling terminal,
// so the rollback is not killed by a hangup when the shell or SSH session that
// started the agent goes away — which is exactly the moment a host that just
// lost connectivity would otherwise lose its undo as well.
func detachedAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
