package tenant

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// MaxCorosyncLinks is what Proxmox supports: link0 through link7.
//
// From `pvecm help add --verbose` on a live node, not from memory:
//
//	--link[n]  [address=]<IP> [,priority=<integer>]
//	     Address and priority information of a single corosync link.
//	     (up to 8 links supported; link0..link7)
const MaxCorosyncLinks = 8

// A CorosyncLink is one corosync ring for a node.
//
// Plural by design. A single link makes the cluster network a single point of
// failure for quorum, which is the failure mode that fences healthy nodes — so
// redundant rings are the normal production shape, not an advanced option, and
// modelling one address would have forced every multi-ring cluster to be
// configured outside this system.
type CorosyncLink struct {
	// Number is the ring number, 0-7. It is not an index into a list: an operator
	// may legitimately configure link0 and link2, and renumbering their rings to
	// close the gap would put traffic on a ring the rest of the cluster is not
	// using.
	Number int `json:"number" minimum:"0" maximum:"7"`

	// Address is this node's address on that ring.
	Address string `json:"address"`

	// Priority orders knet's link preference; higher wins. Optional, and absent
	// is not zero — corosync's own default applies, and writing an explicit 0
	// would silently deprioritise a link the operator said nothing about.
	Priority *int `json:"priority,omitempty" minimum:"0"`
}

// Render produces the value half of a `--link<n>` argument.
//
// Always the explicit `address=` form. The bare form is accepted by pvecm, but
// the moment a priority is present the value must be a comma-separated key list,
// and having two shapes depending on an optional field is how a renderer starts
// producing `10.0.0.1,priority=20` — which parses as an address of "10.0.0.1"
// only by luck of the parser.
func (l CorosyncLink) Render() string {
	v := "address=" + l.Address
	if l.Priority != nil {
		v += ",priority=" + strconv.Itoa(*l.Priority)
	}
	return v
}

// Flag produces the whole `--linkN value` pair.
func (l CorosyncLink) Flag() (string, string) {
	return "--link" + strconv.Itoa(l.Number), l.Render()
}

// ValidateCorosyncLinks refuses a set that pvecm would reject or that would
// silently do something other than what was asked.
//
// Deliberately does NOT require link0 to be present. `pvecm help add` states the
// range and the count and says nothing about a mandatory ring, so requiring one
// here would be this system inventing a constraint the provider does not
// document — and blocking a valid configuration is as real a failure as allowing
// an invalid one. If pvecm does require it, it refuses with its own message,
// which is the correct place for that rule to live.
func ValidateCorosyncLinks(links []CorosyncLink) error {
	if len(links) == 0 {
		// Not defaulted to the host's management address. A cluster's corosync
		// network routinely differs from the one the control plane reaches the
		// host over, and guessing puts cluster traffic on whichever interface was
		// convenient — which works until that network is busy and then presents as
		// unexplained fencing.
		return fmt.Errorf("at least one corosync link is required; it is not assumed to be the " +
			"host's management address, because cluster traffic routinely belongs on a different " +
			"network")
	}
	if len(links) > MaxCorosyncLinks {
		return fmt.Errorf("at most %d corosync links are supported (link0-link%d), got %d",
			MaxCorosyncLinks, MaxCorosyncLinks-1, len(links))
	}

	seen := make(map[int]bool, len(links))
	for _, l := range links {
		if l.Number < 0 || l.Number >= MaxCorosyncLinks {
			return fmt.Errorf("corosync link number %d is out of range; Proxmox supports link0-link%d",
				l.Number, MaxCorosyncLinks-1)
		}
		if seen[l.Number] {
			// Two values for one ring is not a merge: pvecm takes one --linkN, so
			// the second would silently replace the first and the node would join
			// on an address nobody chose.
			return fmt.Errorf("corosync link%d is given more than once", l.Number)
		}
		seen[l.Number] = true

		if strings.TrimSpace(l.Address) == "" {
			return fmt.Errorf("corosync link%d has no address", l.Number)
		}
		// The address is interpolated into a command run as root on a hypervisor.
		// A comma or an equals sign would be read as another key=value pair by
		// pvecm's own parser, turning an address into an option.
		if strings.ContainsAny(l.Address, ",= \t") {
			return fmt.Errorf("corosync link%d address %q contains a character that pvecm reads as "+
				"an option separator; give the address alone and set priority separately",
				l.Number, l.Address)
		}
		if l.Priority != nil && *l.Priority < 0 {
			return fmt.Errorf("corosync link%d has a negative priority", l.Number)
		}
	}
	return nil
}

// SortCorosyncLinks returns the links in ring order.
//
// Ordering is for the operator reading a command or a log line, not for
// correctness — pvecm takes the flags in any order. Sorted so that a job's
// recorded arguments are stable between runs and a diff of two attempts shows
// what actually changed.
func SortCorosyncLinks(links []CorosyncLink) []CorosyncLink {
	out := make([]CorosyncLink, len(links))
	copy(out, links)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}

// clusterNameRe is Proxmox's own node-name format, which is what it validates a
// cluster name against.
//
// Read off a running node rather than inferred: PVE::API2::ClusterConfig declares
// clustername with `format => 'pve-node', maxLength => 15`, and
// PVE::JSONSchema::pve_verify_node_name is this expression. Alphanumeric at both
// ends, hyphens permitted only in between.
var clusterNameRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// ClusterNameMaxLen is Proxmox's maxLength for a cluster name.
//
// Fifteen, and it is a hard limit rather than a guideline: corosync writes the
// name into its ring configuration, and `pvecm create` refuses a longer one
// outright. Worth checking here because the alternative is discovering it on
// real metal, after the SSH session has already opened.
const ClusterNameMaxLen = 15

// ValidateClusterName refuses a name `pvecm create` would.
//
// This is the ONE place a hive's name has to be more than a label. Everywhere
// else it is free text — the arrival path deliberately compares a joiner against
// the target's own reported cluster rather than against the hive name, because
// nothing enforces that the two match. A bootstrap is where the hive name
// BECOMES the cluster name, so it has to be legal before the command runs.
func ValidateClusterName(name string) error {
	if name == "" {
		return fmt.Errorf("a cluster name is required")
	}
	if len(name) > ClusterNameMaxLen {
		return fmt.Errorf("cluster name %q is %d characters; Proxmox allows at most %d, and "+
			"`pvecm create` refuses a longer one", name, len(name), ClusterNameMaxLen)
	}
	if !clusterNameRe.MatchString(name) {
		return fmt.Errorf("cluster name %q is not a valid Proxmox node name: it must start and "+
			"end with a letter or digit and may contain hyphens in between", name)
	}
	return nil
}
