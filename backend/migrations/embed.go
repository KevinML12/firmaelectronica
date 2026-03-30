package migrations

import "embed"

//go:embed *.up.sql
var UpSQL embed.FS
