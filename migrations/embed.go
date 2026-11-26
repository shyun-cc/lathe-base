// Package migrations contains the database schema bundled into the API binary.
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
