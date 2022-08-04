package miner

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	config2 "github.com/filecoin-project/venus-miner/node/config"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/go-state-types/proof"
	"github.com/filecoin-project/venus-miner/lib/journal"
	"github.com/filecoin-project/venus-miner/lib/journal/mockjournal"
	"github.com/filecoin-project/venus-miner/node/modules/slashfilter"
	"github.com/filecoin-project/venus/pkg/chain"
	"github.com/filecoin-project/venus/pkg/config"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	"github.com/filecoin-project/venus/venus-shared/api/chain/v1/mock"
	"github.com/filecoin-project/venus/venus-shared/testutil"
	"github.com/filecoin-project/venus/venus-shared/types"
	"github.com/filecoin-project/venus/venus-shared/utils"
	"github.com/golang/mock/gomock"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/filecoin-project/go-state-types/builtin/v8/miner"
	mock2 "github.com/filecoin-project/venus-miner/miner/mock"
)

type mocMiner struct {
	miner address.Address
	dls   []mockDeadLine
	wpp   *minerWPP
	power big.Int
}

func (mm *mocMiner) deadlines() []types.Deadline {
	var dls []types.Deadline
	for _, dl := range mm.dls {
		dls = append(dls, dl.deadline)
	}
	return dls
}

type mockDeadLine struct {
	deadline   types.Deadline
	partitions []types.Partition
}

type miningMode int

const (
	miningModeNormal = iota
	miningModeChainForkNoBlkOrderChange
	miningModeChainForkBlkOrderChanged
)

type minerTestHelper struct {
	ctx    context.Context
	cancel context.CancelFunc

	networkParam          *types.NetworkParams
	wantedMinedBlockCount int

	miner        *Miner
	api          *mock.MockFullNode
	journal      *mockjournal.MockJournal
	poStProvider *mock2.MockWinningPoStProver

	slashFilter   slashfilter.SlashFilterAPI
	slashFilterDB *gorm.DB

	mockMiners map[address.Address]*mocMiner

	minerAddrs []address.Address

	submitedBlocks              map[abi.ChainEpoch][]*types.BlockHeader
	recodedEventsMinedBlockCids map[cid.Cid]struct{}

	// sense its just a helper, all synchronize control use just one mutex.
	mutx sync.Mutex

	mockChainTipsets []*types.TipSet
	forkedHeight     map[abi.ChainEpoch]struct{}

	mockProofData []byte

	netPower big.Int

	isMinning   bool
	minningMode miningMode
}

func (mt *minerTestHelper) setIsMining(isMinning bool) bool {
	mt.lock()
	defer mt.unlock()
	changed := isMinning != mt.isMinning
	if changed {
		mt.isMinning = isMinning
	}
	return changed
}

func (mt *minerTestHelper) lock() {
	mt.mutx.Lock()
}

func (mt *minerTestHelper) unlock() {
	mt.mutx.Unlock()
}

func (mt *minerTestHelper) chainHead(t *testing.T) *types.TipSet {
	mt.lock()
	defer mt.unlock()

	require.Greater(t, len(mt.mockChainTipsets), 0)

	count := len(mt.mockChainTipsets)

	// todo: should copy deeply?
	head := mt.mockChainTipsets[count-1]
	if head.Height() <= 1 || count <= 2 {
		return head
	}

	if _, ok := mt.forkedHeight[head.Height()]; ok {
		return head
	}

	switch mt.minningMode {
	case miningModeChainForkNoBlkOrderChange, miningModeChainForkBlkOrderChanged:
		var icBlock slashfilter.MinedBlock
		var err error

		if err = mt.slashFilterDB.Model(&slashfilter.MinedBlock{}).
			First(&icBlock, "mine_state = ? and epoch > ?", slashfilter.Mining, head.Height()).
			Order("epoch").Error; err != nil {

			if err == gorm.ErrRecordNotFound {
				t.Logf("since there is no minning blocks,no needs to simulate chain fork....")
			} else {
				t.Errorf("slash filter query failed:%s", err)
			}
			return head
		}

		var newHead *types.TipSet
		blk := buildTestBlock(t, mt.mockChainTipsets[count-2], address.Undef)

		if mt.minningMode == miningModeChainForkBlkOrderChanged {
			newHead, err = types.NewTipSet([]*types.BlockHeader{blk})
		} else {
			newHead, err = types.NewTipSet(append(head.Blocks(), blk))
		}

		if err != nil {
			t.Fatalf("new tipset failed:%s", err)
		}

		t.Logf(`--> simulate chain fork on epoch: %d
  old chain head: %s
  new chain head: %s
`, head.Height(), head.Key().String(), newHead.Key().String())

		mt.mockChainTipsets[count-1] = newHead
		mt.forkedHeight[head.Height()] = struct{}{}
		head = newHead
	case miningModeNormal:
		fallthrough
	default:
	}

	return head
}

func (mt *minerTestHelper) minerBaseInfo(t *testing.T, mAddr address.Address) *types.MiningBaseInfo {
	msCtx, exists := mt.mockMiners[mAddr]
	require.Equal(t, exists, true)
	beaconEntry := &types.BeaconEntry{Round: 1, Data: mt.mockProofData}

	return &types.MiningBaseInfo{
		MinerPower:        msCtx.power,
		NetworkPower:      mt.netPower,
		Sectors:           []proof.ExtendedSectorInfo{},
		WorkerKey:         mAddr,
		SectorSize:        1 << 29,
		PrevBeaconEntry:   *beaconEntry,
		BeaconEntries:     []types.BeaconEntry{*beaconEntry},
		EligibleForMining: true,
	}
}

func (mt *minerTestHelper) clearSlashFilter(t *testing.T) {
	require.NoError(t, mt.slashFilterDB.Delete(&slashfilter.MinedBlock{}, "1 = 1").Error)
}

func (mt *minerTestHelper) syncSubmitBlock(t *testing.T, bm *types.BlockMsg) error {
	t.Logf("submit new block: miner:%s, cid:%s, height:%d", bm.Header.Miner, bm.Header.Cid(), bm.Header.Height)
	mt.mutx.Lock()
	bm.Header.ParentWeight = big.Add(mt.mockChainTipsets[len(mt.mockChainTipsets)-1].ParentWeight(), big.NewInt(1))
	mt.submitedBlocks[bm.Header.Height] = append(mt.submitedBlocks[bm.Header.Height], bm.Header)
	mt.mutx.Unlock()
	return nil
}

func (mt *minerTestHelper) chainTispetWeight(key types.TipSetKey) (big.Int, error) {
	var w = big.NewInt(0)
	mt.mutx.Lock()
	for _, t := range mt.mockChainTipsets {
		if t.Key().Equals(key) {
			w = t.ParentWeight()
			break
		}
	}
	mt.mutx.Unlock()
	return w, nil
}

func (mt *minerTestHelper) chainGetTipsetByHeight(h abi.ChainEpoch) (*types.TipSet, error) {
	mt.lock()
	defer mt.unlock()

	if len(mt.mockChainTipsets) <= int(h) {
		return nil, fmt.Errorf("epoch %d not exists", h)
	}

	return mt.mockChainTipsets[int(h)], nil
}

func (mt *minerTestHelper) setupMockAPI(t *testing.T) {
	fd := mock.NewMockFullNode(gomock.NewController(t))

	mockAny := gomock.Any()

	fd.EXPECT().StateGetNetworkParams(mockAny).AnyTimes().Return(mt.networkParam, nil)

	for mAddr, m := range mt.mockMiners {
		fd.EXPECT().StateMinerDeadlines(mockAny, mAddr, mockAny).AnyTimes().Return(m.deadlines(), nil)

		for idx, dls := range m.dls {
			fd.EXPECT().StateMinerPartitions(mockAny, mAddr, uint64(idx), mockAny).AnyTimes().Return(dls.partitions, nil)

			for _, pt := range dls.partitions {
				require.NoError(t, pt.ActiveSectors.ForEach(func(sid uint64) error {
					fd.EXPECT().StateSectorGetInfo(mockAny, mAddr, abi.SectorNumber(sid), mockAny).
						AnyTimes().Return(buildTestChainSectorInfo(t, sid), nil)
					return nil
				}))
			}
			fd.EXPECT().MinerGetBaseInfo(mockAny, mAddr, mockAny, mockAny).
				AnyTimes().Return(mt.minerBaseInfo(t, mAddr), nil)
		}
	}

	// set miner of first tipset to random
	var randAddress address.Address
	testutil.Provide(t, &randAddress)

	mt.mockChainTipsets = append(mt.mockChainTipsets, buildTestTipset(t, nil, []address.Address{randAddress}, 1))

	fd.EXPECT().ChainHead(mockAny).AnyTimes().AnyTimes().DoAndReturn(
		func(_ context.Context) (*types.TipSet, error) {
			return mt.chainHead(t), nil
		})
	fd.EXPECT().StateNetworkVersion(mockAny, mockAny).AnyTimes().AnyTimes().Return(network.Version16, nil)
	fd.EXPECT().StateGetBeaconEntry(mockAny, mockAny).AnyTimes().Return(
		&types.BeaconEntry{Round: 1, Data: mt.mockProofData}, nil)

	fd.EXPECT().MpoolSelects(mockAny, mockAny, mockAny).AnyTimes().
		DoAndReturn(func(_ context.Context, _ types.TipSetKey, qts []float64) ([][]*types.SignedMessage, error) {
			var msgSlices = make([][]*types.SignedMessage, len(qts))
			for i := 0; i < len(msgSlices); i++ {
				testutil.Provide(t, &msgSlices[i], testutil.WithSliceLen(2))
			}
			return msgSlices, nil
		})
	fd.EXPECT().MinerCreateBlock(mockAny, mockAny).AnyTimes().AnyTimes().
		DoAndReturn(func(_ context.Context, bt *types.BlockTemplate) (*types.BlockMsg, error) {
			next := &types.BlockHeader{
				Miner:         bt.Miner,
				Parents:       bt.Parents.Cids(),
				Ticket:        bt.Ticket,
				ElectionProof: bt.Eproof,
				BeaconEntries: bt.BeaconValues,
				Height:        bt.Epoch,
				Timestamp:     bt.Timestamp,
				WinPoStProof:  bt.WinningPoStProof,
			}
			testutil.Provide(t, &next.ParentStateRoot)
			testutil.Provide(t, &next.ParentMessageReceipts)

			var blsMsgCids, secpkMsgCids []cid.Cid
			for _, msg := range bt.Messages {
				if msg.Signature.Type == crypto.SigTypeBLS {
					blsMsgCids = append(blsMsgCids, msg.Message.Cid())
				} else {
					secpkMsgCids = append(secpkMsgCids, msg.Cid())
				}
			}

			var err error
			if next.Messages, err = chain.ComputeMsgMeta(blockstore.NewBlockstore(datastore.NewMapDatastore()),
				blsMsgCids, secpkMsgCids); err != nil {
				return nil, fmt.Errorf("compute root cid failed:%w", err)
			}

			return &types.BlockMsg{Header: next,
				BlsMessages:   blsMsgCids,
				SecpkMessages: secpkMsgCids}, nil
		})

	// recode submited blocks here.
	fd.EXPECT().SyncSubmitBlock(mockAny, mockAny).AnyTimes().
		DoAndReturn(func(_ context.Context, bm *types.BlockMsg) error {
			return mt.syncSubmitBlock(t, bm)
		})

	fd.EXPECT().ChainTipSetWeight(mockAny, mockAny).AnyTimes().
		DoAndReturn(func(_ context.Context, k types.TipSetKey) (big.Int, error) {
			return mt.chainTispetWeight(k)
		})

	fd.EXPECT().ChainGetTipSetByHeight(mockAny, mockAny, mockAny).AnyTimes().
		DoAndReturn(func(_ context.Context, epoch abi.ChainEpoch, _ types.TipSetKey) (*types.TipSet, error) {
			return mt.chainGetTipsetByHeight(epoch)
		})

	mt.api = fd
}

func (mt *minerTestHelper) setupMiner(t *testing.T) {
	mt.clearSlashFilter(t)

	miner := NewMiner(mt.api, nil, nil, mt.slashFilter, mt.journal)

	miner.signerFunc = func(_ context.Context, _ *config2.GatewayNode) (SignFunc, error) {
		return func(_ context.Context, _ string, _ address.Address, _ []byte, _ types.MsgMeta) (*crypto.Signature, error) {
			return &crypto.Signature{
				Type: crypto.SigTypeBLS,
				Data: mt.mockProofData}, nil
		}, nil
	}
	miner.waitFunc = func(ctx context.Context, baseTime uint64) (func(bool, abi.ChainEpoch, error), abi.ChainEpoch, error) {
		t.Logf("wait func is call...")
		time.Sleep(time.Millisecond)
		return func(bool, abi.ChainEpoch, error) {}, 0, nil
	}

	miner.stop = make(chan struct{})

	for addr, wpp := range mt.mockMiners {
		miner.minerWPPMap[addr] = wpp.wpp
	}
	mt.miner = miner
	mt.submitedBlocks = make(map[abi.ChainEpoch][]*types.BlockHeader)
	mt.forkedHeight = make(map[abi.ChainEpoch]struct{})
}

func (mt *minerTestHelper) currentHeight() abi.ChainEpoch {
	var height abi.ChainEpoch
	mt.lock()
	count := len(mt.mockChainTipsets)
	if count > 0 {
		height = mt.mockChainTipsets[count-1].Height()
	}
	mt.unlock()
	return height
}

func buildMinerTestHelper(t *testing.T, minerStrs []string) *minerTestHelper {
	require.Greater(t, len(minerStrs), 0)
	mtCtx := &minerTestHelper{
		recodedEventsMinedBlockCids: make(map[cid.Cid]struct{}),
		submitedBlocks:              make(map[abi.ChainEpoch][]*types.BlockHeader),
		wantedMinedBlockCount:       2,
		minningMode:                 miningModeNormal,
		forkedHeight:                make(map[abi.ChainEpoch]struct{}),
	}
	//testutil.Provide(t, &mtCtx.mockProofData, testutil.WithSliceLen(32))
	mtCtx.mockProofData = []byte("fake proof data")
	buildMockNetworkParams(t, mtCtx)
	bulildMockJournal(t, mtCtx)
	buildMockPoStProvider(t, mtCtx)
	buildMockMiners(t, mtCtx, minerStrs)
	buildMockSlashfilter(t, mtCtx)
	mtCtx.setupMockAPI(t)
	mtCtx.setupMiner(t)

	mtCtx.ctx, mtCtx.cancel = context.WithCancel(context.TODO())

	return mtCtx
}

// buildMockNetworkCfg can't use something like: venus/venus-shared/networks.Calibration().Network
//	the `networks` pkg depends on `ffi`, we have to declare ours-elf's `buildMockNetworkCfg` here.
func buildMockNetworkCfg(t *testing.T) *config.NetworkParamsConfig {
	return &config.NetworkParamsConfig{
		DevNet:                true,
		NetworkType:           types.NetworkButterfly,
		GenesisNetworkVersion: network.Version15,
		ReplaceProofTypes: []abi.RegisteredSealProof{
			abi.RegisteredSealProof_StackedDrg512MiBV1,
			abi.RegisteredSealProof_StackedDrg32GiBV1,
			abi.RegisteredSealProof_StackedDrg64GiBV1,
		},
		BlockDelay:              2,
		ConsensusMinerMinPower:  2 << 30,
		MinVerifiedDealSize:     1 << 20,
		PreCommitChallengeDelay: abi.ChainEpoch(150),
		ForkUpgradeParam: &config.ForkUpgradeConfig{
			UpgradeBreezeHeight:     -1,
			UpgradeSmokeHeight:      -2,
			UpgradeIgnitionHeight:   -3,
			UpgradeRefuelHeight:     -4,
			UpgradeAssemblyHeight:   -5,
			UpgradeTapeHeight:       -6,
			UpgradeLiftoffHeight:    -7,
			UpgradeKumquatHeight:    -8,
			UpgradeCalicoHeight:     -9,
			UpgradePersianHeight:    -10,
			UpgradeOrangeHeight:     -12,
			UpgradeTrustHeight:      -13,
			UpgradeNorwegianHeight:  -14,
			UpgradeTurboHeight:      -15,
			UpgradeHyperdriveHeight: -16,
			UpgradeChocolateHeight:  -17,
			UpgradeOhSnapHeight:     -18,
			UpgradeSkyrHeight:       50,

			BreezeGasTampingDuration: 120,
			UpgradeClausHeight:       -11,
		},
		DrandSchedule:  map[abi.ChainEpoch]config.DrandEnum{0: 1},
		AddressNetwork: address.Testnet,
	}
}

func buildMockNetworkParams(t *testing.T, mtCtx *minerTestHelper) {
	cfg := buildMockNetworkCfg(t)
	mtCtx.networkParam = &types.NetworkParams{
		NetworkName:             utils.TypeName[cfg.NetworkType],
		BlockDelaySecs:          cfg.BlockDelay,
		ConsensusMinerMinPower:  abi.NewStoragePower(int64(cfg.ConsensusMinerMinPower)),
		SupportedProofTypes:     cfg.ReplaceProofTypes,
		PreCommitChallengeDelay: cfg.PreCommitChallengeDelay,
		ForkUpgradeParams: types.ForkUpgradeParams{
			UpgradeSmokeHeight:       cfg.ForkUpgradeParam.UpgradeSmokeHeight,
			UpgradeBreezeHeight:      cfg.ForkUpgradeParam.UpgradeBreezeHeight,
			UpgradeIgnitionHeight:    cfg.ForkUpgradeParam.UpgradeIgnitionHeight,
			UpgradeLiftoffHeight:     cfg.ForkUpgradeParam.UpgradeLiftoffHeight,
			UpgradeAssemblyHeight:    cfg.ForkUpgradeParam.UpgradeAssemblyHeight,
			UpgradeRefuelHeight:      cfg.ForkUpgradeParam.UpgradeRefuelHeight,
			UpgradeTapeHeight:        cfg.ForkUpgradeParam.UpgradeTapeHeight,
			UpgradeKumquatHeight:     cfg.ForkUpgradeParam.UpgradeKumquatHeight,
			BreezeGasTampingDuration: cfg.ForkUpgradeParam.BreezeGasTampingDuration,
			UpgradeCalicoHeight:      cfg.ForkUpgradeParam.UpgradeCalicoHeight,
			UpgradePersianHeight:     cfg.ForkUpgradeParam.UpgradePersianHeight,
			UpgradeOrangeHeight:      cfg.ForkUpgradeParam.UpgradeOrangeHeight,
			UpgradeClausHeight:       cfg.ForkUpgradeParam.UpgradeClausHeight,
			UpgradeTrustHeight:       cfg.ForkUpgradeParam.UpgradeTrustHeight,
			UpgradeNorwegianHeight:   cfg.ForkUpgradeParam.UpgradeNorwegianHeight,
			UpgradeTurboHeight:       cfg.ForkUpgradeParam.UpgradeTurboHeight,
			UpgradeHyperdriveHeight:  cfg.ForkUpgradeParam.UpgradeHyperdriveHeight,
			UpgradeChocolateHeight:   cfg.ForkUpgradeParam.UpgradeChocolateHeight,
			UpgradeOhSnapHeight:      cfg.ForkUpgradeParam.UpgradeOhSnapHeight,
			UpgradeSkyrHeight:        cfg.ForkUpgradeParam.UpgradeSkyrHeight,
		},
	}
}

func bulildMockJournal(t *testing.T, mtCtx *minerTestHelper) {
	mockCtrl := gomock.NewController(t)
	mockAny := gomock.Any()

	jl := mockjournal.NewMockJournal(mockCtrl)

	jl.EXPECT().RecordEvent(mockAny, mockAny).AnyTimes().DoAndReturn(
		func(evtType journal.EventType, f func() interface{}) {
			if evtType.Event == "block_mined" {
				mtCtx.recodedEventsMinedBlockCids[f().(map[string]interface{})["cid"].(cid.Cid)] = struct{}{}
			}
		})

	jl.EXPECT().RegisterEventType(mockAny, mockAny).AnyTimes().DoAndReturn(
		func(system, event string) journal.EventType {
			return journal.NewEventType(system, event, true, true)
		})

	mtCtx.journal = jl
}

func buildMockPoStProvider(t *testing.T, mtCtx *minerTestHelper) {
	ctler := gomock.NewController(t)
	provider := mock2.NewMockWinningPoStProver(ctler)
	mockAny := gomock.Any()

	provider.EXPECT().ComputeProof(mockAny, mockAny, mockAny, mockAny, mockAny).
		AnyTimes().Return([]builtin.PoStProof{{
		PoStProof:  abi.RegisteredPoStProof_StackedDrgWindow512MiBV1,
		ProofBytes: mtCtx.mockProofData,
	}}, nil)

	mtCtx.poStProvider = provider
}

func buildMockMiners(t *testing.T, mtCtx *minerTestHelper, minerStrs []string) {
	var mms = make(map[address.Address]*mocMiner)

	mtCtx.minerAddrs = make([]address.Address, len(minerStrs))

	minerPower := big.NewInt(1 << 29)

	for idx, ms := range minerStrs {
		a, _ := address.NewFromString(ms)
		mtCtx.minerAddrs[idx] = a

		mms[a] = &mocMiner{
			miner: a,
			wpp: &minerWPP{
				account:  "",
				epp:      mtCtx.poStProvider,
				isMining: true,
			},
			dls:   buildMockDealLine(t),
			power: big.NewInt(1 << 29), // 512M
		}
	}

	mtCtx.netPower = big.Mul(minerPower, big.NewInt(int64(len(minerStrs))))

	mtCtx.mockMiners = mms
}

func buildMockSlashfilter(t *testing.T, mtCtx *minerTestHelper) {
	api, db, err := slashfilter.NewMysqlMock()
	require.NoError(t, err)
	require.NoError(t, db.Callback().Create().Register("on_put_block", func(db *gorm.DB) {
		//if schema := db.Statement.Schema; schema != nil && schema.Name == "MinedBlock" {
		//	block := db.Statement.Dest.(*slashfilter.MinedBlock)
		//	if block.MineState == slashfilter.Mining {
		//	}
		//}
	}))
	mtCtx.slashFilter, mtCtx.slashFilterDB = api, db
}

// makeMockDealLine 构造miner的deadline,
//  目前使用的规则是:
//    构造最大(48个)数量的deadline,
//    每个deadline中包含1(max=3000)个partition,
//    每个partition中2(max=2349)个sector
func buildMockDealLine(t *testing.T) []mockDeadLine {
	const maxDeadlineCount = 48

	dls := make([]mockDeadLine, maxDeadlineCount)

	for idx := 0; idx < maxDeadlineCount; idx++ {
		sectors := bitfield.NewFromSet([]uint64{uint64(idx), uint64(idx + 1)})
		dls[idx] = mockDeadLine{
			deadline: types.Deadline{PostSubmissions: sectors, DisputableProofCount: 0},
			partitions: []types.Partition{
				{AllSectors: sectors, LiveSectors: sectors, ActiveSectors: sectors},
			},
		}
	}

	return dls
}

func buildTestChainSectorInfo(t *testing.T, sid uint64) *miner.SectorOnChainInfo {
	var soi *miner.SectorOnChainInfo
	testutil.Provide(t, &soi, func(*testing.T) abi.SectorNumber {
		return abi.SectorNumber(sid)
	})
	return soi
}

func buildTestTipset(t *testing.T, parent *types.TipSet, miners []address.Address, width int) *types.TipSet {
	if width == 0 {
		return nil
	}
	if width <= 0 {
		width = 1
	} else if width > 10 {
		width = 10
	}

	require.Greater(t, len(miners), 0)

	blocks := make([]*types.BlockHeader, width)
	for idx := 0; idx < width; idx++ {
		blocks[idx] = buildTestBlock(t, parent, miners[(len(miners)-1)%width])
	}

	tipset, err := types.NewTipSet(blocks)
	require.NoError(t, err)

	return tipset
}

func buildTestBlock(t *testing.T, parent *types.TipSet, miner address.Address) *types.BlockHeader {
	var block *types.BlockHeader

	var bData []byte
	testutil.Provide(t, &bData, testutil.WithSliceLen(32))

	if miner == address.Undef {
		testutil.Provide(t, &miner)
	}

	testutil.Provide(t, &block,
		func(t *testing.T) types.VRFPi { return types.VRFPi(bData) },
		func(t *testing.T) types.ElectionProof {
			return types.ElectionProof{WinCount: 1, VRFProof: types.VRFPi(bData)}
		},
		func(t *testing.T) types.BeaconEntry { return types.BeaconEntry{Round: 1, Data: bData} },
		func(t *testing.T) []proof.PoStProof {
			return []proof.PoStProof{{PoStProof: abi.RegisteredPoStProof_StackedDrgWinning512MiBV1, ProofBytes: bData}}
		},
		func(t *testing.T) address.Address { return miner },
		func(t *testing.T) *crypto.Signature { return &crypto.Signature{Type: crypto.SigTypeBLS, Data: bData} },
	)

	block.Timestamp = uint64(time.Now().Unix())
	block.ParentBaseFee = abi.NewTokenAmount(1)

	if parent != nil {
		block.Parents = parent.Cids()
		block.Height = parent.Height() + 1
		block.ParentWeight = big.Add(parent.ParentWeight(), big.NewInt(1))
	} else {
		block.ParentWeight = big.NewInt(1)
		block.Height = 0
	}

	return block
}
