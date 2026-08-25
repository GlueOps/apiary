package migrations

import "embed"

// The //go:embed lines below are DIRECTIVES, not documentation. Without them
// these are zero-value embed.FS values, goose.SetBaseFS receives an empty
// filesystem, and `migrate up` reports success having applied nothing — the
// worst possible failure mode for a schema change. They were previously lost
// because tools/main.go strips comments from every .go file in the repo; that
// tool now preserves //go: directives specifically so this cannot recur.

//go:embed control/*.sql
var ControlFS embed.FS

// all: is required because internal/migrations/tenant currently holds only
// .gitkeep, and embed skips dot-prefixed files without it — a bare
// //go:embed tenant would fail to compile on an empty directory.
//
//go:embed all:tenant
var TenantFS embed.FS
