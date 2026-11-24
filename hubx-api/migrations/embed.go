// Package migrations embeds the plain-SQL migrations of hubx-api into the binary.
// Add a new file per story as NNNN_name.sql (reserve the number in memory.md); never edit an applied one.
package migrations

import "embed"

// FS holds every NNNN_name.sql file in this directory.
//
//go:embed *.sql
var FS embed.FS
