package bee

import (
	"fmt"
	"os"
	"strings"
)

// machineIDPaths are read in order. systemd writes the first; the second is the
// D-Bus fallback still present on older and non-systemd hosts.
var machineIDPaths = []string{"/etc/machine-id", "/var/lib/dbus/machine-id"}

// MachineID returns this host's stable identifier.
//
// Not the hostname: hostnames are renamed, reused across rebuilds, and duplicated
// by templating mistakes. The control plane keys a bee on this value so that a
// rebuilt hypervisor re-enrols onto its existing row instead of leaving a ghost
// that is permanently offline — which, mid-rotation, is indistinguishable from a
// host that genuinely died.
//
// Failing loudly is deliberate. A bee that invented an identifier when it could
// not read one would enrol as a NEW host on every restart, quietly multiplying
// the fleet.
func MachineID() (string, error) {
	for _, p := range machineIDPaths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if id := strings.TrimSpace(string(b)); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("bee: could not read a machine id from any of %v; this host has no stable "+
		"identity, and enrolling without one would create a new bee on every restart", machineIDPaths)
}
