package repo

import (
	"context"
	"errors"
	types2 "github.com/filecoin-project/venus-miner/types"

	"github.com/ipfs/go-datastore"
	"github.com/multiformats/go-multiaddr"
)

var (
	ErrNoAPIEndpoint     = errors.New("API not running (no endpoint)")
	ErrRepoAlreadyLocked = errors.New("repo is already locked")
)

type Repo interface {
	// APIEndpoint returns multiaddress for communication with Lotus API
	APIEndpoint() (multiaddr.Multiaddr, error)

	// APIToken returns JWT API Token for use in operations that require auth
	APIToken() ([]byte, error)

	// Lock locks the repo for exclusive use.
	Lock(RepoType) (LockedRepo, error)
}

type LockedRepo interface {
	// Close closes repo and removes lock.
	Close() error

	// Returns datastore defined in this repo.
	// The supplied context must only be used to initialize the datastore.
	// The implementation should not retain the context for usage throughout
	// the lifecycle.
	Datastore(ctx context.Context, namespace string) (datastore.Batching, error)

	// Returns config in this repo
	Config() (interface{}, error)
	SetConfig(func(interface{})) error

	// SetAPIEndpoint sets the endpoint of the current API
	// so it can be read by API clients
	SetAPIEndpoint(multiaddr.Multiaddr) error

	// SetAPIToken sets JWT API Token for CLI
	SetAPIToken([]byte) error

	// KeyStore returns store of private keys for Filecoin transactions
	KeyStore() (types2.KeyStore, error)

	// Path returns absolute path of the repo
	Path() string
}
