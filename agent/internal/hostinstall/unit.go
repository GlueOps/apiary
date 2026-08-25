package hostinstall

import _ "embed"

// Unit is the service file this binary installs.
//
// Embedded rather than read from disk, because the whole point of the manual
// path is that the only thing on the host is this binary. A unit fetched at
// install time would be one more thing to copy across, and the version skew
// between a binary and a separately-supplied unit is exactly the failure the
// automated path avoids by shipping them together.
//
// The copy under embedded/ is generated from packaging/systemd — see
// unit_test.go, which fails if the two drift.
//
//go:embed embedded/apiary-agent.service
var Unit []byte
