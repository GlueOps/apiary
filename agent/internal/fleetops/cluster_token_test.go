package fleetops

import (
	"strings"
	"testing"
)

// The whole-token paste. It is the natural mistake — PVE_TOKEN is one string in
// every example and every .env — and the resulting header is
// "PVEAPIToken=<id>=<id>=<uuid>", which Proxmox rejects as unauthorized without
// saying why. The credential looks correct in every place it is displayed.
func TestNewLocalClientRefusesAWholeTokenAsTheSecret(t *testing.T) {
	_, err := NewLocalClient("https://pve.example:8006", "root@pam!token",
		"root@pam!token=7d6f1a2e-0000-4000-8000-1234567890ab")
	if err == nil {
		t.Fatal("a whole token string was accepted as the secret; the doubled header that " +
			"produces fails at Proxmox with a bare 401")
	}
	if !strings.Contains(err.Error(), "token id") {
		t.Fatalf("the refusal does not explain the split, so it is no better than the 401: %v", err)
	}
}

// A real Proxmox token secret is a UUID, so the guard must pass one untouched.
// A check that rejects valid credentials is worse than none.
func TestNewLocalClientAcceptsASplitToken(t *testing.T) {
	if _, err := NewLocalClient("https://pve.example:8006", "root@pam!token",
		"7d6f1a2e-0000-4000-8000-1234567890ab"); err != nil {
		t.Fatalf("a correctly split token was refused: %v", err)
	}
}

func TestNewLocalClientRefusesMissingHalves(t *testing.T) {
	cases := []struct{ name, id, secret string }{
		{"no token id", "", "7d6f1a2e-0000-4000-8000-1234567890ab"},
		{"no secret", "root@pam!token", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewLocalClient("https://pve.example:8006", c.id, c.secret); err == nil {
				t.Fatal("accepted a half-empty credential")
			}
		})
	}
}
