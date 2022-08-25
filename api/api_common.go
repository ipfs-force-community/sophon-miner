package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/filecoin-project/venus/venus-shared/api"
)

type Common interface {

	// Version provides information about API provider
	Version(context.Context) (APIVersion, error) //perm:read

	LogList(context.Context) ([]string, error)         //perm:write
	LogSetLevel(context.Context, string, string) error //perm:write

	// trigger graceful shutdown
	Shutdown(context.Context) error //perm:admin

	// Session returns a random UUID of api provider session
	Session(context.Context) (uuid.UUID, error) //perm:read

	Closing(context.Context) (<-chan struct{}, error) //perm:read
}

// APIVersion provides various build-time information
type APIVersion struct {
	Version string

	// APIVersion is a binary encoded semver version of the remote implementing
	// this api
	//
	// See APIVersion in build/version.go
	APIVersion api.Version
}

func (v APIVersion) String() string {
	return fmt.Sprintf("%s+api%s", v.Version, v.APIVersion.String())
}
