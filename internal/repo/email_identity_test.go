package repo

import (
	"errors"
	"reflect"
	"testing"
)

func TestNormalizeEmailIdentity_Normalises(t *testing.T) {
	cases := []struct {
		name            string
		in              string
		wantEmail       string
		wantDomain      string
		wantRegistrable string
		wantICANN       bool
	}{
		{
			name:            "plus tag is stripped so an exact-email deny cannot be evaded",
			in:              "  A+spam@Mail.EvilCorp.com  ",
			wantEmail:       "a@mail.evilcorp.com",
			wantDomain:      "mail.evilcorp.com",
			wantRegistrable: "evilcorp.com",
			wantICANN:       true,
		},
		{
			name:            "uppercase is folded before the public-suffix lookup",
			in:              "USER@GMAIL.COM",
			wantEmail:       "user@gmail.com",
			wantDomain:      "gmail.com",
			wantRegistrable: "gmail.com",
			wantICANN:       true,
		},
		{
			name:            "dots in the local part are preserved, they are not a universal alias",
			in:              "first.last@example.test",
			wantEmail:       "first.last@example.test",
			wantDomain:      "example.test",
			wantRegistrable: "example.test",
			wantICANN:       false,
		},
		{
			name:            "multi-label suffix resolves to the registrable domain, not the suffix",
			in:              "a@shop.yahoo.co.uk",
			wantEmail:       "a@shop.yahoo.co.uk",
			wantDomain:      "shop.yahoo.co.uk",
			wantRegistrable: "yahoo.co.uk",
			wantICANN:       true,
		},
		{
			name:            "a PSL private suffix yields its own bucket and is flagged non-ICANN",
			in:              "a@tenant.blogspot.com",
			wantEmail:       "a@tenant.blogspot.com",
			wantDomain:      "tenant.blogspot.com",
			wantRegistrable: "tenant.blogspot.com",
			wantICANN:       false,
		},
		{
			name:            "only the last @ separates local from domain",
			in:              `"a@b"@example.test`,
			wantEmail:       `"a@b"@example.test`,
			wantDomain:      "example.test",
			wantRegistrable: "example.test",
			wantICANN:       false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeEmailIdentity(tc.in)
			if err != nil {
				t.Fatalf("NormalizeEmailIdentity(%q): %v", tc.in, err)
			}
			if got.Email != tc.wantEmail {
				t.Errorf("Email = %q, want %q", got.Email, tc.wantEmail)
			}
			if got.Domain != tc.wantDomain {
				t.Errorf("Domain = %q, want %q", got.Domain, tc.wantDomain)
			}
			if got.RegistrableDomain != tc.wantRegistrable {
				t.Errorf("RegistrableDomain = %q, want %q — this is the ceiling's bucket key; substring-after-@ would hand out a free bucket per subdomain", got.RegistrableDomain, tc.wantRegistrable)
			}
			if got.ICANNSuffix != tc.wantICANN {
				t.Errorf("ICANNSuffix = %v, want %v", got.ICANNSuffix, tc.wantICANN)
			}
		})
	}
}

func TestNormalizeEmailIdentity_RefusesUndecidableAddresses(t *testing.T) {
	// Every one of these previously normalised to something the old CheckAccess
	// happily allowed. They must now be a hard denial: no rule can be keyed to
	// them, so no security decision is reachable.
	for _, in := range []string{
		"",
		"   ",
		"nodomain",
		"user@",
		"@evilcorp.com",
		"+tag@example.test",
		"a@localhost",
		"a@co.uk",
		"a@com",
		"a@1.2.3.4",
		"a@[192.168.0.1]",
		"a@exam ple.test",
		"a@exámple.test",
		"a@example..test",
		"a@.example.test",
		"a@example.test.",
		"a@-example.test",
		"a@example.test:25",
	} {
		t.Run(in, func(t *testing.T) {
			got, err := NormalizeEmailIdentity(in)
			if !errors.Is(err, ErrUnparsableEmail) {
				t.Fatalf("NormalizeEmailIdentity(%q) = %+v, %v — want ErrUnparsableEmail", in, got, err)
			}
		})
	}
}

func TestEmailIdentity_DomainCandidates(t *testing.T) {
	// Most specific first. Dropping the registrable domain from this list is
	// what lets a deny on evilcorp.com be evaded by a@mail.evilcorp.com.
	sub, err := NormalizeEmailIdentity("a@mail.evilcorp.com")
	if err != nil {
		t.Fatalf("normalise: %v", err)
	}
	if want := []string{"mail.evilcorp.com", "evilcorp.com"}; !reflect.DeepEqual(sub.DomainCandidates(), want) {
		t.Fatalf("DomainCandidates() = %q, want %q", sub.DomainCandidates(), want)
	}

	apex, err := NormalizeEmailIdentity("a@evilcorp.com")
	if err != nil {
		t.Fatalf("normalise: %v", err)
	}
	if want := []string{"evilcorp.com"}; !reflect.DeepEqual(apex.DomainCandidates(), want) {
		t.Fatalf("DomainCandidates() = %q, want %q — an apex address must not query the same value twice", apex.DomainCandidates(), want)
	}
}
