package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/dline"

	"github.com/filecoin-project/venus-miner/chain/actors/builtin"
	"github.com/filecoin-project/venus-miner/chain/actors/builtin/miner"
	"github.com/filecoin-project/venus-miner/chain/actors/builtin/power"
	"github.com/filecoin-project/venus-miner/chain/types"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

// FullNode API is a low-level interface to the Filecoin network full node
type FullNode interface {
	Common

	// MethodGroup: Chain
	// The Chain method group contains methods for interacting with the
	// blockchain, but that do not require any form of state computation.

	// ChainNotify returns channel with chain head updates.
	// First message is guaranteed to be of len == 1, and type == 'current'.
	ChainNotify(context.Context) (<-chan []*HeadChange, error)

	// ChainHead returns the current head of the chain.
	ChainHead(context.Context) (*types.TipSet, error)

	// ChainGetRandomnessFromTickets is used to sample the chain for randomness.
	ChainGetRandomnessFromTickets(ctx context.Context, tsk types.TipSetKey, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error)

	// ChainGetRandomnessFromBeacon is used to sample the beacon for randomness.
	ChainGetRandomnessFromBeacon(ctx context.Context, tsk types.TipSetKey, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error)

	// ChainGetBlock returns the block specified by the given CID.
	ChainGetBlock(context.Context, cid.Cid) (*types.BlockHeader, error)
	// ChainGetTipSet returns the tipset specified by the given TipSetKey.
	ChainGetTipSet(context.Context, types.TipSetKey) (*types.TipSet, error)

	// ChainGetBlockMessages returns messages stored in the specified block.
	ChainGetBlockMessages(ctx context.Context, blockCid cid.Cid) (*BlockMessages, error)

	// ChainGetParentReceipts returns receipts for messages in parent tipset of
	// the specified block.
	ChainGetParentReceipts(ctx context.Context, blockCid cid.Cid) ([]*types.MessageReceipt, error)

	// ChainGetParentMessages returns messages stored in parent tipset of the
	// specified block.
	ChainGetParentMessages(ctx context.Context, blockCid cid.Cid) ([]Message, error)

	// ChainGetTipSetByHeight looks back for a tipset at the specified epoch.
	// If there are no blocks at the specified epoch, a tipset at an earlier epoch
	// will be returned.
	ChainGetTipSetByHeight(context.Context, abi.ChainEpoch, types.TipSetKey) (*types.TipSet, error)

	// ChainReadObj reads ipld nodes referenced by the specified CID from chain
	// blockstore and returns raw bytes.
	ChainReadObj(context.Context, cid.Cid) ([]byte, error)

	// ChainDeleteObj deletes node referenced by the given CID
	ChainDeleteObj(context.Context, cid.Cid) error

	// ChainHasObj checks if a given CID exists in the chain blockstore.
	ChainHasObj(context.Context, cid.Cid) (bool, error)

	// ChainStatObj returns statistics about the graph referenced by 'obj'.
	// If 'base' is also specified, then the returned stat will be a diff
	// between the two objects.
	ChainStatObj(ctx context.Context, obj cid.Cid, base cid.Cid) (ObjStat, error)

	// ChainSetHead forcefully sets current chain head. Use with caution.
	ChainSetHead(context.Context, types.TipSetKey) error

	// ChainGetGenesis returns the genesis tipset.
	ChainGetGenesis(context.Context) (*types.TipSet, error)

	// ChainTipSetWeight computes weight for the specified tipset.
	ChainTipSetWeight(context.Context, types.TipSetKey) (types.BigInt, error)
	ChainGetNode(ctx context.Context, p string) (*IpldObject, error)

	// ChainGetMessage reads a message referenced by the specified CID from the
	// chain blockstore.
	ChainGetMessage(context.Context, cid.Cid) (*types.Message, error)

	// ChainGetPath returns a set of revert/apply operations needed to get from
	// one tipset to another, for example:
	//```
	//        to
	//         ^
	// from   tAA
	//   ^     ^
	// tBA    tAB
	//  ^---*--^
	//      ^
	//     tRR
	//```
	// Would return `[revert(tBA), apply(tAB), apply(tAA)]`
	ChainGetPath(ctx context.Context, from types.TipSetKey, to types.TipSetKey) ([]*HeadChange, error)

	// ChainExport returns a stream of bytes with CAR dump of chain data.
	// The exported chain data includes the header chain from the given tipset
	// back to genesis, the entire genesis state, and the most recent 'nroots'
	// state trees.
	// If oldmsgskip is set, messages from before the requested roots are also not included.
	ChainExport(ctx context.Context, nroots abi.ChainEpoch, oldmsgskip bool, tsk types.TipSetKey) (<-chan []byte, error)

	// MethodGroup: Beacon
	// The Beacon method group contains methods for interacting with the random beacon (DRAND)

	// BeaconGetEntry returns the beacon entry for the given filecoin epoch. If
	// the entry has not yet been produced, the call will block until the entry
	// becomes available
	BeaconGetEntry(ctx context.Context, epoch abi.ChainEpoch) (*types.BeaconEntry, error)

	// GasEstimateFeeCap estimates gas fee cap
	GasEstimateFeeCap(context.Context, *types.Message, int64, types.TipSetKey) (types.BigInt, error)

	// GasEstimateGasLimit estimates gas used by the message and returns it.
	// It fails if message fails to execute.
	GasEstimateGasLimit(context.Context, *types.Message, types.TipSetKey) (int64, error)

	// GasEstimateGasPremium estimates what gas price should be used for a
	// message to have high likelihood of inclusion in `nblocksincl` epochs.

	GasEstimateGasPremium(_ context.Context, nblocksincl uint64,
		sender address.Address, gaslimit int64, tsk types.TipSetKey) (types.BigInt, error)

	// GasEstimateMessageGas estimates gas values for unset message gas fields
	GasEstimateMessageGas(context.Context, *types.Message, *MessageSendSpec, types.TipSetKey) (*types.Message, error)

	// MethodGroup: Sync
	// The Sync method group contains methods for interacting with and
	// observing the venus sync service.

	// SyncState returns the current status of the venus sync system.
	SyncState(context.Context) (*SyncState, error)

	// SyncSubmitBlock can be used to submit a newly created block to the.
	// network through this node
	SyncSubmitBlock(ctx context.Context, blk *types.BlockMsg) error

	// SyncIncomingBlocks returns a channel streaming incoming, potentially not
	// yet synced block headers.
	SyncIncomingBlocks(ctx context.Context) (<-chan *types.BlockHeader, error)

	// SyncCheckpoint marks a blocks as checkpointed, meaning that it won't ever fork away from it.
	SyncCheckpoint(ctx context.Context, tsk types.TipSetKey) error

	// SyncMarkBad marks a blocks as bad, meaning that it won't ever by synced.
	// Use with extreme caution.
	SyncMarkBad(ctx context.Context, bcid cid.Cid) error

	// SyncUnmarkBad unmarks a blocks as bad, making it possible to be validated and synced again.
	SyncUnmarkBad(ctx context.Context, bcid cid.Cid) error

	// SyncUnmarkAllBad purges bad block cache, making it possible to sync to chains previously marked as bad
	SyncUnmarkAllBad(ctx context.Context) error

	// SyncCheckBad checks if a block was marked as bad, and if it was, returns
	// the reason.
	SyncCheckBad(ctx context.Context, bcid cid.Cid) (string, error)

	// SyncValidateTipset indicates whether the provided tipset is valid or not
	SyncValidateTipset(ctx context.Context, tsk types.TipSetKey) (bool, error)

	// MethodGroup: Mpool
	// The Mpool methods are for interacting with the message pool. The message pool
	// manages all incoming and outgoing 'messages' going over the network.

	// MpoolPending returns pending mempool messages.
	MpoolPending(context.Context, types.TipSetKey) ([]*types.SignedMessage, error)

	// MpoolSelect returns a list of pending messages for inclusion in the next block
	MpoolSelect(context.Context, types.TipSetKey, float64) ([]*types.SignedMessage, error)

	// MpoolPush pushes a signed message to mempool.
	MpoolPush(context.Context, *types.SignedMessage) (cid.Cid, error)

	// MpoolPushUntrusted pushes a signed message to mempool from untrusted sources.
	MpoolPushUntrusted(context.Context, *types.SignedMessage) (cid.Cid, error)

	// MpoolPushMessage atomically assigns a nonce, signs, and pushes a message
	// to mempool.
	// maxFee is only used when GasFeeCap/GasPremium fields aren't specified
	//
	// When maxFee is set to 0, MpoolPushMessage will guess appropriate fee
	// based on current chain conditions
	MpoolPushMessage(ctx context.Context, msg *types.Message, spec *MessageSendSpec) (*types.SignedMessage, error)

	// MpoolBatchPush batch pushes a signed message to mempool.
	MpoolBatchPush(context.Context, []*types.SignedMessage) ([]cid.Cid, error)

	// MpoolBatchPushUntrusted batch pushes a signed message to mempool from untrusted sources.
	MpoolBatchPushUntrusted(context.Context, []*types.SignedMessage) ([]cid.Cid, error)

	// MpoolBatchPushMessage batch pushes a unsigned message to mempool.
	MpoolBatchPushMessage(context.Context, []*types.Message, *MessageSendSpec) ([]*types.SignedMessage, error)

	// MpoolGetNonce gets next nonce for the specified sender.
	// Note that this method may not be atomic. Use MpoolPushMessage instead.
	MpoolGetNonce(context.Context, address.Address) (uint64, error)
	MpoolSub(context.Context) (<-chan MpoolUpdate, error)

	// MpoolClear clears pending messages from the mpool
	MpoolClear(context.Context, bool) error

	// MpoolGetConfig returns (a copy of) the current mpool config
	MpoolGetConfig(context.Context) (*types.MpoolConfig, error)
	// MpoolSetConfig sets the mpool config to (a copy of) the supplied config
	MpoolSetConfig(context.Context, *types.MpoolConfig) error

	// MethodGroup: Miner

	MinerGetBaseInfo(context.Context, address.Address, abi.ChainEpoch, types.TipSetKey) (*MiningBaseInfo, error)
	MinerCreateBlock(context.Context, *BlockTemplate) (*types.BlockMsg, error)

	// // UX ?

	// MethodGroup: Wallet

	// WalletNew creates a new address in the wallet with the given sigType.
	// Available key types: bls, secp256k1, secp256k1-ledger
	// Support for numerical types: 1 - secp256k1, 2 - BLS is deprecated
	WalletNew(context.Context, types.KeyType) (address.Address, error)
	// WalletHas indicates whether the given address is in the wallet.
	WalletHas(context.Context, address.Address) (bool, error)
	// WalletList lists all the addresses in the wallet.
	WalletList(context.Context) ([]address.Address, error)
	// WalletBalance returns the balance of the given address at the current head of the chain.
	WalletBalance(context.Context, address.Address) (types.BigInt, error)
	// WalletSign signs the given bytes using the given address.
	WalletSign(context.Context, address.Address, []byte) (*crypto.Signature, error)
	// WalletSignMessage signs the given message using the given address.
	WalletSignMessage(context.Context, address.Address, *types.Message) (*types.SignedMessage, error)
	// WalletVerify takes an address, a signature, and some bytes, and indicates whether the signature is valid.
	// The address does not have to be in the wallet.
	WalletVerify(context.Context, address.Address, []byte, *crypto.Signature) (bool, error)
	// WalletDefaultAddress returns the address marked as default in the wallet.
	WalletDefaultAddress(context.Context) (address.Address, error)
	// WalletSetDefault marks the given address as as the default one.
	WalletSetDefault(context.Context, address.Address) error
	// WalletExport returns the private key of an address in the wallet.
	WalletExport(context.Context, address.Address) (*types.KeyInfo, error)
	// WalletImport receives a KeyInfo, which includes a private key, and imports it into the wallet.
	WalletImport(context.Context, *types.KeyInfo) (address.Address, error)
	// WalletDelete deletes an address from the wallet.
	WalletDelete(context.Context, address.Address) error
	// WalletValidateAddress validates whether a given string can be decoded as a well-formed address
	WalletValidateAddress(context.Context, string) (address.Address, error)

	// Other

	// MethodGroup: State
	// The State methods are used to query, inspect, and interact with chain state.
	// Most methods take a TipSetKey as a parameter. The state looked up is the state at that tipset.
	// A nil TipSetKey can be provided as a param, this will cause the heaviest tipset in the chain to be used.

	// StateCall runs the given message and returns its result without any persisted changes.
	StateCall(context.Context, *types.Message, types.TipSetKey) (*InvocResult, error)
	// StateReplay replays a given message, assuming it was included in a block in the specified tipset.
	// If no tipset key is provided, the appropriate tipset is looked up.
	StateReplay(context.Context, types.TipSetKey, cid.Cid) (*InvocResult, error)
	// StateGetActor returns the indicated actor's nonce and balance.
	StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error)
	// StateReadState returns the indicated actor's state.
	StateReadState(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*ActorState, error)
	// StateListMessages looks back and returns all messages with a matching to or from address, stopping at the given height.
	StateListMessages(ctx context.Context, match *MessageMatch, tsk types.TipSetKey, toht abi.ChainEpoch) ([]cid.Cid, error)
	// StateDecodeParams attempts to decode the provided params, based on the recipient actor address and method number.
	StateDecodeParams(ctx context.Context, toAddr address.Address, method abi.MethodNum, params []byte, tsk types.TipSetKey) (interface{}, error)

	// StateNetworkName returns the name of the network the node is synced to
	StateNetworkName(context.Context) (dtypes.NetworkName, error)
	// StateMinerSectors returns info about the given miner's sectors. If the filter bitfield is nil, all sectors are included.
	StateMinerSectors(context.Context, address.Address, *bitfield.BitField, types.TipSetKey) ([]*miner.SectorOnChainInfo, error)
	// StateMinerActiveSectors returns info about sectors that a given miner is actively proving.
	StateMinerActiveSectors(context.Context, address.Address, types.TipSetKey) ([]*miner.SectorOnChainInfo, error)
	// StateMinerProvingDeadline calculates the deadline at some epoch for a proving period
	// and returns the deadline-related calculations.
	StateMinerProvingDeadline(context.Context, address.Address, types.TipSetKey) (*dline.Info, error)
	// StateMinerPower returns the power of the indicated miner
	StateMinerPower(context.Context, address.Address, types.TipSetKey) (*MinerPower, error)
	// StateMinerInfo returns info about the indicated miner
	StateMinerInfo(context.Context, address.Address, types.TipSetKey) (miner.MinerInfo, error)
	// StateMinerDeadlines returns all the proving deadlines for the given miner
	StateMinerDeadlines(context.Context, address.Address, types.TipSetKey) ([]Deadline, error)
	// StateMinerPartitions returns all partitions in the specified deadline
	StateMinerPartitions(ctx context.Context, m address.Address, dlIdx uint64, tsk types.TipSetKey) ([]Partition, error)
	// StateMinerFaults returns a bitfield indicating the faulty sectors of the given miner
	StateMinerFaults(context.Context, address.Address, types.TipSetKey) (bitfield.BitField, error)
	// StateAllMinerFaults returns all non-expired Faults that occur within lookback epochs of the given tipset
	StateAllMinerFaults(ctx context.Context, lookback abi.ChainEpoch, ts types.TipSetKey) ([]*Fault, error)
	// StateMinerRecoveries returns a bitfield indicating the recovering sectors of the given miner
	StateMinerRecoveries(context.Context, address.Address, types.TipSetKey) (bitfield.BitField, error)
	// StateMinerInitialPledgeCollateral returns the precommit deposit for the specified miner's sector
	StateMinerPreCommitDepositForPower(context.Context, address.Address, miner.SectorPreCommitInfo, types.TipSetKey) (types.BigInt, error)
	// StateMinerInitialPledgeCollateral returns the initial pledge collateral for the specified miner's sector
	StateMinerInitialPledgeCollateral(context.Context, address.Address, miner.SectorPreCommitInfo, types.TipSetKey) (types.BigInt, error)
	// StateMinerAvailableBalance returns the portion of a miner's balance that can be withdrawn or spent
	StateMinerAvailableBalance(context.Context, address.Address, types.TipSetKey) (types.BigInt, error)
	// StateMinerSectorAllocated checks if a sector is allocated
	StateMinerSectorAllocated(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (bool, error)
	// StateSectorPreCommitInfo returns the PreCommit info for the specified miner's sector
	StateSectorPreCommitInfo(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (miner.SectorPreCommitOnChainInfo, error)
	// StateSectorGetInfo returns the on-chain info for the specified miner's sector. Returns null in case the sector info isn't found
	// NOTE: returned info.Expiration may not be accurate in some cases, use StateSectorExpiration to get accurate
	// expiration epoch
	StateSectorGetInfo(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorOnChainInfo, error)
	// StateSectorExpiration returns epoch at which given sector will expire
	StateSectorExpiration(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorExpiration, error)
	// StateSectorPartition finds deadline/partition with the specified sector
	StateSectorPartition(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok types.TipSetKey) (*miner.SectorLocation, error)
	// StateSearchMsg searches for a message in the chain, and returns its receipt and the tipset where it was executed
	StateSearchMsg(context.Context, cid.Cid) (*MsgLookup, error)
	// StateSearchMsgLimited looks back up to limit epochs in the chain for a message, and returns its receipt and the tipset where it was executed
	StateSearchMsgLimited(ctx context.Context, msg cid.Cid, limit abi.ChainEpoch) (*MsgLookup, error)
	// StateWaitMsg looks back in the chain for a message. If not found, it blocks until the
	// message arrives on chain, and gets to the indicated confidence depth.
	StateWaitMsg(ctx context.Context, cid cid.Cid, confidence uint64) (*MsgLookup, error)
	// StateWaitMsgLimited looks back up to limit epochs in the chain for a message.
	// If not found, it blocks until the message arrives on chain, and gets to the
	// indicated confidence depth.
	StateWaitMsgLimited(ctx context.Context, cid cid.Cid, confidence uint64, limit abi.ChainEpoch) (*MsgLookup, error)
	// StateListMiners returns the addresses of every miner that has claimed power in the Power Actor
	StateListMiners(context.Context, types.TipSetKey) ([]address.Address, error)
	// StateListActors returns the addresses of every actor in the state
	StateListActors(context.Context, types.TipSetKey) ([]address.Address, error)
	// StateLookupID retrieves the ID address of the given address
	StateLookupID(context.Context, address.Address, types.TipSetKey) (address.Address, error)
	// StateAccountKey returns the public key address of the given ID address
	StateAccountKey(context.Context, address.Address, types.TipSetKey) (address.Address, error)
	// StateChangedActors returns all the actors whose states change between the two given state CIDs
	// TODO: Should this take tipset keys instead?
	StateChangedActors(context.Context, cid.Cid, cid.Cid) (map[string]types.Actor, error)
	// StateGetReceipt returns the message receipt for the given message
	StateGetReceipt(context.Context, cid.Cid, types.TipSetKey) (*types.MessageReceipt, error)
	// StateMinerSectorCount returns the number of sectors in a miner's sector set and proving set
	StateMinerSectorCount(context.Context, address.Address, types.TipSetKey) (MinerSectors, error)
	// StateCompute is a flexible command that applies the given messages on the given tipset.
	// The messages are run as though the VM were at the provided height.
	StateCompute(context.Context, abi.ChainEpoch, []*types.Message, types.TipSetKey) (*ComputeStateOutput, error)
	// StateVerifierStatus returns the data cap for the given address.
	// Returns nil if there is no entry in the data cap table for the
	// address.
	StateVerifierStatus(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*abi.StoragePower, error)
	// StateVerifiedClientStatus returns the data cap for the given address.
	// Returns nil if there is no entry in the data cap table for the
	// address.
	StateVerifiedClientStatus(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*abi.StoragePower, error)
	// StateVerifiedClientStatus returns the address of the Verified Registry's root key
	StateVerifiedRegistryRootKey(ctx context.Context, tsk types.TipSetKey) (address.Address, error)
	// StateDealProviderCollateralBounds returns the min and max collateral a storage provider
	// can issue. It takes the deal size and verified status as parameters.
	StateDealProviderCollateralBounds(context.Context, abi.PaddedPieceSize, bool, types.TipSetKey) (DealCollateralBounds, error)

	// StateCirculatingSupply returns the exact circulating supply of Filecoin at the given tipset.
	// This is not used anywhere in the protocol itself, and is only for external consumption.
	StateCirculatingSupply(context.Context, types.TipSetKey) (abi.TokenAmount, error)
	// StateVMCirculatingSupplyInternal returns an approximation of the circulating supply of Filecoin at the given tipset.
	// This is the value reported by the runtime interface to actors code.
	StateVMCirculatingSupplyInternal(context.Context, types.TipSetKey) (CirculatingSupply, error)
	// StateNetworkVersion returns the network version at the given tipset
	StateNetworkVersion(context.Context, types.TipSetKey) (network.Version, error)

	// CreateBackup creates node backup onder the specified file name. The
	// method requires that the venus daemon is running with the
	// LOTUS_BACKUP_BASE_PATH environment variable set to some path, and that
	// the path specified when calling CreateBackup is within the base path
	CreateBackup(ctx context.Context, fpath string) error
}

type FileRef struct {
	Path  string
	IsCAR bool
}

type MinerSectors struct {
	// Live sectors that should be proven.
	Live uint64
	// Sectors actively contributing to power.
	Active uint64
	// Sectors with failed proofs.
	Faulty uint64
}

type ImportRes struct {
	Root     cid.Cid
	ImportID multistore.StoreID
}

type Import struct {
	Key multistore.StoreID
	Err string

	Root     *cid.Cid
	Source   string
	FilePath string
}

type DealInfo struct {
	ProposalCid cid.Cid
	State       storagemarket.StorageDealStatus
	Message     string // more information about deal state, particularly errors
	Provider    address.Address

	DataRef  *storagemarket.DataRef
	PieceCID cid.Cid
	Size     uint64

	PricePerEpoch types.BigInt
	Duration      uint64

	DealID abi.DealID

	CreationTime time.Time
	Verified     bool

	TransferChannelID *datatransfer.ChannelID
	DataTransfer      *DataTransferChannel
}

type MsgLookup struct {
	Message   cid.Cid // Can be different than requested, in case it was replaced, but only gas values changed
	Receipt   types.MessageReceipt
	ReturnDec interface{}
	TipSet    types.TipSetKey
	Height    abi.ChainEpoch
}

type MsgGasCost struct {
	Message            cid.Cid // Can be different than requested, in case it was replaced, but only gas values changed
	GasUsed            abi.TokenAmount
	BaseFeeBurn        abi.TokenAmount
	OverEstimationBurn abi.TokenAmount
	MinerPenalty       abi.TokenAmount
	MinerTip           abi.TokenAmount
	Refund             abi.TokenAmount
	TotalCost          abi.TokenAmount
}

// BlsMessages[x].cid = Cids[x]
// SecpkMessages[y].cid = Cids[BlsMessages.length + y]
type BlockMessages struct {
	BlsMessages   []*types.Message
	SecpkMessages []*types.SignedMessage

	Cids []cid.Cid
}

type Message struct {
	Cid     cid.Cid
	Message *types.Message
}

type ActorState struct {
	Balance types.BigInt
	State   interface{}
}

type MinerPower struct {
	MinerPower  power.Claim
	TotalPower  power.Claim
	HasMinPower bool
}

type InvocResult struct {
	MsgCid         cid.Cid
	Msg            *types.Message
	MsgRct         *types.MessageReceipt
	GasCost        MsgGasCost
	ExecutionTrace types.ExecutionTrace
	Error          string
	Duration       time.Duration
}

type MethodCall struct {
	types.MessageReceipt
	Error string
}

type StartDealParams struct {
	Data               *storagemarket.DataRef
	Wallet             address.Address
	Miner              address.Address
	EpochPrice         types.BigInt
	MinBlocksDuration  uint64
	ProviderCollateral big.Int
	DealStartEpoch     abi.ChainEpoch
	FastRetrieval      bool
	VerifiedDeal       bool
}

func (s *StartDealParams) UnmarshalJSON(raw []byte) (err error) {
	type sdpAlias StartDealParams

	sdp := sdpAlias{
		FastRetrieval: true,
	}

	if err := json.Unmarshal(raw, &sdp); err != nil {
		return err
	}

	*s = StartDealParams(sdp)

	return nil
}

type IpldObject struct {
	Cid cid.Cid
	Obj interface{}
}

type ActiveSync struct {
	WorkerID uint64
	Base     *types.TipSet
	Target   *types.TipSet

	Stage  SyncStateStage
	Height abi.ChainEpoch

	Start   time.Time
	End     time.Time
	Message string
}

type SyncState struct {
	ActiveSyncs []ActiveSync

	VMApplied uint64
}

type SyncStateStage int

const (
	StageIdle = SyncStateStage(iota)
	StageHeaders
	StagePersistHeaders
	StageMessages
	StageSyncComplete
	StageSyncErrored
	StageFetchingMessages
)

func (v SyncStateStage) String() string {
	switch v {
	case StageHeaders:
		return "header sync"
	case StagePersistHeaders:
		return "persisting headers"
	case StageMessages:
		return "message sync"
	case StageSyncComplete:
		return "complete"
	case StageSyncErrored:
		return "error"
	case StageFetchingMessages:
		return "fetching messages"
	default:
		return fmt.Sprintf("<unknown: %d>", v)
	}
}

type MpoolChange int

const (
	MpoolAdd MpoolChange = iota
	MpoolRemove
)

type MpoolUpdate struct {
	Type    MpoolChange
	Message *types.SignedMessage
}

type ComputeStateOutput struct {
	Root  cid.Cid
	Trace []*InvocResult
}

type DealCollateralBounds struct {
	Min abi.TokenAmount
	Max abi.TokenAmount
}

type CirculatingSupply struct {
	FilVested      abi.TokenAmount
	FilMined       abi.TokenAmount
	FilBurnt       abi.TokenAmount
	FilLocked      abi.TokenAmount
	FilCirculating abi.TokenAmount
}

type MiningBaseInfo struct {
	MinerPower        types.BigInt
	NetworkPower      types.BigInt
	Sectors           []builtin.SectorInfo
	WorkerKey         address.Address
	SectorSize        abi.SectorSize
	PrevBeaconEntry   types.BeaconEntry
	BeaconEntries     []types.BeaconEntry
	EligibleForMining bool
}

type BlockTemplate struct {
	Miner            address.Address
	Parents          types.TipSetKey
	Ticket           *types.Ticket
	Eproof           *types.ElectionProof
	BeaconValues     []types.BeaconEntry
	Messages         []*types.SignedMessage
	Epoch            abi.ChainEpoch
	Timestamp        uint64
	WinningPoStProof []builtin.PoStProof
}

type DataSize struct {
	PayloadSize int64
	PieceSize   abi.PaddedPieceSize
}

type DataCIDSize struct {
	PayloadSize int64
	PieceSize   abi.PaddedPieceSize
	PieceCID    cid.Cid
}

type CommPRet struct {
	Root cid.Cid
	Size abi.UnpaddedPieceSize
}
type HeadChange struct {
	Type string
	Val  *types.TipSet
}

type MsigProposeResponse int

const (
	MsigApprove MsigProposeResponse = iota
	MsigCancel
)

type Deadline struct {
	PostSubmissions      bitfield.BitField
	DisputableProofCount uint64
}

type Partition struct {
	AllSectors        bitfield.BitField
	FaultySectors     bitfield.BitField
	RecoveringSectors bitfield.BitField
	LiveSectors       bitfield.BitField
	ActiveSectors     bitfield.BitField
}

type Fault struct {
	Miner address.Address
	Epoch abi.ChainEpoch
}

type MessageMatch struct {
	To   address.Address
	From address.Address
}
