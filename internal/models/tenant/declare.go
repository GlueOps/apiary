package tenant

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Declaring a machine is how a bare host — one nobody has touched, running no
// agent, in no cluster — enters the fleet.
//
// It was the missing entry point of the automated path. Rows were only ever
// created by the capacity sync (which needs the host to already be a node in a
// configured cluster) or by an agent enrolling (which needs somebody to have run
// a command on it by hand). Neither describes a machine that has just been
// racked, so the automated path had a trigger, a ticker, a worker and no way to
// name a host for it to work on.

// MaxDeclaredHosts caps one bulk declaration.
//
// A rack is the unit this exists for, so the limit is generous — but not
// unbounded: each accepted address becomes an SSH connection attempt on the next
// tick, and a paste that was meant to be a hostname list and was actually a log
// file should be refused rather than turned into a thousand connections.
const MaxDeclaredHosts = 512

// DeclaredHost is one machine an operator is asking the fleet to bring in.
type DeclaredHost struct {
	// Address is how the control plane reaches it: a hostname or an IP. Not a
	// URL and not host:port — the port is SSH's, and letting it vary here would
	// mean every downstream command needs to carry one.
	Address string
	// User to connect as. Empty means root, which is what a Proxmox node has.
	User string
}

// ParseHostList turns pasted text into addresses.
//
// Accepts what an operator actually has to hand: one host per line, or several
// separated by commas or spaces, with blank lines and #-comments ignored. That
// list usually comes from a spreadsheet, an inventory file or a terminal, and
// refusing anything but a strict format means the operator reformats it by hand
// — which is where a typo enters.
func ParseHostList(text string) ([]string, error) {
	var out []string
	seen := map[string]bool{}

	for _, line := range strings.Split(text, "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		for _, field := range strings.FieldsFunc(line, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == ';'
		}) {
			addr := strings.TrimSpace(field)
			if addr == "" {
				continue
			}
			if err := ValidateHostAddress(addr); err != nil {
				return nil, err
			}
			// Deduplicated. The same host twice would be two rows racing to onboard
			// one machine, and the unique index on address is not what should be
			// reporting that to an operator who pasted a list with a repeat in it.
			key := strings.ToLower(addr)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, addr)
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no addresses found: give one host per line, or several "+
			"separated by commas; blank lines and lines starting with %q are ignored", "#")
	}
	if len(out) > MaxDeclaredHosts {
		return nil, fmt.Errorf("%d addresses is more than the %d this accepts at once; each one "+
			"becomes an SSH connection attempt, so a paste that was meant to be a host list and "+
			"was actually something else should be refused rather than dialled",
			len(out), MaxDeclaredHosts)
	}
	return out, nil
}

// ValidateHostAddress refuses what cannot be connected to.
//
// The check is deliberately shallow — a name that resolves today may not
// tomorrow, and the onboarding job reports an unreachable host far better than a
// guess here could. What it does catch is the shapes that are wrong on their
// face, because each of them fails much later and much less clearly: a URL
// becomes an ssh host called "https", and host:port silently drops the port.
func ValidateHostAddress(addr string) error {
	a := strings.TrimSpace(addr)
	if a == "" {
		return fmt.Errorf("an empty address is not a host")
	}
	if len(a) > 253 {
		return fmt.Errorf("%q is longer than a hostname can be", truncate(a))
	}
	if strings.Contains(a, "://") {
		u, err := url.Parse(a)
		hint := a
		if err == nil && u.Host != "" {
			hint = u.Host
		}
		return fmt.Errorf("%q is a URL; give just the host, as %q", truncate(a), truncate(hint))
	}
	if strings.ContainsAny(a, " \t") {
		return fmt.Errorf("%q contains a space, so it is not one address", truncate(a))
	}
	// A bare IPv6 address is fine; host:port is not. Distinguished by whether the
	// whole string parses as an address.
	if net.ParseIP(a) != nil {
		return nil
	}
	if h, p, err := net.SplitHostPort(a); err == nil && p != "" {
		return fmt.Errorf("%q names a port; the control plane connects over SSH, so give just "+
			"the host, as %q", truncate(a), truncate(h))
	}
	for _, r := range a {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '-', r == '_', r == ':':
		default:
			return fmt.Errorf("%q contains %q, which a hostname cannot", truncate(a), string(r))
		}
	}
	return nil
}

func truncate(s string) string {
	if len(s) <= 60 {
		return s
	}
	return s[:57] + "..."
}
