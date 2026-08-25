package service

import (
	"net/netip"
	"strings"
)

// SourceTrust ranks how much an admission decision may lean on a client
// address. It is a tier rather than a boolean because the addresses behave
// genuinely differently, and the ceiling should scale with the tier rather than
// treating "came from somewhere" as one fact.
//
// Ordered least to most trusted, so a caller may compare.
type SourceTrust int

const (
	// SourceAnonymised is a Tor exit, open proxy or commercial anonymiser.
	// Addresses here are shared, disposable and deliberately unattributable, so
	// they carry no information about who is asking. The Tor Project publishes
	// its exit list; commercial feeds cover the rest.
	SourceAnonymised SourceTrust = iota

	// SourceDatacenter is cloud or hosting. Automation lives here, and — the
	// point that matters for the ceiling — addresses inside a provider's range
	// are free to an attacker. Bucketing these per address hands out unlimited
	// buckets and the ceiling stops bounding anything, so they are bucketed at
	// network granularity instead.
	SourceDatacenter

	// SourceEnterpriseEgress is a corporate VPN concentrator or a SASE/ZTNA
	// vendor's egress range — Zscaler, Netskope, Cloudflare WARP, Prisma and
	// the like. By ASN these are indistinguishable from cloud hosting, which is
	// why a naive tier model penalises them hardest: the organisations most
	// likely to mandate full-tunnel egress are the regulated ones whose staff
	// are the LEAST likely to be abusing anything. Treating a bank's entire
	// workforce as untrusted cloud traffic is precisely backwards.
	//
	// The vendors publish their egress ranges, so this is the same mechanism as
	// the cloud list with the opposite treatment.
	SourceEnterpriseEgress

	// SourceResidential is an ISP pool: consumer broadband, mobile carriers,
	// dynamic DNS. Where real people are. Note the pools are dynamic, so a
	// per-address bucket leaks in both directions over time — a legitimate
	// user's address changes, and an attacker can re-dial for a fresh one.
	SourceResidential

	// SourceCorporate is an address in a block registered to an organisation
	// rather than to an access provider — someone on their employer's network —
	// where that organisation does NOT correspond to the email's domain.
	SourceCorporate

	// SourceCorroborated is the same, except the block's registered owner DOES
	// correspond to the email's registrable domain. Two independent attributes
	// agree, and an attacker would have to control both: the mailbox at the
	// domain, and an address inside that organisation's own allocation. This is
	// the strongest signal available without verifying the mailbox itself.
	SourceCorroborated
)

// SourceKey identifies the network origin of an admission request, forming the
// second half of the unproven-organization ceiling's key.
//
// Why the ceiling needs a second half at all: keyed on the registrable domain
// ALONE it is a denial-of-service primitive rather than a control. An attacker
// registers three times as a@victim-corp.com, b@ and c@ — nothing on that path
// checks mailbox ownership — and every real employee of that company is refused
// permanently, because the ceiling has no time component and nothing clears it.
// The general rule: a quota keyed on an attribute the attacker can assert about
// SOMEONE ELSE is a weapon, not a bound. Adding the source makes the budget
// attacker-local, so they can only exhaust their own slice.
type SourceKey struct {
	// Key is the bucket. Empty means the source could not be established, in
	// which case Trusted is false and the caller must not enforce on it.
	Key string

	// Trusted is false when the client address cannot be distinguished at all —
	// see the interlock in Classify. The ceiling degrades to observe-only
	// rather than enforce a key it knows is wrong.
	Trusted bool

	// Coarse records bucketing at network rather than address granularity, so a
	// denial can be explained honestly: "one bucket for this whole network" is
	// a very different message from "one bucket for your address".
	Coarse bool

	Trust SourceTrust
}

// SourceClassifier turns a client address into a ceiling bucket and a trust
// tier. This interface is the seam; the shipped implementation deliberately
// covers only what can be decided offline from operator-supplied ranges.
//
// A production classifier wants real network intelligence. Accurate options, as
// far as I know them — this is not an exhaustive list and the commercial
// landscape moves:
//
//   - MaxMind GeoLite2-ASN: free, offline MMDB, gives ASN and AS organisation.
//     The practical basis for datacenter-vs-ISP and for the corroboration check.
//   - Team Cymru IP-to-ASN: free DNS/whois service. Authoritative, but it is a
//     network call.
//   - RDAP (the structured successor to WHOIS): gives the organisation a block
//     is allocated to. bgp.he.net presents the same underlying BGP and RIR data
//     but as a web interface rather than a documented API, so RDAP or RIPEstat
//     is the programmatic route.
//   - The Tor Project's bulk exit list: free and published, for SourceAnonymised.
//   - Cloud providers' own published ranges (AWS ip-ranges.json, Google
//     cloud.json, Azure service tags, and equivalents): free, authoritative,
//     offline — the best starting point for SourceDatacenter.
//   - Commercial feeds (IPinfo, Spur, IPQualityScore and others) additionally
//     flag VPNs, residential proxies and hosting.
//
// The strongest signal is not any single lookup but CORROBORATION: an address
// inside the allocation of the same organisation that owns the email's
// registrable domain means two independent attributes agree, and an attacker
// would have to control both. That is the property the domain-only key was
// reaching for and never had. It is why Classify takes the email domain.
//
// Do NOT put a network call on this path. An admission decision that depends on
// a third party's availability fails whenever they do, and this gate sits in
// front of registration.
type SourceClassifier interface {
	Classify(in SourceInput) SourceKey
}

// SourceInput is what the classifier reasons about.
type SourceInput struct {
	ClientIP string

	// ForwardedFor is the raw inbound header, present only so the interlock can
	// detect a proxy this server is not configured to parse.
	ForwardedFor string

	// TrustedProxies reports whether TRUSTED_PROXY_CIDRS is configured, i.e.
	// whether ClientIP was derived from a header this server actually trusts.
	TrustedProxies bool

	// RegistrableDomain is the email's eTLD+1, carried so a classifier can make
	// the corroboration check described above. The shipped implementation does
	// not use it; a classifier with ASN data should.
	RegistrableDomain string
}

// NetworkClassifier decides from operator-supplied CIDR ranges alone. It cannot
// distinguish corporate from residential — that needs allocation data — so it
// reports everything unmatched as SourceResidential, which is the conservative
// choice: residential gets per-address bucketing and the ordinary ceiling,
// neither the relaxation corroboration would earn nor the tightening a
// datacenter gets.
type NetworkClassifier struct {
	datacenter []netip.Prefix
	anonymised []netip.Prefix
	enterprise []netip.Prefix
}

// NewNetworkClassifier builds a classifier. Unparseable entries are skipped
// rather than fatal: a typo in an operator's range list must not take
// registration down.
func NewNetworkClassifier(datacenterCIDRs, anonymisedCIDRs, enterpriseCIDRs []string) *NetworkClassifier {
	return &NetworkClassifier{
		datacenter: parsePrefixes(datacenterCIDRs),
		anonymised: parsePrefixes(anonymisedCIDRs),
		enterprise: parsePrefixes(enterpriseCIDRs),
	}
}

func parsePrefixes(in []string) []netip.Prefix {
	var out []netip.Prefix
	for _, raw := range in {
		if raw = strings.TrimSpace(raw); raw == "" {
			continue
		}
		if p, err := netip.ParsePrefix(raw); err == nil {
			out = append(out, p)
		}
	}
	return out
}

func (c *NetworkClassifier) Classify(in SourceInput) SourceKey {
	// THE INTERLOCK. clientIPMiddleware falls back to RemoteAddr whenever
	// TRUSTED_PROXY_CIDRS is empty, so behind an ingress every request carries
	// the ingress pod's address. A composite key computed from that collapses
	// to domain-only and silently reinstates the third-party lockout the key
	// exists to remove — invisibly, because the code still looks like it is
	// bucketing per source.
	//
	// An inbound X-Forwarded-For with no trusted-proxy configuration is proof
	// of a proxy in front that this server is not parsing. Produce no key; the
	// caller degrades to observe-only rather than enforcing a known-wrong one.
	if !in.TrustedProxies && strings.TrimSpace(in.ForwardedFor) != "" {
		return SourceKey{Trusted: false}
	}

	addr, err := netip.ParseAddr(strings.TrimSpace(in.ClientIP))
	if err != nil {
		return SourceKey{Trusted: false}
	}
	addr = addr.Unmap()

	// Anonymised first: a Tor exit inside a hosting range is still a Tor exit,
	// and the lower tier must win.
	for _, p := range c.anonymised {
		if p.Contains(addr) {
			return SourceKey{Key: "anon:" + p.String(), Trusted: true, Coarse: true, Trust: SourceAnonymised}
		}
	}
	// Before datacenter: a SASE egress range frequently sits inside a cloud
	// provider's space, and here the HIGHER tier must win — the opposite of the
	// anonymised case above, because the specific list is the more informative
	// one in both directions.
	for _, p := range c.enterprise {
		if p.Contains(addr) {
			return SourceKey{Key: "corp:" + p.String(), Trusted: true, Coarse: true, Trust: SourceEnterpriseEgress}
		}
	}
	for _, p := range c.datacenter {
		if p.Contains(addr) {
			return SourceKey{Key: "net:" + p.String(), Trusted: true, Coarse: true, Trust: SourceDatacenter}
		}
	}
	return SourceKey{Key: "ip:" + addr.String(), Trusted: true, Trust: SourceResidential}
}

// CeilingFor scales the unproven-organization allowance.
//
// Two INDEPENDENT axes, which is easy to conflate and wrong to:
//
//   - Trust: how much the address tells you about who is asking.
//   - Coarseness: how many distinct actors share the bucket.
//
// A SASE egress is coarse AND high-trust — thousands of one company's staff
// behind a handful of addresses. A cloud host is coarse and low-trust. A
// residential line is fine-grained and middling. Scaling on trust alone gives a
// bank's entire workforce the same allowance as one home broadband connection,
// which is the failure this function exists to avoid.
//
// The multipliers are the knob most likely to need tuning against real traffic,
// which is why the ceiling ships in shadow mode.
func CeilingFor(base int, trust SourceTrust, coarse bool) int {
	if base <= 0 {
		return 0
	}

	switch {
	case trust <= SourceAnonymised:
		// Not "a small allowance" but none. Nothing about a Tor exit or open
		// proxy distinguishes one requester from the next, so any allowance
		// there is really a global one handed to whoever finds it first.
		return 0

	case trust == SourceDatacenter:
		// Coarse and untrusted. One bucket for the provider, base allowance —
		// the point is that an attacker with unlimited addresses inside a cloud
		// still gets one bucket, not one per address.
		return base

	case trust == SourceEnterpriseEgress:
		// Coarse and trusted: many real employees share very few addresses, so
		// the allowance has to cover a workforce rather than a person. Reaching
		// this bucket at all means being inside a corporate tunnel, which is a
		// far higher bar than renting a VPS.
		return base * 8

	case trust >= SourceCorroborated:
		return base * 4

	case trust == SourceCorporate:
		return base * 2

	default:
		// Residential. Fine-grained, so the bucket covers roughly one household
		// and the base allowance is already per-actor. A coarse bucket at this
		// tier would mean a shared NAT, which deserves more room.
		if coarse {
			return base * 2
		}
		return base
	}
}
