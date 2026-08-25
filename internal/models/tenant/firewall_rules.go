package tenant

import (
	"cmp"
	"time"

	"github.com/google/uuid"
)

type Protocol string
type Action string

const (
	TCP  Protocol = "tcp"
	UDP  Protocol = "udp"
	ICMP Protocol = "icmp"
	// ProtocolAny matches regardless of protocol. Without it the common rule —
	// "allow everything from the management range" — cannot be expressed at all,
	// and people write three rules that drift apart.
	ProtocolAny Protocol = "any"

	// Lowercase, because nftables verdicts are lowercase and these are rendered
	// into nft syntax. Uppercase ACCEPT/DROP is iptables spelling; carrying it
	// here would mean a translation step that exists only to undo a naming
	// choice, and one nobody remembers when adding a fourth verdict.
	Accept Action = "accept"
	Drop   Action = "drop"
	Reject Action = "reject"
)

// A FirewallRuleset is a NAMED, ordered group of rules — "management access",
// "ceph replication" — that fleets are bound to. It is the unit an operator
// thinks in and the unit that gets reused.
//
// The alternative was hanging hive_id directly off each rule. That works right
// up to the second fleet, and then every rule an organization wants everywhere
// exists once per fleet. Five fleets means five copies of "allow 22 from the
// jumpbox", an edit that has to be repeated five times, and a fleet that quietly
// misses one — which is the exact class of mistake a firewall exists to prevent.
// Rules are authored once here and bound where they apply.
type FirewallRuleset struct {
	ID   uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name string    `gorm:"not null;uniqueIndex" json:"name"`

	Description string `gorm:"not null;default:''" json:"description"`

	CreatedAt time.Time `json:"created_at" gorm:"type:timestamptz;column:created_at;not null;default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()"`
}

// A FirewallBinding attaches one ruleset to one SCOPE: the whole organization,
// one hive, or one swarm within a hive. Which of the three it is comes from the
// pair (HiveID, SwarmID) — see the table on HiveID below.
//
// Many-to-many, because the operational shape is "a base ruleset everywhere plus
// one for this datacenter", and one-ruleset-per-hive collapses straight back
// into the duplication above.
//
// A change to a ruleset therefore re-renders EVERY hive bound to it. That is the
// point of sharing and also its hazard: the blast radius of an edit is not
// visible from the rule being edited. Anything offering an edit should say how
// many fleets it lands on before it is saved.
type FirewallBinding struct {
	ID uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`

	// HiveID is nullable, and null means "every bee in this organization,
	// whatever fleet it is in — including none".
	//
	// That state has to be expressible. A host between fleets belongs to no hive,
	// and it is precisely then that it still needs management access and a route
	// home. Hive-scoped bindings alone would leave a migrating host with an empty
	// ruleset at its most exposed moment.
	//
	// It is also how a base ruleset stops being copied per fleet, which was the
	// duplication this whole structure exists to avoid.
	//
	// The foreign key is real, now that hives live in this database. While the
	// hive lived in control this was an unenforceable cross-database reference,
	// and binding a ruleset to another tenant's fleet was a mistake only the
	// service layer could catch.
	// THREE partial unique indexes, one per representable scope, and their
	// predicates must PARTITION the table.
	//
	// A plain unique index on (hive_id, ruleset_id) does not constrain the
	// org-wide rows at all: in Postgres NULL is not equal to NULL, so every
	// org-wide binding is unique against every other one, and the same ruleset
	// can be bound org-wide any number of times. Each copy renders, so the
	// ruleset appears repeatedly in the output — verified against Postgres 18,
	// which accepted two identical org-wide rows under the single-index version.
	//
	// Partial rather than NULLS NOT DISTINCT: the predicate says out loud which
	// case each index covers, and it does not depend on a Postgres 15+ feature
	// that GORM would have to learn to emit.
	//
	//	hive_id  swarm_id   scope      index
	//	NULL     NULL       org-wide   idx_firewall_bindings_org_ruleset
	//	SET      NULL       one hive   idx_firewall_bindings_hive_ruleset
	//	SET      SET        one swarm  idx_firewall_bindings_swarm_ruleset
	//	NULL     SET        —          forbidden by the CHECK
	//
	// EXHAUSTIVENESS IS THE PROPERTY THAT MATTERS. A row matching no partial
	// index is unconstrained and can be inserted any number of times, which is
	// the same duplicate-render bug in a new corner.
	//
	// The hive index's predicate GAINED `AND swarm_id IS NULL`, and that is a
	// correctness fix rather than tidying. Swarm-scoped rows also have hive_id
	// set, so without it they fall into an index keyed (hive_id, ruleset_id) —
	// and ruleset "base-extra" could then be bound to fleet-prod OR fleet-nonprod
	// within one cluster but never both. A correct operation refused, with an
	// error naming a different swarm.
	//
	// The org index's `AND swarm_id IS NULL` is strictly redundant given the
	// CHECK. It is kept anyway so the three predicates partition the table on
	// their own face; if the CHECK is ever dropped, a (NULL, swarm) row would
	// otherwise be indexed as org-wide.
	HiveID *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_firewall_bindings_hive_ruleset,priority:1,where:hive_id IS NOT NULL AND swarm_id IS NULL;index;check:chk_firewall_bindings_swarm_needs_hive,swarm_id IS NULL OR hive_id IS NOT NULL" json:"hive_id,omitempty"`
	Hive   *Hive      `gorm:"foreignKey:HiveID;constraint:OnDelete:CASCADE" json:"-"`

	// SwarmID narrows a binding to one group within a hive. Null means the
	// binding is not swarm-scoped — read the table above, not this field alone.
	//
	// It carries hive_id ALONGSIDE it rather than standing on its own, matching
	// Hypervisor.SwarmID and Pool.SwarmID, and the reason is the composite
	// foreign key: (hive_id, swarm_id) → swarms(hive_id, id) makes "the binding's
	// swarm is in the binding's hive" a database rule instead of a service-layer
	// hope. The key itself is (swarm_id, ruleset_id) because swarm_id is already
	// a global primary key and hive_id would add nothing to it.
	//
	// ON DELETE SET NULL HERE WOULD BE A SECURITY BUG, not a preference. Postgres
	// nulls EVERY column of a composite foreign key on SET NULL. On Hypervisor
	// that merely produced a confusing error; here both columns are nullable so
	// it would SUCCEED — silently converting a swarm-scoped binding into an
	// ORG-WIDE one. Deleting a swarm would widen its ruleset to every host in the
	// organization with nothing reporting it. RESTRICT, matching Hypervisor and
	// Pool.
	SwarmID *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_firewall_bindings_swarm_ruleset,priority:1,where:swarm_id IS NOT NULL;index" json:"swarm_id,omitempty"`
	Swarm   *Swarm     `gorm:"foreignKey:HiveID,SwarmID;references:HiveID,ID;constraint:OnDelete:RESTRICT" json:"-"`

	RulesetID uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:idx_firewall_bindings_hive_ruleset,priority:2;uniqueIndex:idx_firewall_bindings_swarm_ruleset,priority:2;uniqueIndex:idx_firewall_bindings_org_ruleset,where:hive_id IS NULL AND swarm_id IS NULL" json:"ruleset_id"`
	Ruleset   FirewallRuleset `gorm:"foreignKey:RulesetID;constraint:OnDelete:CASCADE" json:"-"`

	// Priority orders the rulesets against each other; Rule.Position orders rules
	// inside one. Both are needed: without this, two bound rulesets render in an
	// arbitrary order, and for a firewall an accept above a drop is a different
	// policy from the reverse.
	//
	// Lower renders first. Space them out (10, 20, 30) so a ruleset can be slid
	// between two others without renumbering.
	Priority int `gorm:"not null;default:100" json:"priority"`

	Enabled bool `gorm:"not null;default:true" json:"enabled"`

	CreatedAt time.Time `json:"created_at" gorm:"type:timestamptz;column:created_at;not null;default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()"`
}

func (FirewallBinding) TableName() string { return "firewall_bindings" }

// Scope ranks order the three scopes against each other when their bindings
// carry the SAME priority. Lower renders first.
//
// SWARM FIRST IS THE LOAD-BEARING DIRECTION, and it is easy to get backwards.
// Rules render into an nft chain and evaluation stops at the first terminal
// verdict, so earlier-rendered WINS — the opposite of the "last match wins"
// intuition people bring from other rule engines. Rank org-wide first and an
// org-wide catch-all drop shadows the swarm ruleset that was meant to override
// it, which is a silent lockout rather than a visible misconfiguration.
const (
	ScopeRankSwarm = 0
	ScopeRankHive  = 1
	ScopeRankOrg   = 2
)

// ScopeRank reports how specific this binding's scope is.
//
// It reads SwarmID first because a swarm-scoped binding carries hive_id TOO —
// the composite foreign key needs it — so `HiveID != nil` no longer means
// "hive-scoped" on its own. Testing hive first would rank every swarm binding as
// hive-scoped and quietly undo the ordering above.
func (b FirewallBinding) ScopeRank() int {
	switch {
	case b.SwarmID != nil:
		return ScopeRankSwarm
	case b.HiveID != nil:
		return ScopeRankHive
	default:
		return ScopeRankOrg
	}
}

// CompareBindingsForRender is THE order bound rulesets render in, written once
// so the repository's ORDER BY and OrderedRules cannot drift apart. Two
// orderings meant to agree and expressed in two places do not stay in agreement,
// and here the consequence is an operator reading a different policy from the
// one the host enforces.
//
// PRIORITY OUTRANKS SPECIFICITY, deliberately, with specificity only breaking
// ties. Priority is the operator's explicit knob, spaced 10/20/30; letting an
// implicit scope tier beat it would remove the ability to put an org-wide "drop
// bogons" ruleset above everything, which is a legitimate thing to want.
//
// Specificity as the TIE-BREAK is not cosmetic either: every row defaults to
// priority 100, so ties are the common case rather than the edge case. The
// previous tie-break was age alone, which let an org-wide binding created a year
// ago outrank a swarm binding made this morning purely by accident of when it
// was written. Rows whose priorities already differ are unaffected.
//
// Age then id remain below that. Age is what a person would guess; the id is
// there because rows created in one transaction share a timestamp, and a
// remaining tie would change the rendered digest between renders of unchanged
// data — every bee in the hive re-applying a ruleset that did not change.
func CompareBindingsForRender(a, b FirewallBinding) int {
	if c := cmp.Compare(a.Priority, b.Priority); c != 0 {
		return c
	}
	if c := cmp.Compare(a.ScopeRank(), b.ScopeRank()); c != 0 {
		return c
	}
	if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
		return c
	}
	return cmp.Compare(a.ID.String(), b.ID.String())
}

// A BindingScope is where a host sits: which hive, and which swarm inside it.
// Both nil is a real answer — a machine in no cluster — and it resolves to the
// org-wide bindings rather than to nothing.
//
// THE PAIR TRAVELS TOGETHER AND IS NEVER ASSEMBLED FROM TWO READS. Taking the
// hive from one row and the swarm from another reintroduces exactly the
// two-columns-that-disagree problem the hypervisor-as-machine shape exists to
// remove, and here it resolves a swarm binding from a cluster the host has
// already left. The composite foreign key on Hypervisor prevents that pairing at
// write time; a hand-assembled scope would reintroduce it at read time.
//
// A stale observation is NOT grounds to refuse. The departure gate refuses on
// stale fleet state, and is right to, but refusing here yields no rules — and
// Bee.AppliedFirewallRevision is explicit that an absent instruction means HOLD,
// never flush. Degrade to the less specific scope instead: a host whose swarm we
// cannot currently establish still needs its hive's and the organization's
// management access.
type BindingScope struct {
	HiveID  *uuid.UUID
	SwarmID *uuid.UUID
}

// A FirewallRule is one line of a ruleset.
//
// These are DESIRED STATE, not commands. A hive's whole ruleset is rendered and
// applied atomically rather than executing individual changes, so a host that
// missed an update converges on its next poll instead of replaying history it
// never saw.
type FirewallRule struct {
	ID uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`

	// A rule belongs to exactly one ruleset. Sharing happens by binding the
	// ruleset to several hives, not by pointing a rule at several rulesets — a
	// rule reachable from two groups has no single position, and position is
	// meaning here.
	RulesetID uuid.UUID       `gorm:"type:uuid;not null;index:idx_firewall_rules_order,priority:1" json:"ruleset_id"`
	Ruleset   FirewallRuleset `gorm:"foreignKey:RulesetID;constraint:OnDelete:CASCADE" json:"-"`

	Name    string `gorm:"not null" json:"name"`
	Enabled bool   `gorm:"not null;default:true" json:"enabled"`

	Family    string `gorm:"not null;default:'inet';index:idx_firewall_rules_order,priority:2" json:"family"`
	TableName string `gorm:"not null;default:'filter';index:idx_firewall_rules_order,priority:3" json:"table_name"`
	ChainName string `gorm:"not null;default:'input';index:idx_firewall_rules_order,priority:4" json:"chain_name"`

	// Position orders rules within this ruleset's chain.
	//
	// Not unique, so a rule can be inserted between two others without
	// renumbering — but rendering therefore orders by (position, created_at, id).
	// Ties resolve by age, which is both deterministic and the order a person
	// would guess. Leaving them unordered would let the same rows render as two
	// different rulesets.
	//
	// Space positions out (10, 20, 30) so there is room to insert.
	Position int `gorm:"not null;default:100;index:idx_firewall_rules_order,priority:5" json:"position"`

	// A rule names its source and destination EITHER as a literal prefix OR as a
	// set, never both.
	//
	// Both are kept because they answer different needs. A literal is right for
	// the common cases — 0.0.0.0/0, or one host — and forcing those through a
	// named set would be ceremony. A set is right the moment the same list
	// appears in more than one rule, which is exactly when hand-copied literals
	// start to drift.
	//
	// Validate() refuses a rule carrying both: nft would render one of them and
	// silently ignore the other, so the stored rule and the enforced rule would
	// differ with nothing reporting it.
	SrcCidr  string       `gorm:"not null;default:'0.0.0.0/0'" json:"src_cidr"`
	SrcSetID *uuid.UUID   `gorm:"type:uuid;index" json:"src_set_id,omitempty"`
	SrcSet   *FirewallSet `gorm:"foreignKey:SrcSetID;constraint:OnDelete:RESTRICT" json:"-"`

	DstCidr  string       `gorm:"not null;default:'0.0.0.0/0'" json:"dst_cidr"`
	DstSetID *uuid.UUID   `gorm:"type:uuid;index" json:"dst_set_id,omitempty"`
	DstSet   *FirewallSet `gorm:"foreignKey:DstSetID;constraint:OnDelete:RESTRICT" json:"-"`

	Protocol Protocol `gorm:"not null;default:'tcp'" json:"protocol"`

	// Ports are ranges. A single int cannot express a NodePort range or a
	// passive-FTP range, and the workaround — one rule per port — is how a chain
	// grows to hundreds of lines that nobody can read.
	//
	// Zero means "any". SrcPortEnd/DstPortEnd zero means "just the start port".
	SrcPort    int `gorm:"not null;default:0" json:"src_port"`
	SrcPortEnd int `gorm:"not null;default:0" json:"src_port_end"`
	DstPort    int `gorm:"not null;default:0" json:"dst_port"`
	DstPortEnd int `gorm:"not null;default:0" json:"dst_port_end"`

	// Interface matching. A hypervisor has several NICs and "trusted management
	// range" usually means "on the management interface", which a CIDR alone
	// cannot say. Empty matches any interface.
	InIface  string `gorm:"not null;default:''" json:"in_iface"`
	OutIface string `gorm:"not null;default:''" json:"out_iface"`

	Action  Action `gorm:"not null;default:'accept'" json:"action"`
	Comment string `gorm:"not null;default:''" json:"comment"`

	CreatedAt time.Time `json:"created_at" gorm:"type:timestamptz;column:created_at;not null;default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()"`
}

// PortRange returns the effective range for a start/end pair, collapsing the
// "end unset" case so callers do not each reinvent it.
func PortRange(start, end int) (int, int) {
	if start == 0 {
		return 0, 0
	}
	if end <= start {
		return start, start
	}
	return start, end
}

// A FirewallSnapshot is one hive's rendered ruleset at one revision. Immutable:
// rows are inserted, never updated.
//
// It exists because rules plus a revision number cannot express a backout. Rules
// are edited in place, so once revision 2 is written, revision 1's content is
// gone — and reverting is the plan for exactly the situation where the current
// ruleset is what broke. A backout that depends on remembering what the rules
// used to say is not a backout.
//
// It is also where composition stops mattering. Rulesets, bindings and
// priorities are authoring concerns; what a bee applies is a flat ordered list
// it did not have to assemble. However elaborate the composition gets, the
// artifact stays one row that can be read.
//
// The stored form is the RENDERED ruleset rather than the rows it came from, so
// that a change to the renderer cannot silently change what an already-applied
// revision means. Otherwise upgrading agents mid-fleet leaves half the hosts
// enforcing a different reading of the same revision number, with nothing
// anywhere reporting a difference.
type FirewallSnapshot struct {
	ID uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`

	// Revision is per hive and never reused within one. See Hive.FirewallRevision
	// in the control models for why it only ever increases.
	HiveID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_firewall_snapshots_hive_rev,priority:1" json:"hive_id"`
	Hive     *Hive     `gorm:"foreignKey:HiveID;constraint:OnDelete:CASCADE" json:"-"`
	Revision int       `gorm:"not null;uniqueIndex:idx_firewall_snapshots_hive_rev,priority:2" json:"revision"`

	// Content is the rendered ruleset, exactly as the bee will apply it.
	Content string `gorm:"not null;type:text" json:"content"`

	// Digest is the SHA-256 of Content, as "sha256:<hex>". A bee reports the
	// digest of what it actually loaded, so "applied revision 7" can be checked
	// rather than trusted — a host that applied a truncated or hand-edited
	// ruleset reports the same revision as a healthy one and is otherwise
	// indistinguishable.
	Digest string `gorm:"not null;index" json:"digest"`

	// RevertOf names the revision this one reproduces, when it was created as a
	// backout. A revert is a NEW revision carrying an OLD body, so without this
	// the history reads as an unexplained change back.
	RevertOf *int `json:"revert_of,omitempty"`

	CreatedByID *uuid.UUID `gorm:"type:uuid" json:"created_by_id,omitempty"`
	Reason      string     `gorm:"not null;default:''" json:"reason"`
	CreatedAt   time.Time  `json:"created_at" gorm:"type:timestamptz;column:created_at;not null;default:now()"`
}

func (FirewallSnapshot) TableName() string { return "firewall_snapshots" }
