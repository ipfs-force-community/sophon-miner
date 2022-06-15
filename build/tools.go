//go:build tools
// +build tools

package build

import (
	_ "github.com/GeertJohan/go.rice/rice"
	_ "golang.org/x/tools/cmd/stringer"
)
