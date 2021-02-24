package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/venus-miner/build"
	"github.com/libp2p/go-libp2p-core/network"
)

type Common interface {

	// MethodGroup: Auth

	AuthVerify(ctx context.Context, token string) ([]auth.Permission, error)
	AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error)

	// Version provides information about API provider
	Version(context.Context) (Version, error)

	LogList(context.Context) ([]string, error)
	LogSetLevel(context.Context, string, string) error

	// trigger graceful shutdown
	Shutdown(context.Context) error

	// Session returns a random UUID of api provider session
	Session(context.Context) (uuid.UUID, error)

	Closing(context.Context) (<-chan struct{}, error)
}

// Version provides various build-time information
type Version struct {
	Version string

	// APIVersion is a binary encoded semver version of the remote implementing
	// this api
	//
	// See APIVersion in build/version.go
	APIVersion build.Version

	// TODO: git commit / os / genesis cid?

	// Seconds
	BlockDelay uint64
}

func (v Version) String() string {
	return fmt.Sprintf("%s+api%s", v.Version, v.APIVersion.String())
}

type NatInfo struct {
	Reachability network.Reachability
	PublicAddr   string
}
