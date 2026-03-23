// Package migrations embeds all SQL migration files.
package migrations

import "embed"

//go:embed *.up.sql
var FS embed.FS
