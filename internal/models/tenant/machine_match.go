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
func MatchMachineByAddress(candidates []Hypervisor, address string) *Hypervisor {
	nodeName := NodeNameFromAddress(address)
	if nodeName == "" {
		return nil
	}
	var found *Hypervisor
	for i := range candidates {
		c := &candidates[i]
		if c.MachineID != nil || c.Name == nil || *c.Name != nodeName {
			continue
		}
		if found != nil {
			return nil
		}
		found = c
	}
	return found
}
