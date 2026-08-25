package tenant

import (
	"strings"
	"testing"
)

func intp(n int) *int { return &n }

func TestCorosyncLinkRender(t *testing.T) {
	cases := []struct {
		name string
		link CorosyncLink
		flag string
		val  string
	}{
		{"bare", CorosyncLink{Number: 0, Address: "10.1.0.9"}, "--link0", "address=10.1.0.9"},
		{"with priority", CorosyncLink{Number: 2, Address: "10.2.0.9", Priority: intp(20)},
			"--link2", "address=10.2.0.9,priority=20"},
		{"priority zero is explicit", CorosyncLink{Number: 1, Address: "10.1.0.9", Priority: intp(0)},
			"--link1", "address=10.1.0.9,priority=0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			flag, val := c.link.Flag()
			if flag != c.flag || val != c.val {
				t.Fatalf("want %s %s, got %s %s", c.flag, c.val, flag, val)
			}
		})
	}
}

func TestCorosyncLinkAlwaysUsesTheExplicitAddressForm(t *testing.T) {
	// One shape regardless of whether a priority is set. Two shapes is how a
	// renderer starts emitting "10.0.0.1,priority=20", which parses as an address
	// of "10.0.0.1" only by luck of the parser.
	l := CorosyncLink{Number: 0, Address: "10.1.0.9"}
	if !strings.HasPrefix(l.Render(), "address=") {
		t.Fatalf("want the explicit address= form even with no priority, got %q", l.Render())
	}
}

func TestValidateCorosyncLinks(t *testing.T) {
	cases := []struct {
		name    string
		links   []CorosyncLink
		wantErr string
	}{
		{"one link", []CorosyncLink{{Number: 0, Address: "10.1.0.9"}}, ""},
		{
			"redundant rings",
			[]CorosyncLink{{Number: 0, Address: "10.1.0.9"}, {Number: 1, Address: "10.2.0.9"}},
			"",
		},
		{
			// An operator may legitimately use link0 and link2. Renumbering to close
			// the gap would put traffic on a ring the rest of the cluster is not on.
			"non-contiguous numbers are fine",
			[]CorosyncLink{{Number: 0, Address: "10.1.0.9"}, {Number: 2, Address: "10.3.0.9"}},
			"",
		},
		{
			// pvecm documents the range and the count and says nothing about a
			// mandatory ring. Inventing that rule here would block a valid config.
			"link0 is not required by this validator",
			[]CorosyncLink{{Number: 1, Address: "10.2.0.9"}},
			"",
		},
		{"none", nil, "at least one corosync link"},
		{
			"too many",
			func() []CorosyncLink {
				var ls []CorosyncLink
				for i := 0; i < 9; i++ {
					ls = append(ls, CorosyncLink{Number: i % 8, Address: "10.0.0.1"})
				}
				return ls
			}(),
			"at most 8",
		},
		{"out of range high", []CorosyncLink{{Number: 8, Address: "10.1.0.9"}}, "out of range"},
		{"out of range negative", []CorosyncLink{{Number: -1, Address: "10.1.0.9"}}, "out of range"},
		{
			"duplicate ring",
			[]CorosyncLink{{Number: 0, Address: "10.1.0.9"}, {Number: 0, Address: "10.2.0.9"}},
			"more than once",
		},
		{"empty address", []CorosyncLink{{Number: 0, Address: "  "}}, "no address"},
		{
			// The address reaches a command running as root. A comma is read by
			// pvecm's own parser as another key=value pair.
			"address carrying an option separator",
			[]CorosyncLink{{Number: 0, Address: "10.1.0.9,priority=5"}},
			"option separator",
		},
		{
			"address carrying an equals",
			[]CorosyncLink{{Number: 0, Address: "address=10.1.0.9"}},
			"option separator",
		},
		{
			"negative priority",
			[]CorosyncLink{{Number: 0, Address: "10.1.0.9", Priority: intp(-1)}},
			"negative priority",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateCorosyncLinks(c.links)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected refusal: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected a refusal containing %q", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("want a message containing %q, got %q", c.wantErr, err.Error())
			}
		})
	}
}

func TestSortCorosyncLinksDoesNotMutateTheCaller(t *testing.T) {
	in := []CorosyncLink{{Number: 3, Address: "c"}, {Number: 0, Address: "a"}, {Number: 1, Address: "b"}}
	got := SortCorosyncLinks(in)

	if got[0].Number != 0 || got[1].Number != 1 || got[2].Number != 3 {
		t.Fatalf("not sorted by ring: %+v", got)
	}
	if in[0].Number != 3 {
		t.Fatal("the caller's slice must not be reordered underneath it")
	}
}
