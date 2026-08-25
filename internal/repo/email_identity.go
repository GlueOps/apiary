package repo

import (
	"errors"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// ErrUnparsableEmail means no security decision can be reached for this
// address. Every caller must treat it as a DENY, never as a fallthrough.
var ErrUnparsableEmail = errors.New("email cannot be normalised for an access decision")

// EmailIdentity is the one normalisation shared by the access-rule resolvers,
// EnsureAutoMembership and the admission gate's unproven-organization ceiling,
// so the rule an operator wrote and the bucket the ceiling counts are keyed
// identically.
type EmailIdentity struct {
	Email             string
	Domain            string
	RegistrableDomain string
	ICANNSuffix       bool
}

func NormalizeEmailIdentity(email string) (EmailIdentity, error) {
	// Lowercase BEFORE publicsuffix: the library is case-sensitive and does not
	// error on mixed case, it answers wrongly. PublicSuffix("GMAIL.COM") returns
	// suffix "COM" with icann=false, so a deny rule on gmail.com would silently
	// miss GMAIL.COM.
	s := strings.ToLower(strings.TrimSpace(email))

	at := strings.LastIndex(s, "@")

	// at <= 0 rejects "@evilcorp.com", which the previous guard (at < 0) let
	// through as an empty local part attached to a real domain.
	if at <= 0 || at >= len(s)-1 {
		return EmailIdentity{}, ErrUnparsableEmail
	}

	local, domain := s[:at], s[at+1:]

	// Plus-addressing is stripped so a+spam@x.com is decided by the rule written
	// for a@x.com; otherwise an exact-email deny is evaded by one character.
	// Dots are deliberately NOT stripped: that is a Gmail-only convention and
	// would conflate genuinely distinct addresses everywhere else.
	if i := strings.IndexByte(local, '+'); i >= 0 {
		local = local[:i]
	}
	if local == "" {
		return EmailIdentity{}, ErrUnparsableEmail
	}

	if !isPlausibleMailDomain(domain) {
		return EmailIdentity{}, ErrUnparsableEmail
	}

	// EffectiveTLDPlusOne errors for localhost, IP literals and bare public
	// suffixes such as co.uk. Those cannot be keyed, so they are a denial.
	registrable, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return EmailIdentity{}, ErrUnparsableEmail
	}

	// ICANNSuffix is false when the eTLD+1 came from the PSL private section
	// (a.blogspot.com is its own eTLD+1). Such a bucket is free to multiply, so
	// the admission ceiling may want to distrust it.
	_, icann := publicsuffix.PublicSuffix(domain)

	return EmailIdentity{
		Email:             local + "@" + domain,
		Domain:            domain,
		RegistrableDomain: registrable,
		ICANNSuffix:       icann,
	}, nil
}

// PublicSuffixOf is THE public-suffix lookup for the whole codebase, exported so
// that the cohort audit's private-suffix rollup groups on exactly the suffix
// table NormalizeEmailIdentity keyed the bucket with. A second package calling
// publicsuffix directly is how the audit and the gate would come to disagree
// about which domains belong to one operator.
//
// The domain is lowercased first: the library answers wrongly rather than
// erroring on mixed case (PublicSuffix("GMAIL.COM") reports suffix "COM" with
// icann=false).
func PublicSuffixOf(domain string) (suffix string, icann bool) {
	return publicsuffix.PublicSuffix(strings.ToLower(strings.TrimSpace(domain)))
}

// isPlausibleMailDomain refuses anything that is not a plain LDH hostname —
// bracketed IPv6 literals, ports, embedded whitespace, unicode. A domain we
// cannot compare byte-for-byte against a rule value must not reach the query.
func isPlausibleMailDomain(domain string) bool {
	if domain == "" || len(domain) > 253 {
		return false
	}
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}
	if strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") {
		return false
	}
	if strings.Contains(domain, "..") {
		return false
	}
	for i := 0; i < len(domain); i++ {
		c := domain[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '.' || c == '-':
		default:
			return false
		}
	}
	for _, label := range strings.Split(domain, ".") {
		if label == "" || len(label) > 63 {
			return false
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}
	return true
}

// DomainCandidates lists the domain values a rule lookup must consider, most
// specific first. Including the eTLD+1 is what stops a deny on evilcorp.com
// from being evaded by a@mail.evilcorp.com.
func (id EmailIdentity) DomainCandidates() []string {
	if id.RegistrableDomain == "" || id.Domain == id.RegistrableDomain {
		return []string{id.Domain}
	}
	return []string{id.Domain, id.RegistrableDomain}
}
