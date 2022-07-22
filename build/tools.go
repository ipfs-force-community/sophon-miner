//go:build tools
// +build tools

package build

// in order to gen check: ci
import (
	_ "github.com/golang/mock/mockgen"
	_ "golang.org/x/tools/cmd/stringer"
)
