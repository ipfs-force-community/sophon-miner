package apistruct

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs-force-community/venus-wallet/core"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/chain/actors/builtin/miner"
	"github.com/filecoin-project/venus-miner/chain/types"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

// All permissions are listed in permissioned.go
var _ = AllPermissions

type CommonStruct struct {
	Internal struct {
		AuthVerify func(ctx context.Context, token string) ([]auth.Permission, error) `perm:"read"`
		AuthNew    func(ctx context.Context, perms []auth.Permission) ([]byte, error) `perm:"admin"`

		ID      func(context.Context) (peer.ID, error)     `perm:"read"`
		Version func(context.Context) (api.Version, error) `perm:"read"`

		LogList     func(context.Context) ([]string, error)     `perm:"write"`
		LogSetLevel func(context.Context, string, string) error `perm:"write"`

		Shutdown func(context.Context) error                    `perm:"admin"`
		Session  func(context.Context) (uuid.UUID, error)       `perm:"read"`
		Closing  func(context.Context) (<-chan struct{}, error) `perm:"read"`
	}
}

// FullNodeStruct implements API passing calls to user-provided function values.
type FullNodeStruct struct {
	CommonStruct

	Internal struct {
		ChainHead         func(context.Context) (*types.TipSet, error)                 `perm:"read"`
		ChainTipSetWeight func(context.Context, types.TipSetKey) (types.BigInt, error) `perm:"read"`

		BeaconGetEntry func(ctx context.Context, epoch abi.ChainEpoch) (*types.BeaconEntry, error) `perm:"read"`

		SyncState func(context.Context) (*api.SyncState, error) `perm:"read"`

		SyncSubmitBlock func(ctx context.Context, blk *types.BlockMsg) error `perm:"write"`

		MpoolSelect  func(context.Context, types.TipSetKey, float64) ([]*types.SignedMessage, error)     `perm:"read"`
		MpoolSelects func(context.Context, types.TipSetKey, []float64) ([][]*types.SignedMessage, error) `perm:"read"`

		MinerGetBaseInfo func(context.Context, address.Address, abi.ChainEpoch, types.TipSetKey) (*api.MiningBaseInfo, error) `perm:"read"`
		MinerCreateBlock func(context.Context, *api.BlockTemplate) (*types.BlockMsg, error)                                   `perm:"write"`

		WalletSign func(context.Context, address.Address, []byte, core.MsgMeta) (*crypto.Signature, error) `perm:"sign"`

		StateMinerInfo       func(context.Context, address.Address, types.TipSetKey) (miner.MinerInfo, error)                            `perm:"read"`
		StateMinerDeadlines  func(context.Context, address.Address, types.TipSetKey) ([]api.Deadline, error)                             `perm:"read"`
		StateMinerPartitions func(ctx context.Context, m address.Address, dlIdx uint64, tsk types.TipSetKey) ([]api.Partition, error)    `perm:"read"`
		StateSectorGetInfo   func(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorOnChainInfo, error) `perm:"read"`
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
func (c *CommonStruct) Version(ctx context.Context) (api.Version, error) {
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

// FullNodeStruct
func (c *FullNodeStruct) MinerCreateBlock(ctx context.Context, bt *api.BlockTemplate) (*types.BlockMsg, error) {
	return c.Internal.MinerCreateBlock(ctx, bt)
}

func (c *FullNodeStruct) ChainHead(ctx context.Context) (*types.TipSet, error) {
	return c.Internal.ChainHead(ctx)
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

var _ api.Common = &CommonStruct{}
var _ api.FullNode = &FullNodeStruct{}

type MinerStruct struct {
	CommonStruct

	Internal struct {
		AddAddress      func(dtypes.MinerInfo) error                         `perm:"write"`
		UpdateAddress   func(dtypes.MinerInfo) error                         `perm:"write"`
		RemoveAddress   func(address.Address) error                          `perm:"write"`
		ListAddress     func() ([]dtypes.MinerInfo, error)                   `perm:"read"`
		StatesForMining func([]address.Address) ([]dtypes.MinerState, error) `perm:"read"`
		Start           func(context.Context, address.Address) error         `perm:"write"`
		Stop            func(context.Context, address.Address) error         `perm:"write"`
	}
}

func (s *MinerStruct) AddAddress(miner dtypes.MinerInfo) error {
	return s.Internal.AddAddress(miner)
}

func (s *MinerStruct) UpdateAddress(miner dtypes.MinerInfo) error {
	return s.Internal.UpdateAddress(miner)
}

func (s *MinerStruct) RemoveAddress(addr address.Address) error {
	return s.Internal.RemoveAddress(addr)
}

func (s *MinerStruct) ListAddress() ([]dtypes.MinerInfo, error) {
	return s.Internal.ListAddress()
}

func (s *MinerStruct) StatesForMining(addrs []address.Address) ([]dtypes.MinerState, error) {
	return s.Internal.StatesForMining(addrs)
}

func (s *MinerStruct) Start(ctx context.Context, addr address.Address) error {
	return s.Internal.Start(ctx, addr)
}

func (s *MinerStruct) Stop(ctx context.Context, addr address.Address) error {
	return s.Internal.Stop(ctx, addr)
}

var _ api.MinerAPI = &MinerStruct{}
