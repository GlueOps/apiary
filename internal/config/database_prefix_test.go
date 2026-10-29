package config

import (
	"strings"
	"testing"
)

func TestCheckDatabasePrefix(t *testing.T) {
	ok := []string{"", "tenant", "apiary", "apiary_", "a1_b2", strings.Repeat("a", maxDatabasePrefixLen)}
	for _, p := range ok {
		if err := checkDatabasePrefix(p); err != nil {
			t.Errorf("checkDatabasePrefix(%q) = %v, want nil", p, err)
		}
	}

	bad := []string{
		"Apiary",  // quoted at creation, so case would stick
		"1apiary", // not a valid leading identifier character
		"api-ary", // would need quoting everywhere
		"api ary",
		"apiary;drop",
		strings.Repeat("a", maxDatabasePrefixLen+1), // truncation would cut into the org id
	}
	for _, p := range bad {
		if err := checkDatabasePrefix(p); err == nil {
			t.Errorf("checkDatabasePrefix(%q) = nil, want an error", p)
		}
	}
}
