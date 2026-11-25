package bee

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// The control plane refuses a log batch holding a line over 16 KiB, and a
// refused batch is resent forever. Nothing the agent ships may be that long.
func TestAnOverlongLogLineIsCutBeforeShipping(t *testing.T) {
	for _, m := range []string{
		strings.Repeat("x", 40<<10),
		strings.Repeat("é", 20<<10), // a cut must not split a rune
	} {
		got := shippableMessage(m)
		if len(got) > maxShippedLineBytes {
			t.Fatalf("shipped %d bytes; the bound is %d", len(got), maxShippedLineBytes)
		}
		if !utf8.ValidString(got) || !strings.HasSuffix(got, "truncated by the agent at 8 KiB") {
			t.Fatalf("not valid UTF-8 or not marked as cut: …%q", got[len(got)-60:])
		}
	}
	if s := strings.Repeat("y", maxShippedLineBytes); shippableMessage(s) != s {
		t.Fatal("a line within the bound was changed")
	}
}
