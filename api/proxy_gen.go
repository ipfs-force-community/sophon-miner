package api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

// All permissions are listed in permissioned.go
var _ = AllPermissions

type CommonStruct struct {
	Internal struct {
		AuthVerify func(ctx context.Context, token string) ([]auth.Permission, error) `perm:"read"`
		AuthNew    func(ctx context.Context, perms []auth.Permission) ([]byte, error) `perm:"admin"`

		ID      func(context.Context) (peer.ID, error)    `perm:"read"`
		Version func(context.Context) (APIVersion, error) `perm:"read"`

		LogList     func(context.Context) ([]string, error)     `perm:"write"`
		LogSetLevel func(context.Context, string, string) error `perm:"write"`

		Shutdown func(context.Context) error                    `perm:"admin"`
		Session  func(context.Context) (uuid.UUID, error)       `perm:"read"`
		Closing  func(context.Context) (<-chan struct{}, error) `perm:"read"`
	}
}

// CommonStruct

func (c *CommonStruct) AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) {
	return c.Internal.AuthVerify(ctx, token)
}

func (c *CommonStruct) AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error) {
	return c.Internal.AuthNew(ctx, perms)
}

// Version implements API.Version
func (c *CommonStruct) Version(ctx context.Context) (APIVersion, error) {
	return c.Internal.Version(ctx)
}

func (c *CommonStruct) LogList(ctx context.Context) ([]string, error) {
	return c.Internal.LogList(ctx)
}

func (c *CommonStruct) LogSetLevel(ctx context.Context, group, level string) error {
	return c.Internal.LogSetLevel(ctx, group, level)
}

func (c *CommonStruct) Shutdown(ctx context.Context) error {
	return c.Internal.Shutdown(ctx)
}

func (c *CommonStruct) Session(ctx context.Context) (uuid.UUID, error) {
	return c.Internal.Session(ctx)
}

func (c *CommonStruct) Closing(ctx context.Context) (<-chan struct{}, error) {
	return c.Internal.Closing(ctx)
}

var _ Common = &CommonStruct{}

type MinerStruct struct {
	CommonStruct

	Internal struct {
		UpdateAddress   func(context.Context, int64, int64) ([]dtypes.MinerInfo, error)                                         `perm:"write"`
		ListAddress     func(context.Context) ([]dtypes.MinerInfo, error)                                                       `perm:"read"`
		StatesForMining func(context.Context, []address.Address) ([]dtypes.MinerState, error)                                   `perm:"read"`
		CountWinners    func(context.Context, []address.Address, abi.ChainEpoch, abi.ChainEpoch) ([]dtypes.CountWinners, error) `perm:"read"`
		Start           func(context.Context, []address.Address) error                                                          `perm:"admin"`
		Stop            func(context.Context, []address.Address) error                                                          `perm:"admin"`
		AddAddress      func(context.Context, dtypes.MinerInfo) error                                                           `perm:"admin"`
	}
}

func (s *MinerStruct) UpdateAddress(ctx context.Context, skip int64, limit int64) ([]dtypes.MinerInfo, error) {
	return s.Internal.UpdateAddress(ctx, skip, limit)
}

func (s *MinerStruct) ListAddress(ctx context.Context) ([]dtypes.MinerInfo, error) {
	return s.Internal.ListAddress(ctx)
}

func (s *MinerStruct) StatesForMining(ctx context.Context, addrs []address.Address) ([]dtypes.MinerState, error) {
	return s.Internal.StatesForMining(ctx, addrs)
}

func (s *MinerStruct) CountWinners(ctx context.Context, addrs []address.Address, start abi.ChainEpoch, end abi.ChainEpoch) ([]dtypes.CountWinners, error) {
	return s.Internal.CountWinners(ctx, addrs, start, end)
}

func (s *MinerStruct) Start(ctx context.Context, addrs []address.Address) error {
	return s.Internal.Start(ctx, addrs)
}

func (s *MinerStruct) Stop(ctx context.Context, addrs []address.Address) error {
	return s.Internal.Stop(ctx, addrs)
}

func (s *MinerStruct) AddAddress(ctx context.Context, mi dtypes.MinerInfo) error {
	return s.Internal.AddAddress(ctx, mi)
}

var _ MinerAPI = &MinerStruct{}
