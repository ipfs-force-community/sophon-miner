package api

import (
	"context"

	"github.com/google/uuid"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
)

type Common interface {

	// Version provides information about API provider
	Version(context.Context) (sharedTypes.Version, error) //perm:read

	LogList(context.Context) ([]string, error)         //perm:write
	LogSetLevel(context.Context, string, string) error //perm:write

	// trigger graceful shutdown
	Shutdown(context.Context) error //perm:admin

	// Session returns a random UUID of api provider session
	Session(context.Context) (uuid.UUID, error) //perm:read

	Closing(context.Context) (<-chan struct{}, error) //perm:read
}
