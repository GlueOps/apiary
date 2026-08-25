package hostinstall

import (
	"bytes"
	"os"
	"testing"
)

// TestTheEmbeddedUnitMatchesPackaging.
//
// Two copies of the unit exist: packaging/systemd is what a distribution package
// and the docs reference, and embedded/ is what this binary installs. They must
// be the same file. A //go:embed of a missing path is a compile error, but a
// //go:embed of a STALE path compiles perfectly and installs last month's unit —
// which would silently undo a watchdog or hardening change made in packaging.
func TestTheEmbeddedUnitMatchesPackaging(t *testing.T) {
	t.Parallel()

	packaged, err := os.ReadFile("../../packaging/systemd/apiary-agent.service")
	if err != nil {
		t.Fatalf("read the packaged unit: %v", err)
	}
	if !bytes.Equal(bytes.TrimSpace(packaged), bytes.TrimSpace(Unit)) {
		t.Fatal("the embedded unit has drifted from packaging/systemd/apiary-agent.service; " +
			"copy the packaged one over agent/internal/hostinstall/embedded/apiary-agent.service")
	}
}

// TestTheUnitStillCarriesTheSupervisionItExistsFor.
//
// The manual path installs this to get supervision, not merely to have a unit
// file. Each of these was reasoned about in the unit's own comments, and losing
// one leaves a host that looks onboarded and is not supervised.
func TestTheUnitStillCarriesTheSupervisionItExistsFor(t *testing.T) {
	t.Parallel()

	for _, want := range []string{
		"Type=notify",     // the watchdog needs it
		"WatchdogSec=",    // catches a wedged process, which Restart cannot
		"Restart=always",  // catches one that died
		"[Install]",       // without it, enable has nothing to link
		"WantedBy=",       // and nothing to link it into
		"StateDirectory=", // or the agent's database has nowhere to live
	} {
		if !bytes.Contains(Unit, []byte(want)) {
			t.Errorf("the unit no longer contains %q, so an installed host loses that "+
				"supervision without anything failing", want)
		}
	}
}
