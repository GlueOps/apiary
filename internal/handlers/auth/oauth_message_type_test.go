package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// uiAuthProvider is the file that listens for the popup's message.
const uiAuthProvider = "../../../ui/src/providers/auth-provider.tsx"

// The popup and the page that opened it agree on one string, and nothing else
// connects them. A mismatch is silent in the worst way: the popup posts, no
// listener matches, the window closes, and the user lands back on the login page
// with no error in either console and nothing in the server log.
//
// It has already happened. A product rename swept the .ts/.tsx tree and updated
// the listener, while the emitter sat inside a JavaScript template literal in a
// .go file and was missed — leaving social sign-in broken with every test green.
//
// So this test reads the UI source rather than restating the constant: a test
// that compared the constant to a copy of itself would have passed throughout.
func TestOAuthCompleteMessageTypeMatchesTheUI(t *testing.T) {
	b, err := os.ReadFile(filepath.Clean(uiAuthProvider))
	if err != nil {
		// Not a skip. If the UI moves, this test must be repointed, not silently
		// stop checking — a skipped guard is how the drift got in.
		t.Fatalf("cannot read %s: %v — repoint this test at the file that now "+
			"listens for the OAuth popup message", uiAuthProvider, err)
	}

	want := `'` + OAuthCompleteMessageType + `'`
	if !strings.Contains(string(b), want) {
		t.Fatalf("the Go side posts %q, but %s does not filter on it. Social sign-in fails "+
			"silently on a mismatch: the popup posts, nothing listens, the window closes and "+
			"the user is returned to the login page with no error anywhere.",
			OAuthCompleteMessageType, uiAuthProvider)
	}

	// And the emitted page must actually carry it — the constant existing is not
	// the same as the template using it.
	page := string(writePostMessageHTMLRaw("https://apiary.test", []byte(`{"ok":true}`)))
	if !strings.Contains(page, OAuthCompleteMessageType) {
		t.Fatalf("the rendered popup page does not contain %q; the constant is declared but the "+
			"template still hard-codes something else", OAuthCompleteMessageType)
	}
}
