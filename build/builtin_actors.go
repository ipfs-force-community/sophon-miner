package build

import (
	"embed"
)

//go:embed builtin-actors/v8
var Actorsv8FS embed.FS

//go:embed builtin-actors/v7
var Actorsv7FS embed.FS
