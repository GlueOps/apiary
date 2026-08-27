// Package firewall exposes rule authoring.
//
// Everything here is MEMBERSHIP-gated, not platform-gated: whose hypervisors
// these rules protect is a different question from what job title someone
// holds. Reading is open to any member; changing anything requires admin,
// because a firewall rule decides who can reach a customer's metal.
//
// Nothing here publishes. Authoring changes no host until a revision is cut,
// which is what makes it safe to edit these in the middle of the day.
package firewall

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/google/uuid"
)

type RulesetView struct {
	ID          string `json:"id" format:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RuleView struct {
	ID        string `json:"id" format:"uuid"`
	RulesetID string `json:"ruleset_id" format:"uuid"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`

	Family   string `json:"family" example:"inet"`
	Table    string `json:"table_name" example:"filter"`
	Chain    string `json:"chain_name" example:"input"`
	Position int    `json:"position" doc:"Order within this ruleset. Order is the policy: an accept above a drop is a different rule from the reverse."`

	SrcCidr string `json:"src_cidr"`
	DstCidr string `json:"dst_cidr"`

	// Set references. A rule names EITHER a literal or a set, never both.
	SrcSetID string `json:"src_set_id,omitempty" format:"uuid"`
	DstSetID string `json:"dst_set_id,omitempty" format:"uuid"`

	Protocol   string `json:"protocol" example:"tcp"`
	SrcPort    int    `json:"src_port"`
	SrcPortEnd int    `json:"src_port_end"`
	DstPort    int    `json:"dst_port"`
	DstPortEnd int    `json:"dst_port_end"`

	InIface  string `json:"in_iface"`
	OutIface string `json:"out_iface"`

	Action  string `json:"action" example:"accept"`
	Comment string `json:"comment"`
}

type BindingView struct {
	ID string `json:"id" format:"uuid"`
	// The scope is the PAIR, not either field alone. Both absent means org-wide:
	// every host in the organization, whatever fleet it is in — including one
	// that is currently in none. A swarm-scoped binding carries BOTH, so hive_id
	// alone no longer distinguishes a fleet binding from a swarm one — read
	// swarm_id first.
	HiveID string `json:"hive_id,omitempty" format:"uuid"`
	// Present only for a swarm-scoped binding. Without it on the wire the blast
	// radius of a ruleset reads as a whole cluster when it is one swarm, or the
	// reverse — and the blast-radius screen exists precisely to be believed.
	SwarmID     string `json:"swarm_id,omitempty" format:"uuid"`
	RulesetID   string `json:"ruleset_id" format:"uuid"`
	RulesetName string `json:"ruleset_name"`
	Priority    int    `json:"priority" doc:"Orders rulesets against each other. Lower renders first, and it outranks the positions inside them."`
	Enabled     bool   `json:"enabled"`
}

func toRulesetView(rs tenant.FirewallRuleset) RulesetView {
	return RulesetView{ID: rs.ID.String(), Name: rs.Name, Description: rs.Description}
}

func toRuleView(r tenant.FirewallRule) RuleView {
	return RuleView{
		ID: r.ID.String(), RulesetID: r.RulesetID.String(), Name: r.Name, Enabled: r.Enabled,
		Family: r.Family, Table: r.TableName, Chain: r.ChainName, Position: r.Position,
		SrcCidr: r.SrcCidr, DstCidr: r.DstCidr,
		SrcSetID: uuidString(r.SrcSetID), DstSetID: uuidString(r.DstSetID),
		Protocol: string(r.Protocol),
		SrcPort:  r.SrcPort, SrcPortEnd: r.SrcPortEnd,
		DstPort: r.DstPort, DstPortEnd: r.DstPortEnd,
		InIface: r.InIface, OutIface: r.OutIface,
		Action: string(r.Action), Comment: r.Comment,
	}
}

func toBindingView(b tenant.FirewallBinding) BindingView {
	v := BindingView{
		ID: b.ID.String(), RulesetID: b.RulesetID.String(),
		RulesetName: b.Ruleset.Name, Priority: b.Priority, Enabled: b.Enabled,
	}
	if b.HiveID != nil {
		v.HiveID = b.HiveID.String()
	}
	if b.SwarmID != nil {
		v.SwarmID = b.SwarmID.String()
	}
	return v
}

// RuleBody is the whole rule on every write. There is no partial update: half an
// edit — a new port against the old action — is a rule nobody intended and no
// review would have approved.
type RuleBody struct {
	Name    string `json:"name" required:"true" minLength:"1" maxLength:"100"`
	Enabled bool   `json:"enabled" default:"true"`

	Family string `json:"family,omitempty" enum:"inet,ip,ip6" default:"inet" doc:"inet covers both address families and is almost always what is wanted."`
	Table  string `json:"table_name,omitempty" maxLength:"64" default:"filter"`
	Chain  string `json:"chain_name,omitempty" maxLength:"64" default:"input"`

	Position int `json:"position,omitempty" minimum:"0" default:"100" doc:"Space these out (10, 20, 30) so a rule can be inserted between two others without renumbering."`

	SrcCidr string `json:"src_cidr,omitempty" maxLength:"64" doc:"CIDR. Must match the rule's family — nft rejects the WHOLE ruleset over one mismatch, taking the host's entire firewall with it."`
	DstCidr string `json:"dst_cidr,omitempty" maxLength:"64"`

	SrcSetID string `json:"src_set_id,omitempty" format:"uuid" doc:"Match a named set instead of a literal CIDR. Four addresses across two ports is two rules and one set; as literals it is eight, and every address added makes it two more. Giving both a set and a CIDR is refused: nft renders one and ignores the other, so the stored rule and the enforced rule would differ with nothing reporting it."`
	DstSetID string `json:"dst_set_id,omitempty" format:"uuid"`

	Protocol   string `json:"protocol,omitempty" enum:"tcp,udp,icmp,any" default:"tcp"`
	SrcPort    int    `json:"src_port,omitempty" minimum:"0" maximum:"65535"`
	SrcPortEnd int    `json:"src_port_end,omitempty" minimum:"0" maximum:"65535" doc:"Zero means just the start port."`
	DstPort    int    `json:"dst_port,omitempty" minimum:"0" maximum:"65535"`
	DstPortEnd int    `json:"dst_port_end,omitempty" minimum:"0" maximum:"65535"`

	InIface  string `json:"in_iface,omitempty" maxLength:"32" doc:"Empty matches any interface."`
	OutIface string `json:"out_iface,omitempty" maxLength:"32"`

	Action  string `json:"action,omitempty" enum:"accept,drop,reject" default:"accept"`
	Comment string `json:"comment,omitempty" maxLength:"200"`
}

func (b RuleBody) toInput() service.RuleInput {
	return service.RuleInput{
		Name: b.Name, Enabled: b.Enabled,
		Family: b.Family, Table: b.Table, Chain: b.Chain, Position: b.Position,
		SrcCidr: b.SrcCidr, DstCidr: b.DstCidr,
		SrcSetID: parseOptionalUUID(b.SrcSetID), DstSetID: parseOptionalUUID(b.DstSetID),
		Protocol: b.Protocol,
		SrcPort:  b.SrcPort, SrcPortEnd: b.SrcPortEnd,
		DstPort: b.DstPort, DstPortEnd: b.DstPortEnd,
		InIface: b.InIface, OutIface: b.OutIface,
		Action: b.Action, Comment: b.Comment,
	}
}

// mapErr turns refusals into the right status.
//
// Every validation failure below is a deliberate decision about something an
// operator typed, so none may surface as 500 — a policy refusal reported as a
// server error tells the caller the system is broken and puts their own typo on
// a 5xx dashboard.
func mapErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil

	case errors.Is(err, repo.ErrFirewallRulesetNotFound):
		return huma.Error404NotFound("no such ruleset")
	case errors.Is(err, repo.ErrFirewallRuleNotFound):
		return huma.Error404NotFound("no such rule")
	case errors.Is(err, repo.ErrFirewallBindingNotFound):
		return huma.Error404NotFound("no such binding")
	case errors.Is(err, repo.ErrFirewallHiveNotFound):
		return huma.Error404NotFound("no such fleet")
	case errors.Is(err, repo.ErrFirewallSwarmNotFound):
		return huma.Error404NotFound("no such swarm in that fleet")
	case errors.Is(err, repo.ErrFirewallSetNotFound):
		return huma.Error404NotFound("no such set")
	case errors.Is(err, repo.ErrFirewallSetElemNotFound):
		return huma.Error404NotFound("no such element in that set")

	// 409, not 422: the request is well formed and the caller is not wrong — the
	// set simply is not theirs to edit, and the next convergence would revert it.
	case errors.Is(err, repo.ErrFirewallSetManaged):
		return huma.Error409Conflict(err.Error())

	// 422, not 404: the swarm may well exist. What the caller sent is a scope the
	// schema cannot represent — a swarm without its fleet — which is a malformed
	// request rather than a missing thing.
	case errors.Is(err, repo.ErrFirewallSwarmNeedsHive):
		return huma.Error422UnprocessableEntity(err.Error())

	case errors.Is(err, repo.ErrFirewallRulesetTaken):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, repo.ErrFirewallBindingExists):
		return huma.Error409Conflict(err.Error())

	case service.IsFirewallValidation(err):
		// The message names the offending field and why, which is what the
		// operator needs, and reveals nothing about anyone else's data.
		return huma.Error422UnprocessableEntity(err.Error())

	case errors.Is(err, service.ErrTenantUnavailable):
		slog.ErrorContext(ctx, "tenant database unavailable", "error", err)
		return huma.NewError(http.StatusServiceUnavailable, "this organization's storage is not ready yet")

	default:
		slog.ErrorContext(ctx, "firewall operation failed", "error", err)
		return huma.Error500InternalServerError("firewall operation failed")
	}
}

// parseOptionalUUID turns an empty string into nil rather than an error.
//
// The field is optional and huma has already enforced format:"uuid" on anything
// non-empty, so a parse failure here cannot come from a caller — treating it as
// "not set" keeps a malformed value from silently becoming a reference to
// something else.
func parseOptionalUUID(s string) *uuid.UUID {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}

func uuidString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
