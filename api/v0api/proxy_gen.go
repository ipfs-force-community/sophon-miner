package v0api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"

	"github.com/filecoin-project/venus-wallet/core"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus/pkg/types/specactors/builtin/miner"
	"github.com/filecoin-project/venus-miner/chain/types"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

// FullNodeStruct implements API passing calls to user-provided function values.
type FullNodeStruct struct {
	CommonStruct

	Internal struct {
		ChainHead              func(p0 context.Context) (*types.TipSet, error)                                        `perm:"read"`
		ChainGetTipSetByHeight func(p0 context.Context, p1 abi.ChainEpoch, p2 types.TipSetKey) (*types.TipSet, error) `perm:"read"`
		ChainTipSetWeight      func(p0 context.Context, p1 types.TipSetKey) (types.BigInt, error)                     `perm:"read"`

		BeaconGetEntry func(p0 context.Context, p1 abi.ChainEpoch) (*types.BeaconEntry, error) `perm:"read"`

		SyncState func(p0 context.Context) (*api.SyncState, error) `perm:"read"`

		SyncSubmitBlock func(p0 context.Context, p1 *types.BlockMsg) error `perm:"write"`

		MpoolSelect  func(p0 context.Context, p1 types.TipSetKey, p2 float64) ([]*types.SignedMessage, error)     `perm:"read"`
		MpoolSelects func(p0 context.Context, p1 types.TipSetKey, p2 []float64) ([][]*types.SignedMessage, error) `perm:"read"`

		MinerGetBaseInfo func(p0 context.Context, p1 address.Address, p2 abi.ChainEpoch, p3 types.TipSetKey) (*api.MiningBaseInfo, error) `perm:"read"`
		MinerCreateBlock func(p0 context.Context, p1 *api.BlockTemplate) (*types.BlockMsg, error)                                         `perm:"write"`

		WalletSign func(p0 context.Context, p1 address.Address, p2 []byte, p3 core.MsgMeta) (*crypto.Signature, error) `perm:"sign"`

		StateMinerInfo       func(p0 context.Context, p1 address.Address, p2 types.TipSetKey) (miner.MinerInfo, error)                               `perm:"read"`
		StateMinerDeadlines  func(p0 context.Context, p1 address.Address, p2 types.TipSetKey) ([]api.Deadline, error)                                `perm:"read"`
		StateMinerPartitions func(p0 context.Context, p1 address.Address, p2 uint64, p3 types.TipSetKey) ([]api.Partition, error)                    `perm:"read"`
		StateSectorGetInfo   func(p0 context.Context, p1 address.Address, p2 abi.SectorNumber, p3 types.TipSetKey) (*miner.SectorOnChainInfo, error) `perm:"read"`
	}
}

// FullNodeStruct
func (c *FullNodeStruct) MinerCreateBlock(ctx context.Context, bt *api.BlockTemplate) (*types.BlockMsg, error) {
	return c.Internal.MinerCreateBlock(ctx, bt)
}

func (c *FullNodeStruct) ChainHead(ctx context.Context) (*types.TipSet, error) {
	return c.Internal.ChainHead(ctx)
}

func (c *FullNodeStruct) ChainGetTipSetByHeight(ctx context.Context, h abi.ChainEpoch, tsk types.TipSetKey) (*types.TipSet, error) {
	return c.Internal.ChainGetTipSetByHeight(ctx, h, tsk)
}

func (c *FullNodeStruct) ChainTipSetWeight(ctx context.Context, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.ChainTipSetWeight(ctx, tsk)
}

func (c *FullNodeStruct) WalletSign(ctx context.Context, k address.Address, msg []byte, meta core.MsgMeta) (*crypto.Signature, error) {
	return c.Internal.WalletSign(ctx, k, msg, meta)
}

func (c *FullNodeStruct) BeaconGetEntry(ctx context.Context, epoch abi.ChainEpoch) (*types.BeaconEntry, error) {
	return c.Internal.BeaconGetEntry(ctx, epoch)
}

func (c *FullNodeStruct) SyncState(ctx context.Context) (*api.SyncState, error) {
	return c.Internal.SyncState(ctx)
}

func (c *FullNodeStruct) MinerGetBaseInfo(ctx context.Context, maddr address.Address, epoch abi.ChainEpoch, tsk types.TipSetKey) (*api.MiningBaseInfo, error) {
	return c.Internal.MinerGetBaseInfo(ctx, maddr, epoch, tsk)
}

func (c *FullNodeStruct) SyncSubmitBlock(ctx context.Context, blk *types.BlockMsg) error {
	return c.Internal.SyncSubmitBlock(ctx, blk)
}

func (c *FullNodeStruct) StateMinerInfo(ctx context.Context, actor address.Address, tsk types.TipSetKey) (miner.MinerInfo, error) {
	return c.Internal.StateMinerInfo(ctx, actor, tsk)
}

func (c *FullNodeStruct) StateMinerDeadlines(ctx context.Context, actor address.Address, tsk types.TipSetKey) ([]api.Deadline, error) {
	return c.Internal.StateMinerDeadlines(ctx, actor, tsk)
}

func (c *FullNodeStruct) StateMinerPartitions(ctx context.Context, m address.Address, dlIdx uint64, tsk types.TipSetKey) ([]api.Partition, error) {
	return c.Internal.StateMinerPartitions(ctx, m, dlIdx, tsk)
}

func (c *FullNodeStruct) StateSectorGetInfo(ctx context.Context, maddr address.Address, n abi.SectorNumber, tsk types.TipSetKey) (*miner.SectorOnChainInfo, error) {
	return c.Internal.StateSectorGetInfo(ctx, maddr, n, tsk)
}

func (c *FullNodeStruct) MpoolSelect(ctx context.Context, tsk types.TipSetKey, ticketQuality float64) ([]*types.SignedMessage, error) {
	return c.Internal.MpoolSelect(ctx, tsk, ticketQuality)
}

func (c *FullNodeStruct) MpoolSelects(ctx context.Context, tsk types.TipSetKey, ticketQualitys []float64) ([][]*types.SignedMessage, error) {
	return c.Internal.MpoolSelects(ctx, tsk, ticketQualitys)
}

var _ Common = &CommonStruct{}
var _ FullNode = &FullNodeStruct{}

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
