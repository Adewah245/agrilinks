// Package migrations contains the schema shipped with the application.
package migrations

import "embed"

// Files lets the server and tests apply the same migrations independently of
// the directory from which the binary is started.
//
//go:embed *.sql
var Files embed.FS
