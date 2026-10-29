package utils

import "strings"

// DefaultTenantDatabasePrefix names tenant databases when DATABASE_PREFIX is
// unset. It is what every tenant was called before the prefix was configurable,
// so leaving the setting out changes nothing.
const DefaultTenantDatabasePrefix = "tenant"

// TenantDatabasePrefix normalizes a configured DATABASE_PREFIX.
//
// The one place it happens, so the worker's startup validation checks exactly
// the value tenants are then named from. A trailing underscore is dropped
// because the separator is added by the caller: both "apiary" and "apiary_"
// name databases apiary_<uuid>, never apiary__<uuid>.
func TenantDatabasePrefix(prefix string) string {
	p := strings.TrimRight(strings.TrimSpace(prefix), "_")
	if p == "" {
		return DefaultTenantDatabasePrefix
	}
	return p
}
