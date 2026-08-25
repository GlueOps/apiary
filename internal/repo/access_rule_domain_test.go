package repo

import (
	"strings"
	"testing"
)

// A domain rule is compared against DomainCandidates() — a real hostname and its
// registrable domain. A value that normalisation rejects stores cleanly, lists
// as active policy, and can never fire: the operator believes a domain is
// blocked while every address on it signs in. That is worse than no rule.
func TestDomainRuleValuesMustBeAbleToMatch(t *testing.T) {
	usable := []string{"gmail.com", "mail.evilcorp.com", "a.b.c.co.uk", "xn--80ak6aa92e.com"}
	dead := []string{"com", "notadomain", "..foo", "-leading.com", "trailing-.com"}

	for _, d := range usable {
		t.Run("usable/"+d, func(t *testing.T) {
			id, err := NormalizeEmailIdentity("probe@" + d)
			if err != nil || id.Domain != d {
				t.Fatalf("a real domain was rejected: err=%v domain=%q", err, id.Domain)
			}
			// The check that matters: it must appear in what the evaluator compares against.
			found := false
			for _, c := range id.DomainCandidates() {
				if c == d {
					found = true
				}
			}
			if !found {
				t.Fatalf("%q is not among its own candidates %v", d, id.DomainCandidates())
			}
		})
	}

	for _, d := range dead {
		t.Run("dead/"+d, func(t *testing.T) {
			id, err := NormalizeEmailIdentity("probe@" + d)
			if err == nil && id.Domain == d {
				t.Fatalf("%q normalised cleanly, so the guard would admit a rule that can never fire", d)
			}
		})
	}
}

// The refusal has to say WHY, or an operator retries with another spelling of
// the same unmatched value.
func TestDeadDomainRefusalExplainsItself(t *testing.T) {
	if !strings.Contains(ErrInvalidRuleValue.Error(), "value") {
		t.Fatalf("the sentinel does not name what was wrong: %v", ErrInvalidRuleValue)
	}
}
