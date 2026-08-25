package proxmox

import "math"

// Conversions that keep a bad value from becoming an absurd one.
//
// The Proxmox API reports sizes as JSON numbers that this package parses into
// int64 and stores as uint64. A negative — from an error payload, a field that
// means "unknown", or a firmware quirk — wraps to something near 2^64, and a
// storage panel then cheerfully reports eighteen exabytes free. Clamping turns
// that into zero, which is wrong in a way an operator immediately recognises
// rather than one they act on.
//
// VMIDs get the same treatment for the same reason: Proxmox ids are positive,
// and a negative one is a parse failure, not an id.

// clampToUint64 converts a signed value, flooring negatives at zero.
func clampToUint64(v int64) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v)
}

// clampFloatToUint64 does the same for a float, and also caps at the maximum a
// uint64 can hold so that an out-of-range float does not become an
// implementation-defined value.
func clampFloatToUint64(v float64) uint64 {
	switch {
	case v < 0 || math.IsNaN(v):
		return 0
	case v >= math.MaxUint64:
		return math.MaxUint64
	default:
		return uint64(v)
	}
}

// clampToInt narrows a uint64, capping rather than wrapping. Relevant on 32-bit
// builds, where a large id or count would otherwise become a small or negative
// one silently.
func clampToInt(v uint64) int {
	if v > uint64(math.MaxInt) {
		return math.MaxInt
	}
	return int(v)
}

// clampIntToUint64 is the VMID case: a negative id is a parse failure.
func clampIntToUint64(v int) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v)
}
