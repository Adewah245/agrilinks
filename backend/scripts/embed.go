// Package scripts contains the reviewed legacy cutover scripts.
package scripts

import "embed"

//go:embed *.sql
var Files embed.FS
