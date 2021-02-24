package apistruct

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/dline"
	stnetwork "github.com/filecoin-project/go-state-types/network"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"

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
		ChainNotify                   func(context.Context) (<-chan []*api.HeadChange, error)                                                            `perm:"read"`
		ChainHead                     func(context.Context) (*types.TipSet, error)                                                                       `perm:"read"`
		ChainGetRandomnessFromTickets func(context.Context, types.TipSetKey, crypto.DomainSeparationTag, abi.ChainEpoch, []byte) (abi.Randomness, error) `perm:"read"`
		ChainGetRandomnessFromBeacon  func(context.Context, types.TipSetKey, crypto.DomainSeparationTag, abi.ChainEpoch, []byte) (abi.Randomness, error) `perm:"read"`
		ChainGetBlock                 func(context.Context, cid.Cid) (*types.BlockHeader, error)                                                         `perm:"read"`
		ChainGetTipSet                func(context.Context, types.TipSetKey) (*types.TipSet, error)                                                      `perm:"read"`
		ChainGetBlockMessages         func(context.Context, cid.Cid) (*api.BlockMessages, error)                                                         `perm:"read"`
		ChainGetParentReceipts        func(context.Context, cid.Cid) ([]*types.MessageReceipt, error)                                                    `perm:"read"`
		ChainGetParentMessages        func(context.Context, cid.Cid) ([]api.Message, error)                                                              `perm:"read"`
		ChainGetTipSetByHeight        func(context.Context, abi.ChainEpoch, types.TipSetKey) (*types.TipSet, error)                                      `perm:"read"`
		ChainReadObj                  func(context.Context, cid.Cid) ([]byte, error)                                                                     `perm:"read"`
		ChainDeleteObj                func(context.Context, cid.Cid) error                                                                               `perm:"admin"`
		ChainHasObj                   func(context.Context, cid.Cid) (bool, error)                                                                       `perm:"read"`
		ChainStatObj                  func(context.Context, cid.Cid, cid.Cid) (api.ObjStat, error)                                                       `perm:"read"`
		ChainSetHead                  func(context.Context, types.TipSetKey) error                                                                       `perm:"admin"`
		ChainGetGenesis               func(context.Context) (*types.TipSet, error)                                                                       `perm:"read"`
		ChainTipSetWeight             func(context.Context, types.TipSetKey) (types.BigInt, error)                                                       `perm:"read"`
		ChainGetNode                  func(ctx context.Context, p string) (*api.IpldObject, error)                                                       `perm:"read"`
		ChainGetMessage               func(context.Context, cid.Cid) (*types.Message, error)                                                             `perm:"read"`
		ChainGetPath                  func(context.Context, types.TipSetKey, types.TipSetKey) ([]*api.HeadChange, error)                                 `perm:"read"`
		ChainExport                   func(context.Context, abi.ChainEpoch, bool, types.TipSetKey) (<-chan []byte, error)                                `perm:"read"`

		BeaconGetEntry func(ctx context.Context, epoch abi.ChainEpoch) (*types.BeaconEntry, error) `perm:"read"`

		GasEstimateGasPremium func(context.Context, uint64, address.Address, int64, types.TipSetKey) (types.BigInt, error)         `perm:"read"`
		GasEstimateGasLimit   func(context.Context, *types.Message, types.TipSetKey) (int64, error)                                `perm:"read"`
		GasEstimateFeeCap     func(context.Context, *types.Message, int64, types.TipSetKey) (types.BigInt, error)                  `perm:"read"`
		GasEstimateMessageGas func(context.Context, *types.Message, *api.MessageSendSpec, types.TipSetKey) (*types.Message, error) `perm:"read"`

		SyncState          func(context.Context) (*api.SyncState, error)                `perm:"read"`
		SyncSubmitBlock    func(ctx context.Context, blk *types.BlockMsg) error         `perm:"write"`
		SyncIncomingBlocks func(ctx context.Context) (<-chan *types.BlockHeader, error) `perm:"read"`
		SyncCheckpoint     func(ctx context.Context, key types.TipSetKey) error         `perm:"admin"`
		SyncMarkBad        func(ctx context.Context, bcid cid.Cid) error                `perm:"admin"`
		SyncUnmarkBad      func(ctx context.Context, bcid cid.Cid) error                `perm:"admin"`
		SyncUnmarkAllBad   func(ctx context.Context) error                              `perm:"admin"`
		SyncCheckBad       func(ctx context.Context, bcid cid.Cid) (string, error)      `perm:"read"`
		SyncValidateTipset func(ctx context.Context, tsk types.TipSetKey) (bool, error) `perm:"read"`

		MpoolGetConfig func(context.Context) (*types.MpoolConfig, error) `perm:"read"`
		MpoolSetConfig func(context.Context, *types.MpoolConfig) error   `perm:"write"`

		MpoolSelect func(context.Context, types.TipSetKey, float64) ([]*types.SignedMessage, error) `perm:"read"`

		MpoolPending func(context.Context, types.TipSetKey) ([]*types.SignedMessage, error) `perm:"read"`
		MpoolClear   func(context.Context, bool) error                                      `perm:"write"`

		MpoolPush          func(context.Context, *types.SignedMessage) (cid.Cid, error) `perm:"write"`
		MpoolPushUntrusted func(context.Context, *types.SignedMessage) (cid.Cid, error) `perm:"write"`

		MpoolPushMessage func(context.Context, *types.Message, *api.MessageSendSpec) (*types.SignedMessage, error) `perm:"sign"`
		MpoolGetNonce    func(context.Context, address.Address) (uint64, error)                                    `perm:"read"`
		MpoolSub         func(context.Context) (<-chan api.MpoolUpdate, error)                                     `perm:"read"`

		MpoolBatchPush          func(ctx context.Context, smsgs []*types.SignedMessage) ([]cid.Cid, error)                                  `perm:"write"`
		MpoolBatchPushUntrusted func(ctx context.Context, smsgs []*types.SignedMessage) ([]cid.Cid, error)                                  `perm:"write"`
		MpoolBatchPushMessage   func(ctx context.Context, msgs []*types.Message, spec *api.MessageSendSpec) ([]*types.SignedMessage, error) `perm:"sign"`

		MinerGetBaseInfo func(context.Context, address.Address, abi.ChainEpoch, types.TipSetKey) (*api.MiningBaseInfo, error) `perm:"read"`
		MinerCreateBlock func(context.Context, *api.BlockTemplate) (*types.BlockMsg, error)                                   `perm:"write"`

		WalletNew             func(context.Context, types.KeyType) (address.Address, error)                        `perm:"write"`
		WalletHas             func(context.Context, address.Address) (bool, error)                                 `perm:"write"`
		WalletList            func(context.Context) ([]address.Address, error)                                     `perm:"write"`
		WalletBalance         func(context.Context, address.Address) (types.BigInt, error)                         `perm:"read"`
		WalletSign            func(context.Context, address.Address, []byte) (*crypto.Signature, error)            `perm:"sign"`
		WalletSignMessage     func(context.Context, address.Address, *types.Message) (*types.SignedMessage, error) `perm:"sign"`
		WalletVerify          func(context.Context, address.Address, []byte, *crypto.Signature) (bool, error)      `perm:"read"`
		WalletDefaultAddress  func(context.Context) (address.Address, error)                                       `perm:"write"`
		WalletSetDefault      func(context.Context, address.Address) error                                         `perm:"admin"`
		WalletExport          func(context.Context, address.Address) (*types.KeyInfo, error)                       `perm:"admin"`
		WalletImport          func(context.Context, *types.KeyInfo) (address.Address, error)                       `perm:"admin"`
		WalletDelete          func(context.Context, address.Address) error                                         `perm:"write"`
		WalletValidateAddress func(context.Context, string) (address.Address, error)                               `perm:"read"`

		StateNetworkName                   func(context.Context) (dtypes.NetworkName, error)                                                                   `perm:"read"`
		StateMinerSectors                  func(context.Context, address.Address, *bitfield.BitField, types.TipSetKey) ([]*miner.SectorOnChainInfo, error)     `perm:"read"`
		StateMinerActiveSectors            func(context.Context, address.Address, types.TipSetKey) ([]*miner.SectorOnChainInfo, error)                         `perm:"read"`
		StateMinerProvingDeadline          func(context.Context, address.Address, types.TipSetKey) (*dline.Info, error)                                        `perm:"read"`
		StateMinerPower                    func(context.Context, address.Address, types.TipSetKey) (*api.MinerPower, error)                                    `perm:"read"`
		StateMinerInfo                     func(context.Context, address.Address, types.TipSetKey) (miner.MinerInfo, error)                                    `perm:"read"`
		StateMinerDeadlines                func(context.Context, address.Address, types.TipSetKey) ([]api.Deadline, error)                                     `perm:"read"`
		StateMinerPartitions               func(ctx context.Context, m address.Address, dlIdx uint64, tsk types.TipSetKey) ([]api.Partition, error)            `perm:"read"`
		StateMinerFaults                   func(context.Context, address.Address, types.TipSetKey) (bitfield.BitField, error)                                  `perm:"read"`
		StateAllMinerFaults                func(context.Context, abi.ChainEpoch, types.TipSetKey) ([]*api.Fault, error)                                        `perm:"read"`
		StateMinerRecoveries               func(context.Context, address.Address, types.TipSetKey) (bitfield.BitField, error)                                  `perm:"read"`
		StateMinerPreCommitDepositForPower func(context.Context, address.Address, miner.SectorPreCommitInfo, types.TipSetKey) (types.BigInt, error)            `perm:"read"`
		StateMinerInitialPledgeCollateral  func(context.Context, address.Address, miner.SectorPreCommitInfo, types.TipSetKey) (types.BigInt, error)            `perm:"read"`
		StateMinerAvailableBalance         func(context.Context, address.Address, types.TipSetKey) (types.BigInt, error)                                       `perm:"read"`
		StateMinerSectorAllocated          func(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (bool, error)                             `perm:"read"`
		StateSectorPreCommitInfo           func(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (miner.SectorPreCommitOnChainInfo, error) `perm:"read"`
		StateSectorGetInfo                 func(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorOnChainInfo, error)         `perm:"read"`
		StateSectorExpiration              func(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorExpiration, error)          `perm:"read"`
		StateSectorPartition               func(context.Context, address.Address, abi.SectorNumber, types.TipSetKey) (*miner.SectorLocation, error)            `perm:"read"`
		StateCall                          func(context.Context, *types.Message, types.TipSetKey) (*api.InvocResult, error)                                    `perm:"read"`
		StateReplay                        func(context.Context, types.TipSetKey, cid.Cid) (*api.InvocResult, error)                                           `perm:"read"`
		StateGetActor                      func(context.Context, address.Address, types.TipSetKey) (*types.Actor, error)                                       `perm:"read"`
		StateReadState                     func(context.Context, address.Address, types.TipSetKey) (*api.ActorState, error)                                    `perm:"read"`
		StateWaitMsg                       func(ctx context.Context, cid cid.Cid, confidence uint64) (*api.MsgLookup, error)                                   `perm:"read"`
		StateWaitMsgLimited                func(context.Context, cid.Cid, uint64, abi.ChainEpoch) (*api.MsgLookup, error)                                      `perm:"read"`
		StateSearchMsg                     func(context.Context, cid.Cid) (*api.MsgLookup, error)                                                              `perm:"read"`
		StateSearchMsgLimited              func(context.Context, cid.Cid, abi.ChainEpoch) (*api.MsgLookup, error)                                              `perm:"read"`
		StateListMiners                    func(context.Context, types.TipSetKey) ([]address.Address, error)                                                   `perm:"read"`
		StateListActors                    func(context.Context, types.TipSetKey) ([]address.Address, error)                                                   `perm:"read"`
		StateLookupID                      func(ctx context.Context, addr address.Address, tsk types.TipSetKey) (address.Address, error)                       `perm:"read"`
		StateAccountKey                    func(context.Context, address.Address, types.TipSetKey) (address.Address, error)                                    `perm:"read"`
		StateChangedActors                 func(context.Context, cid.Cid, cid.Cid) (map[string]types.Actor, error)                                             `perm:"read"`
		StateGetReceipt                    func(context.Context, cid.Cid, types.TipSetKey) (*types.MessageReceipt, error)                                      `perm:"read"`
		StateMinerSectorCount              func(context.Context, address.Address, types.TipSetKey) (api.MinerSectors, error)                                   `perm:"read"`
		StateListMessages                  func(ctx context.Context, match *api.MessageMatch, tsk types.TipSetKey, toht abi.ChainEpoch) ([]cid.Cid, error)     `perm:"read"`
		StateDecodeParams                  func(context.Context, address.Address, abi.MethodNum, []byte, types.TipSetKey) (interface{}, error)                 `perm:"read"`
		StateCompute                       func(context.Context, abi.ChainEpoch, []*types.Message, types.TipSetKey) (*api.ComputeStateOutput, error)           `perm:"read"`
		StateVerifierStatus                func(context.Context, address.Address, types.TipSetKey) (*abi.StoragePower, error)                                  `perm:"read"`
		StateVerifiedClientStatus          func(context.Context, address.Address, types.TipSetKey) (*abi.StoragePower, error)                                  `perm:"read"`
		StateVerifiedRegistryRootKey       func(ctx context.Context, tsk types.TipSetKey) (address.Address, error)                                             `perm:"read"`
		StateDealProviderCollateralBounds  func(context.Context, abi.PaddedPieceSize, bool, types.TipSetKey) (api.DealCollateralBounds, error)                 `perm:"read"`
		StateCirculatingSupply             func(context.Context, types.TipSetKey) (abi.TokenAmount, error)                                                     `perm:"read"`
		StateVMCirculatingSupplyInternal   func(context.Context, types.TipSetKey) (api.CirculatingSupply, error)                                               `perm:"read"`
		StateNetworkVersion                func(context.Context, types.TipSetKey) (stnetwork.Version, error)                                                   `perm:"read"`

		CreateBackup func(ctx context.Context, fpath string) error `perm:"admin"`
	}
}

func (c *FullNodeStruct) StateMinerSectorCount(ctx context.Context, addr address.Address, tsk types.TipSetKey) (api.MinerSectors, error) {
	return c.Internal.StateMinerSectorCount(ctx, addr, tsk)
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

func (c *FullNodeStruct) GasEstimateGasPremium(ctx context.Context, nblocksincl uint64, sender address.Address, gaslimit int64, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.GasEstimateGasPremium(ctx, nblocksincl, sender, gaslimit, tsk)
}

func (c *FullNodeStruct) GasEstimateFeeCap(ctx context.Context, msg *types.Message, maxqueueblks int64, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.GasEstimateFeeCap(ctx, msg, maxqueueblks, tsk)
}

func (c *FullNodeStruct) GasEstimateMessageGas(ctx context.Context, msg *types.Message, spec *api.MessageSendSpec, tsk types.TipSetKey) (*types.Message, error) {
	return c.Internal.GasEstimateMessageGas(ctx, msg, spec, tsk)
}

func (c *FullNodeStruct) GasEstimateGasLimit(ctx context.Context, msg *types.Message, tsk types.TipSetKey) (int64, error) {
	return c.Internal.GasEstimateGasLimit(ctx, msg, tsk)
}

func (c *FullNodeStruct) MpoolGetConfig(ctx context.Context) (*types.MpoolConfig, error) {
	return c.Internal.MpoolGetConfig(ctx)
}

func (c *FullNodeStruct) MpoolSetConfig(ctx context.Context, cfg *types.MpoolConfig) error {
	return c.Internal.MpoolSetConfig(ctx, cfg)
}

func (c *FullNodeStruct) MpoolSelect(ctx context.Context, tsk types.TipSetKey, tq float64) ([]*types.SignedMessage, error) {
	return c.Internal.MpoolSelect(ctx, tsk, tq)
}

func (c *FullNodeStruct) MpoolPending(ctx context.Context, tsk types.TipSetKey) ([]*types.SignedMessage, error) {
	return c.Internal.MpoolPending(ctx, tsk)
}

func (c *FullNodeStruct) MpoolClear(ctx context.Context, local bool) error {
	return c.Internal.MpoolClear(ctx, local)
}

func (c *FullNodeStruct) MpoolPush(ctx context.Context, smsg *types.SignedMessage) (cid.Cid, error) {
	return c.Internal.MpoolPush(ctx, smsg)
}

func (c *FullNodeStruct) MpoolPushUntrusted(ctx context.Context, smsg *types.SignedMessage) (cid.Cid, error) {
	return c.Internal.MpoolPushUntrusted(ctx, smsg)
}

func (c *FullNodeStruct) MpoolPushMessage(ctx context.Context, msg *types.Message, spec *api.MessageSendSpec) (*types.SignedMessage, error) {
	return c.Internal.MpoolPushMessage(ctx, msg, spec)
}

func (c *FullNodeStruct) MpoolBatchPush(ctx context.Context, smsgs []*types.SignedMessage) ([]cid.Cid, error) {
	return c.Internal.MpoolBatchPush(ctx, smsgs)
}

func (c *FullNodeStruct) MpoolBatchPushUntrusted(ctx context.Context, smsgs []*types.SignedMessage) ([]cid.Cid, error) {
	return c.Internal.MpoolBatchPushUntrusted(ctx, smsgs)
}

func (c *FullNodeStruct) MpoolBatchPushMessage(ctx context.Context, msgs []*types.Message, spec *api.MessageSendSpec) ([]*types.SignedMessage, error) {
	return c.Internal.MpoolBatchPushMessage(ctx, msgs, spec)
}

func (c *FullNodeStruct) MpoolSub(ctx context.Context) (<-chan api.MpoolUpdate, error) {
	return c.Internal.MpoolSub(ctx)
}

func (c *FullNodeStruct) MinerGetBaseInfo(ctx context.Context, maddr address.Address, epoch abi.ChainEpoch, tsk types.TipSetKey) (*api.MiningBaseInfo, error) {
	return c.Internal.MinerGetBaseInfo(ctx, maddr, epoch, tsk)
}

func (c *FullNodeStruct) MinerCreateBlock(ctx context.Context, bt *api.BlockTemplate) (*types.BlockMsg, error) {
	return c.Internal.MinerCreateBlock(ctx, bt)
}

func (c *FullNodeStruct) ChainHead(ctx context.Context) (*types.TipSet, error) {
	return c.Internal.ChainHead(ctx)
}

func (c *FullNodeStruct) ChainGetRandomnessFromTickets(ctx context.Context, tsk types.TipSetKey, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error) {
	return c.Internal.ChainGetRandomnessFromTickets(ctx, tsk, personalization, randEpoch, entropy)
}

func (c *FullNodeStruct) ChainGetRandomnessFromBeacon(ctx context.Context, tsk types.TipSetKey, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error) {
	return c.Internal.ChainGetRandomnessFromBeacon(ctx, tsk, personalization, randEpoch, entropy)
}

func (c *FullNodeStruct) ChainGetTipSetByHeight(ctx context.Context, h abi.ChainEpoch, tsk types.TipSetKey) (*types.TipSet, error) {
	return c.Internal.ChainGetTipSetByHeight(ctx, h, tsk)
}

func (c *FullNodeStruct) WalletNew(ctx context.Context, typ types.KeyType) (address.Address, error) {
	return c.Internal.WalletNew(ctx, typ)
}

func (c *FullNodeStruct) WalletHas(ctx context.Context, addr address.Address) (bool, error) {
	return c.Internal.WalletHas(ctx, addr)
}

func (c *FullNodeStruct) WalletList(ctx context.Context) ([]address.Address, error) {
	return c.Internal.WalletList(ctx)
}

func (c *FullNodeStruct) WalletBalance(ctx context.Context, a address.Address) (types.BigInt, error) {
	return c.Internal.WalletBalance(ctx, a)
}

func (c *FullNodeStruct) WalletSign(ctx context.Context, k address.Address, msg []byte) (*crypto.Signature, error) {
	return c.Internal.WalletSign(ctx, k, msg)
}

func (c *FullNodeStruct) WalletSignMessage(ctx context.Context, k address.Address, msg *types.Message) (*types.SignedMessage, error) {
	return c.Internal.WalletSignMessage(ctx, k, msg)
}

func (c *FullNodeStruct) WalletVerify(ctx context.Context, k address.Address, msg []byte, sig *crypto.Signature) (bool, error) {
	return c.Internal.WalletVerify(ctx, k, msg, sig)
}

func (c *FullNodeStruct) WalletDefaultAddress(ctx context.Context) (address.Address, error) {
	return c.Internal.WalletDefaultAddress(ctx)
}

func (c *FullNodeStruct) WalletSetDefault(ctx context.Context, a address.Address) error {
	return c.Internal.WalletSetDefault(ctx, a)
}

func (c *FullNodeStruct) WalletExport(ctx context.Context, a address.Address) (*types.KeyInfo, error) {
	return c.Internal.WalletExport(ctx, a)
}

func (c *FullNodeStruct) WalletImport(ctx context.Context, ki *types.KeyInfo) (address.Address, error) {
	return c.Internal.WalletImport(ctx, ki)
}

func (c *FullNodeStruct) WalletDelete(ctx context.Context, addr address.Address) error {
	return c.Internal.WalletDelete(ctx, addr)
}

func (c *FullNodeStruct) WalletValidateAddress(ctx context.Context, str string) (address.Address, error) {
	return c.Internal.WalletValidateAddress(ctx, str)
}

func (c *FullNodeStruct) MpoolGetNonce(ctx context.Context, addr address.Address) (uint64, error) {
	return c.Internal.MpoolGetNonce(ctx, addr)
}

func (c *FullNodeStruct) ChainGetBlock(ctx context.Context, b cid.Cid) (*types.BlockHeader, error) {
	return c.Internal.ChainGetBlock(ctx, b)
}

func (c *FullNodeStruct) ChainGetTipSet(ctx context.Context, key types.TipSetKey) (*types.TipSet, error) {
	return c.Internal.ChainGetTipSet(ctx, key)
}

func (c *FullNodeStruct) ChainGetBlockMessages(ctx context.Context, b cid.Cid) (*api.BlockMessages, error) {
	return c.Internal.ChainGetBlockMessages(ctx, b)
}

func (c *FullNodeStruct) ChainGetParentReceipts(ctx context.Context, b cid.Cid) ([]*types.MessageReceipt, error) {
	return c.Internal.ChainGetParentReceipts(ctx, b)
}

func (c *FullNodeStruct) ChainGetParentMessages(ctx context.Context, b cid.Cid) ([]api.Message, error) {
	return c.Internal.ChainGetParentMessages(ctx, b)
}

func (c *FullNodeStruct) ChainNotify(ctx context.Context) (<-chan []*api.HeadChange, error) {
	return c.Internal.ChainNotify(ctx)
}

func (c *FullNodeStruct) ChainReadObj(ctx context.Context, obj cid.Cid) ([]byte, error) {
	return c.Internal.ChainReadObj(ctx, obj)
}

func (c *FullNodeStruct) ChainDeleteObj(ctx context.Context, obj cid.Cid) error {
	return c.Internal.ChainDeleteObj(ctx, obj)
}

func (c *FullNodeStruct) ChainHasObj(ctx context.Context, o cid.Cid) (bool, error) {
	return c.Internal.ChainHasObj(ctx, o)
}

func (c *FullNodeStruct) ChainStatObj(ctx context.Context, obj, base cid.Cid) (api.ObjStat, error) {
	return c.Internal.ChainStatObj(ctx, obj, base)
}

func (c *FullNodeStruct) ChainSetHead(ctx context.Context, tsk types.TipSetKey) error {
	return c.Internal.ChainSetHead(ctx, tsk)
}

func (c *FullNodeStruct) ChainGetGenesis(ctx context.Context) (*types.TipSet, error) {
	return c.Internal.ChainGetGenesis(ctx)
}

func (c *FullNodeStruct) ChainTipSetWeight(ctx context.Context, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.ChainTipSetWeight(ctx, tsk)
}

func (c *FullNodeStruct) ChainGetNode(ctx context.Context, p string) (*api.IpldObject, error) {
	return c.Internal.ChainGetNode(ctx, p)
}

func (c *FullNodeStruct) ChainGetMessage(ctx context.Context, mc cid.Cid) (*types.Message, error) {
	return c.Internal.ChainGetMessage(ctx, mc)
}

func (c *FullNodeStruct) ChainGetPath(ctx context.Context, from types.TipSetKey, to types.TipSetKey) ([]*api.HeadChange, error) {
	return c.Internal.ChainGetPath(ctx, from, to)
}

func (c *FullNodeStruct) ChainExport(ctx context.Context, nroots abi.ChainEpoch, iom bool, tsk types.TipSetKey) (<-chan []byte, error) {
	return c.Internal.ChainExport(ctx, nroots, iom, tsk)
}

func (c *FullNodeStruct) BeaconGetEntry(ctx context.Context, epoch abi.ChainEpoch) (*types.BeaconEntry, error) {
	return c.Internal.BeaconGetEntry(ctx, epoch)
}

func (c *FullNodeStruct) SyncState(ctx context.Context) (*api.SyncState, error) {
	return c.Internal.SyncState(ctx)
}

func (c *FullNodeStruct) SyncSubmitBlock(ctx context.Context, blk *types.BlockMsg) error {
	return c.Internal.SyncSubmitBlock(ctx, blk)
}

func (c *FullNodeStruct) SyncIncomingBlocks(ctx context.Context) (<-chan *types.BlockHeader, error) {
	return c.Internal.SyncIncomingBlocks(ctx)
}

func (c *FullNodeStruct) SyncCheckpoint(ctx context.Context, tsk types.TipSetKey) error {
	return c.Internal.SyncCheckpoint(ctx, tsk)
}

func (c *FullNodeStruct) SyncMarkBad(ctx context.Context, bcid cid.Cid) error {
	return c.Internal.SyncMarkBad(ctx, bcid)
}

func (c *FullNodeStruct) SyncUnmarkBad(ctx context.Context, bcid cid.Cid) error {
	return c.Internal.SyncUnmarkBad(ctx, bcid)
}

func (c *FullNodeStruct) SyncUnmarkAllBad(ctx context.Context) error {
	return c.Internal.SyncUnmarkAllBad(ctx)
}

func (c *FullNodeStruct) SyncCheckBad(ctx context.Context, bcid cid.Cid) (string, error) {
	return c.Internal.SyncCheckBad(ctx, bcid)
}

func (c *FullNodeStruct) SyncValidateTipset(ctx context.Context, tsk types.TipSetKey) (bool, error) {
	return c.Internal.SyncValidateTipset(ctx, tsk)
}

func (c *FullNodeStruct) StateNetworkName(ctx context.Context) (dtypes.NetworkName, error) {
	return c.Internal.StateNetworkName(ctx)
}

func (c *FullNodeStruct) StateMinerSectors(ctx context.Context, addr address.Address, sectorNos *bitfield.BitField, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error) {
	return c.Internal.StateMinerSectors(ctx, addr, sectorNos, tsk)
}

func (c *FullNodeStruct) StateMinerActiveSectors(ctx context.Context, addr address.Address, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error) {
	return c.Internal.StateMinerActiveSectors(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateMinerProvingDeadline(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*dline.Info, error) {
	return c.Internal.StateMinerProvingDeadline(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateMinerPower(ctx context.Context, a address.Address, tsk types.TipSetKey) (*api.MinerPower, error) {
	return c.Internal.StateMinerPower(ctx, a, tsk)
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

func (c *FullNodeStruct) StateMinerFaults(ctx context.Context, actor address.Address, tsk types.TipSetKey) (bitfield.BitField, error) {
	return c.Internal.StateMinerFaults(ctx, actor, tsk)
}

func (c *FullNodeStruct) StateAllMinerFaults(ctx context.Context, cutoff abi.ChainEpoch, endTsk types.TipSetKey) ([]*api.Fault, error) {
	return c.Internal.StateAllMinerFaults(ctx, cutoff, endTsk)
}

func (c *FullNodeStruct) StateMinerRecoveries(ctx context.Context, actor address.Address, tsk types.TipSetKey) (bitfield.BitField, error) {
	return c.Internal.StateMinerRecoveries(ctx, actor, tsk)
}

func (c *FullNodeStruct) StateMinerPreCommitDepositForPower(ctx context.Context, maddr address.Address, pci miner.SectorPreCommitInfo, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.StateMinerPreCommitDepositForPower(ctx, maddr, pci, tsk)
}

func (c *FullNodeStruct) StateMinerInitialPledgeCollateral(ctx context.Context, maddr address.Address, pci miner.SectorPreCommitInfo, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.StateMinerInitialPledgeCollateral(ctx, maddr, pci, tsk)
}

func (c *FullNodeStruct) StateMinerAvailableBalance(ctx context.Context, maddr address.Address, tsk types.TipSetKey) (types.BigInt, error) {
	return c.Internal.StateMinerAvailableBalance(ctx, maddr, tsk)
}

func (c *FullNodeStruct) StateMinerSectorAllocated(ctx context.Context, maddr address.Address, s abi.SectorNumber, tsk types.TipSetKey) (bool, error) {
	return c.Internal.StateMinerSectorAllocated(ctx, maddr, s, tsk)
}

func (c *FullNodeStruct) StateSectorPreCommitInfo(ctx context.Context, maddr address.Address, n abi.SectorNumber, tsk types.TipSetKey) (miner.SectorPreCommitOnChainInfo, error) {
	return c.Internal.StateSectorPreCommitInfo(ctx, maddr, n, tsk)
}

func (c *FullNodeStruct) StateSectorGetInfo(ctx context.Context, maddr address.Address, n abi.SectorNumber, tsk types.TipSetKey) (*miner.SectorOnChainInfo, error) {
	return c.Internal.StateSectorGetInfo(ctx, maddr, n, tsk)
}

func (c *FullNodeStruct) StateSectorExpiration(ctx context.Context, maddr address.Address, n abi.SectorNumber, tsk types.TipSetKey) (*miner.SectorExpiration, error) {
	return c.Internal.StateSectorExpiration(ctx, maddr, n, tsk)
}

func (c *FullNodeStruct) StateSectorPartition(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok types.TipSetKey) (*miner.SectorLocation, error) {
	return c.Internal.StateSectorPartition(ctx, maddr, sectorNumber, tok)
}

func (c *FullNodeStruct) StateCall(ctx context.Context, msg *types.Message, tsk types.TipSetKey) (*api.InvocResult, error) {
	return c.Internal.StateCall(ctx, msg, tsk)
}

func (c *FullNodeStruct) StateReplay(ctx context.Context, tsk types.TipSetKey, mc cid.Cid) (*api.InvocResult, error) {
	return c.Internal.StateReplay(ctx, tsk, mc)
}

func (c *FullNodeStruct) StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	return c.Internal.StateGetActor(ctx, actor, tsk)
}

func (c *FullNodeStruct) StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ActorState, error) {
	return c.Internal.StateReadState(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateWaitMsg(ctx context.Context, msgc cid.Cid, confidence uint64) (*api.MsgLookup, error) {
	return c.Internal.StateWaitMsg(ctx, msgc, confidence)
}

func (c *FullNodeStruct) StateWaitMsgLimited(ctx context.Context, msgc cid.Cid, confidence uint64, limit abi.ChainEpoch) (*api.MsgLookup, error) {
	return c.Internal.StateWaitMsgLimited(ctx, msgc, confidence, limit)
}

func (c *FullNodeStruct) StateSearchMsg(ctx context.Context, msgc cid.Cid) (*api.MsgLookup, error) {
	return c.Internal.StateSearchMsg(ctx, msgc)
}

func (c *FullNodeStruct) StateSearchMsgLimited(ctx context.Context, msgc cid.Cid, limit abi.ChainEpoch) (*api.MsgLookup, error) {
	return c.Internal.StateSearchMsgLimited(ctx, msgc, limit)
}

func (c *FullNodeStruct) StateListMiners(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error) {
	return c.Internal.StateListMiners(ctx, tsk)
}

func (c *FullNodeStruct) StateListActors(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error) {
	return c.Internal.StateListActors(ctx, tsk)
}

func (c *FullNodeStruct) StateLookupID(ctx context.Context, addr address.Address, tsk types.TipSetKey) (address.Address, error) {
	return c.Internal.StateLookupID(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateAccountKey(ctx context.Context, addr address.Address, tsk types.TipSetKey) (address.Address, error) {
	return c.Internal.StateAccountKey(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateChangedActors(ctx context.Context, olnstate cid.Cid, newstate cid.Cid) (map[string]types.Actor, error) {
	return c.Internal.StateChangedActors(ctx, olnstate, newstate)
}

func (c *FullNodeStruct) StateGetReceipt(ctx context.Context, msg cid.Cid, tsk types.TipSetKey) (*types.MessageReceipt, error) {
	return c.Internal.StateGetReceipt(ctx, msg, tsk)
}

func (c *FullNodeStruct) StateListMessages(ctx context.Context, match *api.MessageMatch, tsk types.TipSetKey, toht abi.ChainEpoch) ([]cid.Cid, error) {
	return c.Internal.StateListMessages(ctx, match, tsk, toht)
}

func (c *FullNodeStruct) StateDecodeParams(ctx context.Context, toAddr address.Address, method abi.MethodNum, params []byte, tsk types.TipSetKey) (interface{}, error) {
	return c.Internal.StateDecodeParams(ctx, toAddr, method, params, tsk)
}

func (c *FullNodeStruct) StateCompute(ctx context.Context, height abi.ChainEpoch, msgs []*types.Message, tsk types.TipSetKey) (*api.ComputeStateOutput, error) {
	return c.Internal.StateCompute(ctx, height, msgs, tsk)
}

func (c *FullNodeStruct) StateVerifierStatus(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*abi.StoragePower, error) {
	return c.Internal.StateVerifierStatus(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateVerifiedClientStatus(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*abi.StoragePower, error) {
	return c.Internal.StateVerifiedClientStatus(ctx, addr, tsk)
}

func (c *FullNodeStruct) StateVerifiedRegistryRootKey(ctx context.Context, tsk types.TipSetKey) (address.Address, error) {
	return c.Internal.StateVerifiedRegistryRootKey(ctx, tsk)
}

func (c *FullNodeStruct) StateDealProviderCollateralBounds(ctx context.Context, size abi.PaddedPieceSize, verified bool, tsk types.TipSetKey) (api.DealCollateralBounds, error) {
	return c.Internal.StateDealProviderCollateralBounds(ctx, size, verified, tsk)
}

func (c *FullNodeStruct) StateCirculatingSupply(ctx context.Context, tsk types.TipSetKey) (abi.TokenAmount, error) {
	return c.Internal.StateCirculatingSupply(ctx, tsk)
}

func (c *FullNodeStruct) StateVMCirculatingSupplyInternal(ctx context.Context, tsk types.TipSetKey) (api.CirculatingSupply, error) {
	return c.Internal.StateVMCirculatingSupplyInternal(ctx, tsk)
}

func (c *FullNodeStruct) StateNetworkVersion(ctx context.Context, tsk types.TipSetKey) (stnetwork.Version, error) {
	return c.Internal.StateNetworkVersion(ctx, tsk)
}

func (c *FullNodeStruct) CreateBackup(ctx context.Context, fpath string) error {
	return c.Internal.CreateBackup(ctx, fpath)
}

var _ api.Common = &CommonStruct{}
var _ api.FullNode = &FullNodeStruct{}

type MinerStruct struct {
	CommonStruct

	Internal struct {
		AddAddress    func(dtypes.MinerInfo) error                 `perm:"write"`
		RemoveAddress func(address.Address) error                  `perm:"write"`
		ListAddress   func() ([]dtypes.MinerInfo, error)           `perm:"read"`
		SetDefault    func(address.Address) error                  `perm:"write"`
		Default       func() (address.Address, error)              `perm:"read"`
		Start         func(context.Context, address.Address) error `perm:"write"`
		Stop          func(context.Context, address.Address) error `perm:"write"`
	}
}

func (s *MinerStruct) AddAddress(miner dtypes.MinerInfo) error {
	return s.Internal.AddAddress(miner)
}

func (s *MinerStruct) RemoveAddress(addr address.Address) error {
	return s.Internal.RemoveAddress(addr)
}

func (s *MinerStruct) ListAddress() ([]dtypes.MinerInfo, error) {
	return s.Internal.ListAddress()
}

func (s *MinerStruct) SetDefault(addr address.Address) error {
	return s.Internal.SetDefault(addr)
}

func (s *MinerStruct) Default() (address.Address, error) {
	return s.Internal.Default()
}

func (s *MinerStruct) Start(ctx context.Context, addr address.Address) error {
	return s.Internal.Start(ctx, addr)
}

func (s *MinerStruct) Stop(ctx context.Context, addr address.Address) error {
	return s.Internal.Stop(ctx, addr)
}

var _ api.MinerAPI = &MinerStruct{}
