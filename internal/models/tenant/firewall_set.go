package tenant

import (
	"sort"
	"time"

	"github.com/google/uuid"
)

// A FirewallSet is a named list of addresses that rules reference instead of
// carrying one address each.
//
// It exists because a single src_cidr per rule multiplies: two management
// addresses across ports 22 and 8006 is four rules, ten addresses is twenty. The
// cost is not verbosity, it is DRIFT — changing the SSH port means editing every
// copy, and a half-applied edit leaves some rules saying one thing and the rest
// saying another with nothing reporting the disagreement. A set has one copy of
// the list, so it cannot be half-edited.
//
// nftables expresses this natively, so nothing is being invented here:
//
//	nft add set inet filter mgmt { type ipv4_addr \; flags interval \; }
//	nft add element inet filter mgmt { 194.36.65.187, 192.168.10.60 }
//	nft add rule inet filter input ip saddr @mgmt tcp dport 22 accept
//
// The same list is also what authorized_keys `from=` needs, so a set feeding
// both means they cannot disagree — where two hand-maintained lists eventually
// will, and the symptom is an unexplained "permission denied".
// What a set is FOR. Purpose is declared, never inferred from the ports a rule
// happens to open.
//
// Inferring was the original design and it is wrong in a way that only shows up
// once a cluster exists: corosync peers are permitted on port 22 so that pvecm
// can distribute cluster configuration, so "everything allowed on 22" includes
// every peer. Deriving the control plane's authorized_keys `from=` list from
// that would make a stolen apiary private key usable from any hypervisor in the
// fleet — one compromised node becoming a foothold against all its siblings.
//
// The port cannot distinguish them. The operator can, once.
const (
	// SetPurposeManagement: where the control plane and operators connect FROM.
	// The only purpose that feeds `from=`.
	SetPurposeManagement = "management"

	// SetPurposePeers: cluster members permitted to reach each other. Dynamic —
	// it leads membership on join and lags it on leave, because a node cannot
	// join a cluster it is not yet permitted to reach. Feeds the firewall only.
	SetPurposePeers = "peers"

	// SetPurposeGeneral: anything else an operator wants to name. Feeds the
	// firewall only; never `from=`, because a set with no stated purpose is not
	// evidence that the control plane connects from it.
	SetPurposeGeneral = "general"
)

type FirewallSet struct {
	ID   uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name string    `gorm:"not null;uniqueIndex" json:"name"`

	// Purpose decides what this set is allowed to influence. Defaults to general
	// — the harmless one — so a set created without thought never widens `from=`.
	Purpose string `gorm:"not null;default:'general';index" json:"purpose"`

	// Managed marks a set whose CONTENTS are maintained by the control plane
	// rather than by an operator, which is true of a peers set.
	//
	// Kept explicit so the UI can refuse hand-edits that the next convergence
	// would silently revert — an edit that disappears without explanation is
	// worse than one that is refused.
	Managed bool `gorm:"not null;default:false" json:"managed"`

	// Family is ip or ip6, NEVER inet.
	//
	// An nft set is typed to one address family — ipv4_addr or ipv6_addr — and
	// cannot hold both. A rule declared inet may reference either, since inet
	// only means the rule lives in a table handling both; it does not make a set
	// able to hold mixed addresses. Allowing "inet" here would produce a set
	// declaration nft rejects, and nft rejects the WHOLE ruleset over one bad
	// line, taking the host's entire firewall with it.
	Family string `gorm:"not null;default:'ip'" json:"family"`

	Description string `gorm:"not null;default:''" json:"description"`

	Elements []FirewallSetElement `gorm:"foreignKey:SetID" json:"elements,omitempty"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamptz;autoUpdateTime;not null;default:now()" json:"updated_at"`
}

func (FirewallSet) TableName() string { return "firewall_sets" }

// A FirewallSetElement is one address or prefix in a set.
//
// A row per element rather than a JSON array, so that adding a management
// address is an INSERT rather than a read-modify-write of the whole list. Two
// operators adding an address at once would otherwise have one of them silently
// overwrite the other, which is the failure a list-in-a-column always eventually
// has.
type FirewallSetElement struct {
	ID uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`

	SetID uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:ux_firewall_set_elements,priority:1" json:"set_id"`
	Set   *FirewallSet `gorm:"foreignKey:SetID;constraint:OnDelete:CASCADE" json:"-"`

	// Value is a single address or a CIDR prefix. Unique within the set: the same
	// address twice is not an error nft would report, it is just a longer set,
	// and a duplicate makes "remove this address" ambiguous.
	Value string `gorm:"not null;uniqueIndex:ux_firewall_set_elements,priority:2" json:"value"`

	// Label groups the addresses that are ONE THING — most often one machine with
	// both a public and a private address.
	//
	// A host is reachable by different addresses depending on the path taken to
	// it, so both belong in the set and neither is redundant. But a flat list
	// makes "decommission this machine" two removals that can be half-completed,
	// and the half most likely to survive is the PUBLIC one — on a range that is
	// eventually reallocated to somebody else, who inherits SSH access nobody
	// granted them. With 37 pairs, missing one is a matter of time rather than
	// carelessness.
	//
	// Empty for a standalone address, which is the common case and needs no
	// ceremony. Where it is set, removal should operate on the label rather than
	// on the row.
	Label string `gorm:"not null;default:'';index" json:"label,omitempty"`

	// Comment is what an operator will want when deciding whether an address can
	// be removed a year from now. An address with no explanation never gets
	// removed, because nobody dares.
	Comment string `gorm:"not null;default:''" json:"comment"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
}

func (FirewallSetElement) TableName() string { return "firewall_set_elements" }

// ElementsByLabel groups a set's elements by their label.
//
// Unlabelled elements each land under their own key, so a caller can treat every
// group uniformly rather than special-casing the standalone case — and so a
// standalone address is never silently swept into somebody else's group.
func (s *FirewallSet) ElementsByLabel() map[string][]FirewallSetElement {
	out := make(map[string][]FirewallSetElement, len(s.Elements))
	for _, e := range s.Elements {
		key := e.Label
		if key == "" {
			// The value itself, so two unlabelled addresses cannot collide.
			key = "\x00" + e.Value
		}
		out[key] = append(out[key], e)
	}
	return out
}

// PartialLabels reports labels whose addresses are NOT all present in the given
// list — the signature of a half-completed removal.
//
// This is the check that makes the label worth having: a machine whose private
// address was removed and whose public one was not looks like a healthy set
// until somebody reads all 74 entries. Naming the label lets the UI say "machine
// hv20 is half-removed" instead of leaving it to be noticed.
func (s *FirewallSet) PartialLabels(present map[string]bool) []string {
	var out []string
	for label, elems := range s.ElementsByLabel() {
		// No special case for standalone addresses, and none is needed: an
		// unlabelled element forms a group of exactly one, because values are
		// unique within a set. So have is 0 or 1 and want is 1, and the condition
		// below can never hold for it. A guard here looked prudent and was
		// unreachable — a surviving mutation is what showed it, since removing it
		// changed no behaviour at all.
		have, want := 0, len(elems)
		for _, e := range elems {
			if present[e.Value] {
				have++
			}
		}
		if have != 0 && have != want {
			out = append(out, label)
		}
	}
	sort.Strings(out)
	return out
}
