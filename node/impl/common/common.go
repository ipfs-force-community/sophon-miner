package common

import (
	"context"

	types2 "github.com/filecoin-project/venus/venus-shared/types"

	"github.com/filecoin-project/venus-miner/types"
	"github.com/google/uuid"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/build"
)

var session = uuid.New()

type CommonAPI struct {
	fx.In

	ShutdownChan types.ShutdownChan
}

var apiVersion = types2.NewVer(1, 2, 0)

func (a *CommonAPI) Version(context.Context) (api.APIVersion, error) {
	return api.APIVersion{
		Version:    build.UserVersion(),
		APIVersion: apiVersion,
	}, nil
}

func (a *CommonAPI) LogList(context.Context) ([]string, error) {
	return logging.GetSubsystems(), nil
}

func (a *CommonAPI) LogSetLevel(ctx context.Context, subsystem, level string) error {
	return logging.SetLogLevel(subsystem, level)
}

func (a *CommonAPI) Shutdown(ctx context.Context) error {
	a.ShutdownChan <- struct{}{}
	return nil
}

func (a *CommonAPI) Session(ctx context.Context) (uuid.UUID, error) {
	return session, nil
}

func (a *CommonAPI) Closing(ctx context.Context) (<-chan struct{}, error) {
	return make(chan struct{}), nil // relies on jsonrpc closing
}

var _ api.Common = &CommonAPI{}
