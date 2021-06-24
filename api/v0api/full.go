package v0api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"

	"github.com/filecoin-project/venus-wallet/core"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/chain/actors/builtin/miner"
	"github.com/filecoin-project/venus-miner/chain/types"
)

// FullNode API is a low-level interface to the Filecoin network full node
type FullNode interface {
	Common

	// MethodGroup: Chain
	// The Chain method group contains methods for interacting with the
	// blockchain, but that do not require any form of state computation.

	// ChainHead returns the current head of the chain.
	ChainHead(context.Context) (*types.TipSet, error) //perm:read

	// ChainGetTipSetByHeight looks back for a tipset at the specified epoch.
	// If there are no blocks at the specified epoch, a tipset at an earlier epoch
	// will be returned.
	ChainGetTipSetByHeight(context.Context, abi.ChainEpoch, types.TipSetKey) (*types.TipSet, error) //perm:read

	// ChainTipSetWeight computes weight for the specified tipset.
	ChainTipSetWeight(context.Context, types.TipSetKey) (types.BigInt, error) //perm:read

	// MethodGroup: Beacon
	// The Beacon method group contains methods for interacting with the random beacon (DRAND)

	// BeaconGetEntry returns the beacon entry for the given filecoin epoch. If
	// the entry has not yet been produced, the call will block until the entry
	// becomes available
	BeaconGetEntry(ctx context.Context, epoch abi.ChainEpoch) (*types.BeaconEntry, error) //perm:read

	// MethodGroup: Sync
	// The Sync method group contains methods for interacting with and
	// observing the lotus sync service.

	// SyncState returns the current status of the lotus sync system.
	SyncState(context.Context) (*api.SyncState, error) //perm:read

	// SyncSubmitBlock can be used to submit a newly created block to the.
	// network through this node
	SyncSubmitBlock(ctx context.Context, blk *types.BlockMsg) error //perm:write

	// MethodGroup: Mpool
	// The Mpool methods are for interacting with the message pool. The message pool
	// manages all incoming and outgoing 'messages' going over the network.

	// MpoolSelect returns a list of pending messages for inclusion in the next block
	MpoolSelect(context.Context, types.TipSetKey, float64) ([]*types.SignedMessage, error) //perm:read

	// MpoolSelects returns a list of pending messages for inclusion in the blocks
	MpoolSelects(context.Context, types.TipSetKey, []float64) ([][]*types.SignedMessage, error) //perm:write

	// MethodGroup: Miner

	MinerGetBaseInfo(context.Context, address.Address, abi.ChainEpoch, types.TipSetKey) (*api.MiningBaseInfo, error) //perm:read
	MinerCreateBlock(context.Context, *api.BlockTemplate) (*types.BlockMsg, error)                                   //perm:write

	// // UX ?

	// MethodGroup: Wallet

	// WalletSign signs the given bytes using the given address.
	WalletSign(context.Context, address.Address, []byte, core.MsgMeta) (*crypto.Signature, error) //perm:sign
	//WalletSign(context.Context, address.Address, []byte) (*crypto.Signature, error) //perm:sign

	// Other

	// MethodGroup: State
	// The State methods are used to query, inspect, and interact with chain state.
	// Most methods take a TipSetKey as a parameter. The state looked up is the parent state of the tipset.
	// A nil TipSetKey can be provided as a param, this will cause the heaviest tipset in the chain to be used.

	// StateMinerInfo returns info about the indicated miner
	StateMinerInfo(context.Context, address.Address, types.TipSetKey) (miner.MinerInfo, error) //perm:read
	// StateMinerDeadlines returns all the proving deadlines for the given miner
	StateMinerDeadlines(context.Context, address.Address, types.TipSetKey) ([]api.Deadline, error) //perm:read
	// StateSectorGetInfo returns the on-chain info for the specified miner's sector. Returns null in case the sector info isn't found
	// NOTE: returned info.Expiration may not be accurate in some cases, use StateSectorExpiration to get accurate
	// expiration epoch
	StateSectorGetInfo(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorOnChainInfo, error) //perm:read
	// StateMinerPartitions returns all partitions in the specified deadline
	StateMinerPartitions(ctx context.Context, m address.Address, dlIdx uint64, tsk types.TipSetKey) ([]api.Partition, error) //perm:read
}
