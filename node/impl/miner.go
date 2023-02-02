package impl

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-auth/jwtclient"

	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/node/impl/common"
	"github.com/filecoin-project/venus-miner/types"
)

type MinerAPI struct {
	common.CommonAPI
	miner.MiningAPI

	AuthClient jwtclient.IAuthClient
}

func (m *MinerAPI) UpdateAddress(ctx context.Context, skip int64, limit int64) ([]types.MinerInfo, error) {
	return m.MiningAPI.UpdateAddress(ctx, skip, limit)
}

func (m *MinerAPI) ListAddress(ctx context.Context) ([]types.MinerInfo, error) {
	mis, err := m.MiningAPI.ListAddress(ctx)
	if err != nil {
		return nil, err
	}
	ret := filter(mis, func(mi types.MinerInfo) bool {
		return jwtclient.CheckPermissionByMiner(ctx, m.AuthClient, mi.Addr) == nil
	})

	return ret, nil
}

func (m *MinerAPI) StatesForMining(ctx context.Context, addrs []address.Address) ([]types.MinerState, error) {
	addrsAllowed := filter(addrs, func(addr address.Address) bool {
		return jwtclient.CheckPermissionByMiner(ctx, m.AuthClient, addr) == nil
	})
	return m.MiningAPI.StatesForMining(ctx, addrsAllowed)
}

func (m *MinerAPI) CountWinners(ctx context.Context, addrs []address.Address, start abi.ChainEpoch, end abi.ChainEpoch) ([]types.CountWinners, error) {
	addrsAllowed := filter(addrs, func(addr address.Address) bool {
		return jwtclient.CheckPermissionByMiner(ctx, m.AuthClient, addr) == nil
	})
	return m.MiningAPI.CountWinners(ctx, addrsAllowed, start, end)
}

func (m *MinerAPI) WarmupForMiner(ctx context.Context, maddr address.Address) error {
	if err := jwtclient.CheckPermissionByMiner(ctx, m.AuthClient, maddr); err != nil {
		return err
	}
	return m.MiningAPI.WarmupForMiner(ctx, maddr)
}

func (m *MinerAPI) Start(ctx context.Context, addrs []address.Address) error {
	addrsAllowed := filter(addrs, func(addr address.Address) bool {
		return jwtclient.CheckPermissionByMiner(ctx, m.AuthClient, addr) == nil
	})
	return m.MiningAPI.ManualStart(ctx, addrsAllowed)
}

func (m *MinerAPI) Stop(ctx context.Context, addrs []address.Address) error {
	addrsAllowed := filter(addrs, func(addr address.Address) bool {
		return jwtclient.CheckPermissionByMiner(ctx, m.AuthClient, addr) == nil
	})
	return m.MiningAPI.ManualStop(ctx, addrsAllowed)
}

func filter[T any](src []T, pass func(T) bool) []T {
	ret := make([]T, 0)
	for _, addr := range src {
		if pass(addr) {
			ret = append(ret, addr)
		}
	}
	return ret
}
