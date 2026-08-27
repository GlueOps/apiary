package tenant

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strings"
)

// MachineIDDigest is the ONE form a machine id is ever stored in.
//
// It lives beside the column it feeds rather than in the service that happens to
// call it first, because the two must not be able to drift: enrolment stored the
// digest while the onboarding job carried the RAW value and compared it against
// this column, so the proof-of-success lookup could never match and every onboard
// would have ended in "no heartbeat". That was latent only because nothing
// enqueued the onboarding job yet. Anything that puts a machine id anywhere near
// the database goes through here.
func MachineIDDigest(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

// NodeNameFromAddress derives the Proxmox node name a machine would appear under
// from the address we reach it on: the first label of the hostname.
//
// This is a GUESS, and every caller must treat it as one. It exists because the
// two halves of a machine's identity are learned from different places and never
// at the same time — enrolment learns an address and a machine id, the capacity
// sync learns a hive and a node name — and this is the only thing the two have in
// common until something links them explicitly.
//
// An IP literal yields "", not "10". A first label is only a node name when the
// address is a name; taking one off an address would produce a key that collides
// with every other host on the same /8 and silently merge two machines into one
// record. Empty means "no opinion", and every matcher below refuses to act on it.
func NodeNameFromAddress(addr string) string {
	host := strings.TrimSpace(addr)
	if host == "" {
		return ""
	}
	// SplitHostPort fails on a bare hostname, which is the common case, so its
	// error is not interesting — only its success is.
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if net.ParseIP(host) != nil {
		return ""
	}
	if i := strings.IndexByte(host, '.'); i >= 0 {
		return host[:i]
	}
	return host
}

// MatchMachineByNodeName picks the machine record that a cluster node name refers
// to, from machines belonging to no hive.
//
// This is the capacity sync's half of the merge. A machine onboarded by address
// has a machine id and an address but no hive and no node name, because a node
// name does not exist until it joins a cluster. When it then appears in a
// cluster, the sync knows only (hive, name) — Proxmox's /cluster/resources
// reports no machine id — so without this the sync would find no (hive_id, name)
// row and create a SECOND record for a machine it already had, complete with a
// duplicate of everything except the onboarding history.
//
// AMBIGUITY IS A REFUSAL, not a best guess. Two unclustered machines whose
// addresses share a first label — hv5.dc1.example.com and hv5.dc2.example.com —
// give no evidence which one joined, and attaching an agent's credential and
// machine id to the wrong physical host is a far worse outcome than a duplicate
// row an operator can merge deliberately. Nil means "create a fresh record".
func MatchMachineByNodeName(candidates []Hypervisor, nodeName string) *Hypervisor {
	if nodeName == "" {
		return nil
	}
	var found *Hypervisor
	for i := range candidates {
		c := &candidates[i]
		if c.HiveID != nil {
			continue
		}
		if NodeNameFromAddress(c.Address) != nodeName {
			continue
		}
		if found != nil {
			return nil
		}
		found = c
	}
	return found
}

// MatchMachineByAddress picks the machine record an enrolling agent is running
// on, from machines whose machine id is unknown.
//
// The mirror of MatchMachineByNodeName and the other half of the same merge: a
// node the sync discovered in a cluster that nobody onboarded through this system
// has a hive and a node name but no machine id, so an agent enrolling on it later
// matches nothing by machine id and would create a second record.
//
// Same refusal on ambiguity, and for the same reason. Candidates with a machine
// id are excluded by the caller's query rather than here, because a machine id
// that does not match is a DIFFERENT machine — this fallback must never take a
// record away from a host that already proved its identity.
// peerAddress is where the enrolment request DEMONSTRABLY came from, and it is
// not optional decoration. A candidate's recorded address used to imply it had
// been onboarded — the address this control plane reaches the host on, the same
// kind of string an agent reports as its own. That stopped being true when the
// capacity sync began recording the address the CLUSTER knows a node by, read
// from /cluster/status: a discovered node now carries an IP while an agent still
// calls itself by hostname, so comparing the two rejected every adoption it was
// meant to allow. Observed live — pve4 and pve5 each gained a second record the
// moment their discovered rows learned an address.
//
// So a recorded address must agree with something the agent actually is: the
// name it reports, or the address its request arrived from. Either is evidence;
// neither is a guess.
func MatchMachineByAddress(candidates []Hypervisor, address, peerAddress string) *Hypervisor {
	nodeName := NodeNameFromAddress(address)
	addr := strings.TrimSpace(address)
	peer := strings.TrimSpace(peerAddress)

	var found *Hypervisor
	for i := range candidates {
		c := &candidates[i]
		// A machine id that does not match is a DIFFERENT machine. Excluded by the
		// caller's query too, but never adopt a record that already proved itself.
		if c.MachineID != nil {
			continue
		}

		rec := strings.TrimSpace(c.Address)
		hasName := c.Name != nil && strings.TrimSpace(*c.Name) != ""

		// ADDRESS is evidence on its own, and it has to be — a host DECLARED by an
		// operator has an address and no name, because nothing names it until a
		// capacity sync sees it in a cluster. Requiring a name meant an agent
		// enrolling on a freshly onboarded host matched nothing and created a
		// second record every time. Reproduced from a clean tenant: five hosts
		// declared, onboarded and enrolled produced ten rows.
		//
		// The peer address is what makes it evidence rather than assertion: it is
		// where the enrolment request demonstrably arrived from, not something the
		// agent claims about itself.
		addrAgrees := rec != "" && (strings.EqualFold(rec, addr) || strings.EqualFold(rec, peer))
		nameAgrees := hasName && nodeName != "" && strings.EqualFold(strings.TrimSpace(*c.Name), nodeName)

		switch {
		case rec != "" && hasName:
			// Both recorded, so both must agree. A record the control plane reaches
			// at this address but which the cluster calls something else is not
			// obviously this machine, and adopting on half the evidence is how an
			// agent's identity lands on the wrong host.
			if !addrAgrees || !nameAgrees {
				continue
			}
		case rec != "":
			if !addrAgrees {
				continue
			}
		case hasName:
			// No recorded address, so the node name is the only evidence there is —
			// and adopting on it is the POINT of this fallback. A node the capacity
			// sync discovered before it could read cluster addresses has a hive and
			// a name and nothing else; refusing here would mean every agent
			// enrolling on such a node creates a second record.
			//
			// The residual risk is real and bounded: `hv5.dc2.example.com` truncates
			// to `hv5` and would match an `hv5` discovered in dc1. If both clusters
			// hold that name two candidates match and the ambiguity refusal below
			// returns nil; it only bites when exactly one does.
			if !nameAgrees {
				continue
			}
		default:
			// Neither. Nothing to match on, and matching everything is worse than
			// matching nothing.
			continue
		}

		// Ambiguity refuses outright. Two records that both look like this machine
		// mean the evidence does not identify one, and picking either writes an
		// agent's identity onto a host that might not be it.
		if found != nil {
			return nil
		}
		found = c
	}
	return found
}
