// Package canon holds the opinions as data: decision records, the rules that
// enforce them, repo profiles, scaffold templates and the principles they
// ladder up to. The files are embedded so the binary works on its own.
package canon

import "embed"

// FS is the canon directory: records/, rules/, profiles/, templates/ and
// principles.md at its root.
//
//go:embed principles.md records rules profiles all:templates
var FS embed.FS
