// stm: #unit
package miner

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	leveldb "github.com/ipfs/go-ds-leveldb"
	logging "github.com/ipfs/go-log/v2"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	miner "github.com/filecoin-project/go-state-types/builtin/v9/miner"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/ipfs-force-community/sophon-miner/build"
	"github.com/ipfs-force-community/sophon-miner/lib/journal"
	"github.com/ipfs-force-community/sophon-miner/lib/journal/mockjournal"
	"github.com/ipfs-force-community/sophon-miner/node/config"
	minerecorder "github.com/ipfs-force-community/sophon-miner/node/modules/mine-recorder"
	"github.com/ipfs-force-community/sophon-miner/node/modules/miner-manager/mock"
	"github.com/ipfs-force-community/sophon-miner/node/modules/slashfilter"
	types2 "github.com/ipfs-force-community/sophon-miner/types"

	"github.com/filecoin-project/venus/fixtures/networks"
	config2 "github.com/filecoin-project/venus/pkg/config"
	"github.com/filecoin-project/venus/pkg/constants"
	mockAPI "github.com/filecoin-project/venus/venus-shared/api/chain/v1/mock"
	"github.com/filecoin-project/venus/venus-shared/types"
	"github.com/filecoin-project/venus/venus-shared/utils"
)

func TestSuccessMinerBlocks(t *testing.T) {
	// stm: @VENUSMINER_MINERWPP_COMPUTE_PROOF_001, @VENUSMINER_MULTIMINER_GET_BEST_MINING_CANDIDATE_001, @VENUSMINER_MULTIMINER_SYNC_STATUS_001
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// stm: @VENUSMINER_MULTIMINER_NEW_001
	miner, chain, _ := setMiner(ctx, t, 4)
	chain.keepChainGoing(ctx)
	// stm: @VENUSMINER_MULTIMINER_START_001, @VENUSMINER_NODECONFIGGATEWAYDEF_DIAL_ARGS_001, @VENUSMINER_NODECONFIGGATEWAYDEF_AUTH_HEADER_001
	assert.Nil(t, miner.Start(ctx))
	defer func() {
		// stm: @VENUSMINER_MULTIMINER_STOP_001
		assert.Nil(t, miner.Stop(ctx))
	}()

	for {
		select {
		case blk := <-chain.newBlkCh:
			if blk.Height > 3 {
				return
			}
		case <-time.After(time.Duration(chain.params.BlockDelaySecs) * time.Second * 10):
			t.Errorf("wait too long for miner new block")
			return
		}
	}
}

func TestCountWinner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	miner, chain, _ := setMiner(ctx, t, 4)
	chain.keepChainGoing(ctx)

	assert.Nil(t, miner.Start(ctx))
	defer func() {
		assert.Nil(t, miner.Stop(ctx))
	}()

	minedBlocks := map[string]struct {
		count  int64
		blkCid cid.Cid
	}{}
	var once sync.Once
	for {
		select {
		case blk := <-chain.newBlkCh:
			if blk.Height > 5 {
				once.Do(func() {
					addrs := chain.pcController.listAddress()
					// stm: @VENUSMINER_MINERMGR_COUNT_WINNERS_001
					winners, err := miner.CountWinners(ctx, addrs, 0, 4)
					assert.Nil(t, err)
					for _, minerSt := range winners {
						for _, sWinfo := range minerSt.WinEpochList {
							//block maybe not mined due to many reasons(low performance machine)
							blk, ok := minedBlocks[minerSt.Miner.String()+strconv.Itoa(int(sWinfo.Epoch))]
							if ok {
								assert.Equal(t, blk.count, sWinfo.WinCount, "block id %s epoch %d", blk.blkCid, sWinfo.Epoch)
							}
						}
					}
				})
				return
			} else if blk.Height >= 1 && blk.Height <= 5 {
				minedBlocks[blk.Miner.String()+strconv.Itoa(int(blk.Height))] = struct {
					count  int64
					blkCid cid.Cid
				}{count: blk.ElectionProof.WinCount, blkCid: blk.Cid()}
			}
		case <-time.After(time.Duration(chain.params.BlockDelaySecs) * time.Second * 10):
			t.Errorf("wait too long for miner new block")
			return
		}
	}
}

func TestCountWinnerSignFailed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	miner, chain, _ := setMiner(ctx, t, 4)
	chain.keepChainGoing(ctx)

	assert.Nil(t, miner.Start(ctx))
	defer func() {
		assert.Nil(t, miner.Stop(ctx))
	}()

	var once sync.Once
	for {
		select {
		case blk := <-chain.newBlkCh:
			if blk.Height > 5 {
				once.Do(func() {
					// mock sign failed
					miner.signerFunc = func(ctx context.Context, node *config.GatewayNode) SignFunc {
						return func(ctx context.Context, signer address.Address, accounts []string, toSign []byte, meta types.MsgMeta) (*crypto.Signature, error) {
							return nil, fmt.Errorf("%v %w", "sign failed:", types2.WalletSignError)
						}
					}

					addrs := chain.pcController.listAddress()
					winners, err := miner.CountWinners(ctx, addrs, 0, 4)
					assert.Nil(t, err)
					for _, minerSt := range winners {
						for _, sWinfo := range minerSt.WinEpochList {
							assert.Equal(t, "failed to compute VRF: sign failed: 2", sWinfo.Msg)
						}
					}
				})
				return
			}
		case <-time.After(time.Duration(chain.params.BlockDelaySecs) * time.Second * 10):
			t.Errorf("wait too long for miner new block")
			return
		}
	}
}

func TestSuccessNullRoundMinerBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	miner, chain, _ := setMiner(ctx, t, 4)

	stop := make(chan struct{})
	chain.setAfterEvent(func(round abi.ChainEpoch) {
		if round == 3 {
			t.Log("start null round")
			chain.pcController.clearPower()
		}

		if round == 5 {
			chain.pcController.randPower()
			chain.restartFromNullRound()
			t.Log("recover from null round")
		}
	})

	assert.Nil(t, miner.Start(ctx))
	defer func() {
		assert.Nil(t, miner.Stop(ctx))
	}()

	select {
	case <-stop:
	case <-time.After(time.Duration(chain.params.BlockDelaySecs) * time.Second * 8):
	}

	chain.logMatcher.match("{\"number of wins\": 0, \"total miner\": 4}")
}

func TestForkChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	miner, chain, _ := setMiner(ctx, t, 4)
	chain.keepChainGoing(ctx)

	var once sync.Once
	stop := make(chan struct{})
	chain.baseInfoHook = func(address2 address.Address, round abi.ChainEpoch) {
		if round == 5 {
			once.Do(func() {
				chain.mockFork(3, true)
			})
		} else if round > 8 {
			select {
			case stop <- struct{}{}:
			default:
			}
		}
	}

	assert.Nil(t, miner.Start(ctx))
	defer func() {
		assert.Nil(t, miner.Stop(ctx))
	}()

	select {
	case <-stop:
	case <-time.After(time.Duration(chain.params.BlockDelaySecs) * time.Second * 10):
		t.Errorf("wait too long for miner new block")
	}

	assert.True(t, len(chain.dropBlks) > 0)
	chain.logMatcher.match("chain may be forked")
}

func TestParentGridFail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	miner, chain, _ := setMiner(ctx, t, 3)
	chain.keepChainGoing(ctx)

	var once sync.Once
	stop := make(chan struct{})
	chain.baseInfoHook = func(address2 address.Address, round abi.ChainEpoch) {
		if round == 5 {
			once.Do(func() {
				chain.mockFork(3, false)
			})
		} else if round > 8 {
			select {
			case stop <- struct{}{}:
			default:
			}
		}
	}

	assert.Nil(t, miner.Start(ctx))
	defer func() {
		assert.Nil(t, miner.Stop(ctx))
	}()

	select {
	case <-stop:
	case <-time.After(time.Duration(chain.params.BlockDelaySecs) * time.Second * 10):
		t.Errorf("wait too long for miner new block")
	}

	assert.True(t, len(chain.dropBlks) > 0)
	chain.logMatcher.match("SLASH FILTER ERROR: produced block would trigger 'parent-grinding fault'")
}

func TestSameHeight(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	miner, chain, _ := setMiner(ctx, t, 4)
	chain.keepChainGoing(ctx)

	var once sync.Once
	chain.blockEndHook = func(round abi.ChainEpoch) {
		if round > 5 {
			once.Do(func() {
				chain.fallBack(3)
			})
		}
	}

	assert.Nil(t, miner.Start(ctx))
	defer func() {
		assert.Nil(t, miner.Stop(ctx))
	}()

	<-time.After(time.Duration(chain.params.BlockDelaySecs) * time.Second * 7)
	chain.logMatcher.match("created a block at the same height as another block we've created")
}

func TestSuccessUpdateBaseWhenBaseExtend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	miner, chain, _ := setMiner(ctx, t, 2)
	chain.keepChainGoing(ctx)

	var once sync.Once
	stop := make(chan struct{})
	chain.baseInfoHook = func(address2 address.Address, round abi.ChainEpoch) {
		if round == 3 {
			once.Do(func() {
				chain.replaceWithWeightHead()
			})
		} else if round > 5 {
			select {
			case stop <- struct{}{}:
			default:
			}
		}
	}

	assert.Nil(t, miner.Start(ctx))
	defer func() {
		assert.Nil(t, miner.Stop(ctx))
	}()

	select {
	case <-stop:
	case <-time.After(time.Duration(chain.params.BlockDelaySecs) * time.Second * 10):
		t.Errorf("wait too long for miner new block")
	}

	chain.logMatcher.match("there are better bases here")
}

func TestManualStartAndStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	miner, chain, _ := setMiner(ctx, t, 4)
	chain.keepChainGoing(ctx)

	chain.setAfterEvent(func(round abi.ChainEpoch) {
		if round == 3 {
			addrs := chain.pcController.listAddress()
			err := miner.ManualStop(ctx, []address.Address{addrs[0]})
			assert.Nil(t, err)
			// After stop mining, it will no longer appear in minerWPPMap
			_, ok := miner.minerWPPMap[addrs[0]]
			assert.False(t, ok)
			err = miner.ManualStop(ctx, addrs)
			for _, addr := range addrs {
				_, ok := miner.minerWPPMap[addr]
				assert.False(t, ok)
			}
			assert.Nil(t, err)
		}

		if round == 5 {
			minerList := chain.changeMiner(3)
			// stm: @VENUSMINER_MINERWPP_NEW_WINNING_POST_PROVER_001, @VENUSMINER_MINERMGR_UPDATE_ADDRESS_001, @VENUSMINER_NODE_MODULES_AUTH_MANAGER_UPDATE_001
			_, err := miner.UpdateAddress(ctx, 0, 100)
			assert.Nil(t, err)
			assert.Equal(t, 3, len(miner.minerWPPMap))

			addrs := chain.pcController.listAddress()
			// stm: @VENUSMINER_MINERMGR_MANUAL_STOP_001
			err = miner.ManualStop(ctx, addrs)
			assert.Nil(t, err)
			assert.Equal(t, 0, len(miner.minerWPPMap))

			// stm: @VENUSMINER_MINERMGR_MANUAL_START_001
			err = miner.ManualStart(ctx, []address.Address{minerList[0]})
			assert.Nil(t, err)
			_, ok := miner.minerWPPMap[minerList[0]]
			assert.True(t, ok)

			err = miner.ManualStart(ctx, minerList)
			assert.Nil(t, err)
			assert.Equal(t, len(minerList), len(miner.minerWPPMap))
			for addr := range miner.minerWPPMap {
				assert.True(t, miner.minerManager.IsOpenMining(ctx, addr))
			}
			chain.replaceWithWeightHead()
		}
	})

	assert.Nil(t, miner.Start(ctx))
	defer func() {
		assert.Nil(t, miner.Stop(ctx))
	}()

	for {
		select {
		case blk := <-chain.newBlkCh:
			if blk.Height >= 6 {
				return
			}
		case <-time.After(time.Duration(chain.params.BlockDelaySecs) * time.Second * 10):
			t.Errorf("wait too long for miner new block")
			return
		}
	}
}

func TestMinerStateStartStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	miner, chain, _ := setMiner(ctx, t, 4)

	// stm: @VENUSMINER_MINERMGR_LIST_ADDRESS_001, @VENUSMINER_NODE_MODULES_AUTH_MANAGER_LIST_001
	mInfos, err := miner.ListAddress(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(mInfos), 4)

	stop := make(chan struct{})
	chain.baseInfoHook = func(address2 address.Address, round abi.ChainEpoch) {
		select {
		case stop <- struct{}{}:
		default:
		}
	}

	assert.Nil(t, miner.Start(ctx))
	defer func() {
		assert.Nil(t, miner.Stop(ctx))
	}()

	<-stop
	addrs := chain.pcController.listAddress()
	// stm: @VENUSMINER_MINERMGR_STATES_FOR_MINING_001
	states, err := miner.StatesForMining(ctx, addrs)
	assert.Nil(t, err)
	for _, st := range states {
		assert.True(t, st.IsMining)
	}

	pickToStop := addrs[0]
	err = miner.ManualStop(ctx, []address.Address{pickToStop})
	assert.Nil(t, err)

	st, err := miner.StatesForMining(ctx, []address.Address{pickToStop})
	assert.Nil(t, err)
	assert.Equal(t, pickToStop, st[0].Addr)
	assert.False(t, st[0].IsMining)

	err = miner.ManualStart(ctx, []address.Address{pickToStop})
	assert.Nil(t, err)

	st, err = miner.StatesForMining(ctx, []address.Address{pickToStop})
	assert.Nil(t, err)
	assert.Equal(t, pickToStop, st[0].Addr)
	assert.True(t, st[0].IsMining)
}

func buildMinerBaseInfo(t *testing.T, mAddr address.Address, minerPower, networkPoser int64) *types.MiningBaseInfo {
	return &types.MiningBaseInfo{
		MinerPower:   abi.NewStoragePower(minerPower),
		NetworkPower: abi.NewStoragePower(networkPoser),
		Sectors:      nil,
		WorkerKey:    mAddr,
		SectorSize:   1 << 29,
		PrevBeaconEntry: types.BeaconEntry{
			Round: 1,
			Data:  []byte("just mock"),
		},
		BeaconEntries:     nil,
		EligibleForMining: true,
	}
}

func setMiner(ctx context.Context, t *testing.T, minerCount int) (*Miner, *mockChain, *mockAPI.MockFullNode) {
	logging.SetDebugLogging()
	mockAny := gomock.Any()

	db, err := leveldb.NewDatastore(t.TempDir()+"/leveldb", nil)
	assert.NoError(t, err)
	minerecorder.SetDatastore(db)

	p := networks.Net2k().Network
	p.BlockDelay = 10
	genesisTime := uint64(build.Clock.Now().Unix())
	chain := newMockChain(ctx, t, p)
	chain.processEvent(ctx)
	api := mockAPI.NewMockFullNode(gomock.NewController(t))
	api.EXPECT().StateGetNetworkParams(gomock.Any()).AnyTimes().Return(chain.params, nil)
	api.EXPECT().StateSectorGetInfo(mockAny, mockAny, mockAny, mockAny).AnyTimes().Return(&miner.SectorOnChainInfo{}, nil)

	managerAPI := mock.NewMockMinerManageAPI(gomock.NewController(t))
	slasher, _, _ := slashfilter.NewMysqlMock()
	jl := mockjournal.NewMockJournal(gomock.NewController(t))
	jl.EXPECT().RecordEvent(mockAny, mockAny).AnyTimes().DoAndReturn(func(arg0 journal.EventType, arg1 func() interface{}) {
		if chain.blockEndHook != nil {
			chain.blockEndHook(arg1().(map[string]interface{})["epoch"].(abi.ChainEpoch))
		}
	})

	jl.EXPECT().RegisterEventType(mockAny, mockAny).AnyTimes().DoAndReturn(
		func(system, event string) journal.EventType {
			return journal.NewEventType(system, event, true, true)
		})

	cfg := config.DefaultMinerConfig()
	cfg.PropagationDelaySecs = 3

	miner, err := NewMiner(context.Background(), api, cfg, managerAPI, slasher, jl)
	assert.NoError(t, err)

	miner.signerFunc = func(ctx context.Context, node *config.GatewayNode) SignFunc {
		return func(ctx context.Context, signer address.Address, accounts []string, toSign []byte, meta types.MsgMeta) (*crypto.Signature, error) {

			return &crypto.Signature{
				Type: crypto.SigTypeBLS,
				Data: []byte{1, 2, 3},
			}, nil
		}
	}

	genesisMiner := chain.createMiner()
	for i := 0; i < minerCount; i++ {
		addr := chain.createMiner()
		chain.pcController.setPower(addr, 0)
	}
	chain.pcController.randPower()
	api.EXPECT().SyncSubmitBlock(mockAny, mockAny).AnyTimes().
		DoAndReturn(func(_ context.Context, bm *types.BlockMsg) error {
			chain.setBlock(bm.Header)
			chain.newBlkCh <- bm.Header
			return nil
		})

	managerAPI.EXPECT().List(mockAny).AnyTimes().DoAndReturn(func(arg0 context.Context) (map[address.Address]*types2.MinerInfo, error) {
		return chain.pcController.listMinerInfo(), nil
	})
	managerAPI.EXPECT().Update(mockAny, mockAny, mockAny).AnyTimes().DoAndReturn(func(arg0 context.Context, arg1, arg2 int64) (map[address.Address]*types2.MinerInfo, error) {
		return chain.pcController.listMinerInfo(), nil
	})
	managerAPI.EXPECT().OpenMining(mockAny, mockAny).AnyTimes().DoAndReturn(func(arg0 context.Context, arg1 address.Address) (*types2.MinerInfo, error) {
		return chain.pcController.openMining(arg1), nil
	})
	managerAPI.EXPECT().CloseMining(mockAny, mockAny).AnyTimes().DoAndReturn(func(arg0 context.Context, arg1 address.Address) error {
		return chain.pcController.closeMining(arg1)
	})
	managerAPI.EXPECT().IsOpenMining(mockAny, mockAny).AnyTimes().DoAndReturn(func(arg0 context.Context, arg1 address.Address) bool {
		return chain.pcController.isMining(arg1)
	})

	api.EXPECT().MinerGetBaseInfo(mockAny, mockAny, mockAny, mockAny).AnyTimes().DoAndReturn(func(arg0 context.Context, arg1 address.Address, arg2 abi.ChainEpoch, arg3 types.TipSetKey) (*types.MiningBaseInfo, error) {
		if chain.baseInfoHook != nil {
			chain.baseInfoHook(arg1, arg2)
		}
		return buildMinerBaseInfo(t, arg1, chain.pcController.getPower(arg1), chain.pcController.totalPower), nil
	})

	api.EXPECT().StateNetworkVersion(mockAny, mockAny).AnyTimes().AnyTimes().Return(network.Version16, nil)
	api.EXPECT().StateMinerInfo(mockAny, mockAny, mockAny).AnyTimes().Return(types.MinerInfo{}, nil)
	api.EXPECT().StateMinerDeadlines(mockAny, mockAny, mockAny).AnyTimes().Return([]types.Deadline{{}}, nil)
	api.EXPECT().StateMinerPartitions(mockAny, mockAny, mockAny, mockAny).AnyTimes().Return([]types.Partition{{
		ActiveSectors: bitfield.NewFromSet([]uint64{uint64(1), uint64(2)}),
	}}, nil)
	api.EXPECT().StateGetBeaconEntry(mockAny, mockAny).AnyTimes().Return(nil, nil)
	api.EXPECT().MpoolSelects(mockAny, mockAny, mockAny).AnyTimes().
		DoAndReturn(func(_ context.Context, _ types.TipSetKey, qts []float64) ([][]*types.SignedMessage, error) {
			var msgSlices = make([][]*types.SignedMessage, len(qts))
			for i := 0; i < len(msgSlices); i++ {
				msgSlices[i] = nil
			}
			return msgSlices, nil
		})
	api.EXPECT().SyncState(mockAny).AnyTimes().DoAndReturn(func(arg0 context.Context) (*types.SyncState, error) {
		return &types.SyncState{
			ActiveSyncs: nil,
			VMApplied:   0,
		}, nil
	})
	genesisBlk := &types.BlockHeader{
		Miner:                 genesisMiner,
		ParentWeight:          types.NewInt(0),
		ParentStateRoot:       chain.genRand.Cid(),
		ParentMessageReceipts: chain.genRand.Cid(),
		Messages:              chain.genRand.Cid(),
		Parents:               nil,
		Ticket:                &types.Ticket{VRFProof: []byte("ticket genesis")},
		ElectionProof:         nil,
		BeaconEntries:         nil,
		Height:                0,
		Timestamp:             genesisTime,
		WinPoStProof:          nil,
	}
	chain.setBlock(genesisBlk)

	api.EXPECT().ChainHead(mockAny).AnyTimes().DoAndReturn(func(arg0 context.Context) (*types.TipSet, error) {
		return chain.getHead(), nil
	})

	api.EXPECT().ChainTipSetWeight(mockAny, mockAny).AnyTimes().
		DoAndReturn(func(_ context.Context, k types.TipSetKey) (big.Int, error) {
			return big.Add(chain.getTipset(k).ParentWeight(), big.NewInt(int64(len(k.Cids())))), nil
		})

	api.EXPECT().ChainGetTipSetByHeight(mockAny, mockAny, mockAny).AnyTimes().
		DoAndReturn(func(_ context.Context, epoch abi.ChainEpoch, _ types.TipSetKey) (*types.TipSet, error) {
			return chain.getTipsetByHeight(epoch), nil
		})
	api.EXPECT().MinerCreateBlock(mockAny, mockAny).AnyTimes().AnyTimes().
		DoAndReturn(func(_ context.Context, bt *types.BlockTemplate) (*types.BlockMsg, error) {
			next := &types.BlockHeader{
				Miner: bt.Miner,
				// The parent-weight should be the base, temporarily increase the weight of the epoch
				ParentWeight:          types.NewInt(100 + uint64(len(bt.Parents.Cids())) + uint64(bt.Epoch)*5 + chain.additionWeight),
				ParentStateRoot:       chain.genRand.Cid(),
				ParentMessageReceipts: chain.genRand.Cid(),
				Messages:              chain.genRand.Cid(),
				Parents:               bt.Parents.Cids(),
				Ticket:                bt.Ticket,
				ElectionProof:         bt.Eproof,
				BeaconEntries:         bt.BeaconValues,
				Height:                bt.Epoch,
				Timestamp:             bt.Timestamp,
				WinPoStProof:          bt.WinningPoStProof,
			}
			return &types.BlockMsg{Header: next}, nil
		})

	return miner, chain, api
}

type minerPowerInfo struct {
	power      int64
	openMining bool
}

type powerController struct {
	totalPower int64
	minerPoser map[address.Address]*minerPowerInfo
	lk         sync.Mutex
}

func newPowerController(addrs []address.Address) *powerController {
	v := map[address.Address]*minerPowerInfo{}
	for _, addr := range addrs {
		v[addr] = &minerPowerInfo{0, true}
	}
	return &powerController{
		totalPower: 1000000,
		minerPoser: v,
		lk:         sync.Mutex{},
	}
}

func (pc *powerController) listMinerInfo() map[address.Address]*types2.MinerInfo {
	pc.lk.Lock()
	defer pc.lk.Unlock()

	infos := make(map[address.Address]*types2.MinerInfo)
	for addr, power := range pc.minerPoser {
		infos[addr] = &types2.MinerInfo{
			Addr:       addr,
			Id:         "test",
			Name:       "test",
			OpenMining: power.openMining,
		}
	}
	return infos
}

func (pc *powerController) openMining(addr address.Address) *types2.MinerInfo {
	pc.lk.Lock()
	defer pc.lk.Unlock()

	if power, ok := pc.minerPoser[addr]; ok {
		pc.minerPoser[addr].openMining = true
		return &types2.MinerInfo{
			Addr:       addr,
			Id:         "test",
			Name:       "test",
			OpenMining: power.openMining,
		}
	}
	return nil
}

func (pc *powerController) closeMining(addr address.Address) error {
	pc.lk.Lock()
	defer pc.lk.Unlock()

	if _, ok := pc.minerPoser[addr]; ok {
		pc.minerPoser[addr].openMining = false
	}
	return nil
}

func (pc *powerController) isMining(addr address.Address) bool {
	pc.lk.Lock()
	defer pc.lk.Unlock()

	if _, ok := pc.minerPoser[addr]; ok {
		return pc.minerPoser[addr].openMining
	}
	return false
}

func (pc *powerController) listAddress() []address.Address {
	pc.lk.Lock()
	defer pc.lk.Unlock()

	var addrs []address.Address
	for addr := range pc.minerPoser {
		addrs = append(addrs, addr)
	}
	return addrs
}

func (pc *powerController) randPower() {
	pc.lk.Lock()
	defer pc.lk.Unlock()
	t := pc.totalPower
	count := 1
	for addr := range pc.minerPoser {
		if count == len(pc.minerPoser) {
			pc.minerPoser[addr].power = t
			continue
		}

		p := rand.Int63n(t)
		pc.minerPoser[addr].power = p
		t = t - p
		count++
	}
}

func (pc *powerController) getPower(addr address.Address) int64 {
	pc.lk.Lock()
	defer pc.lk.Unlock()

	if _, ok := pc.minerPoser[addr]; ok {
		return pc.minerPoser[addr].power
	}

	return 0
}

func (pc *powerController) setPower(addr address.Address, power int64) {
	pc.lk.Lock()
	defer pc.lk.Unlock()

	if _, ok := pc.minerPoser[addr]; !ok {
		pc.minerPoser[addr] = &minerPowerInfo{0, true}
	}
	pc.minerPoser[addr].power = power
}

func (pc *powerController) clearPower() {
	pc.lk.Lock()
	defer pc.lk.Unlock()
	for addr := range pc.minerPoser {
		pc.minerPoser[addr].power = 0
	}
}

func (pc *powerController) clearMiner() {
	pc.lk.Lock()
	defer pc.lk.Unlock()
	for addr := range pc.minerPoser {
		delete(pc.minerPoser, addr)
	}
}

type mockChain struct {
	params         *types.NetworkParams
	genRand        *randGen
	head           *types.TipSet
	blockStore     map[cid.Cid]*types.BlockHeader
	key2Ts         map[types.TipSetKey]*types.TipSet
	lk             sync.Mutex
	actoridLk      sync.Mutex
	actorid        uint64
	t              *testing.T
	dropBlks       map[cid.Cid]*types.BlockHeader
	newBlkCh       chan *types.BlockHeader
	pcController   *powerController
	callLk         sync.Mutex
	eventCall      []func(epoch abi.ChainEpoch)
	additionWeight uint64

	baseInfoHook func(address2 address.Address, epoch abi.ChainEpoch)

	blockEndHook func(bt abi.ChainEpoch)
	logMatcher   *logMatcher
}

func newMockChain(ctx context.Context, t *testing.T, cfg config2.NetworkParamsConfig) *mockChain {
	return &mockChain{
		params:       getMockNetworkParams(t, cfg),
		genRand:      new(randGen),
		dropBlks:     map[cid.Cid]*types.BlockHeader{},
		lk:           sync.Mutex{},
		actoridLk:    sync.Mutex{},
		blockStore:   map[cid.Cid]*types.BlockHeader{},
		key2Ts:       map[types.TipSetKey]*types.TipSet{},
		t:            t,
		newBlkCh:     make(chan *types.BlockHeader, 1000), //give enough bug to store block
		pcController: newPowerController(nil),
		logMatcher:   newLogMatcher(ctx, t),
	}
}

func (m *mockChain) restartFromNullRound() {
	m.lk.Lock()
	defer m.lk.Unlock()
	newBlkCopy := *m.head.Blocks()[0]
	newBlkCopy.Miner = m.createMiner()

	blks := m.head.Blocks()[:]
	m.blockStore[newBlkCopy.Cid()] = &newBlkCopy
	blks = append(blks, &newBlkCopy)
	ts, err := types.NewTipSet(blks)
	m.key2Ts[ts.Key()] = ts
	assert.Nil(m.t, err)
	m.head = ts
}

func (m *mockChain) createMiner() address.Address {
	m.actoridLk.Lock()
	defer m.actoridLk.Unlock()
	m.actorid++
	addr, err := address.NewIDAddress(m.actorid)
	assert.Nil(m.t, err)
	return addr
}

func (m *mockChain) setBlock(blk *types.BlockHeader) {
	ts, err := types.NewTipSet([]*types.BlockHeader{blk})
	assert.Nil(m.t, err)
	m.setHead(ts)
}

func (m *mockChain) setHead(fts *types.TipSet) {
	m.lk.Lock()
	defer m.lk.Unlock()

	for _, blk := range fts.Blocks() {
		m.blockStore[blk.Cid()] = blk
	}

	if m.head == nil {
		m.head = fts
		return
	}

	if fts.ParentWeight().LessThan(m.head.ParentWeight()) {
		//low weight drop
		for _, blk := range fts.Blocks() {
			m.dropBlks[blk.Cid()] = blk
		}
		return
	}

	//merge blk
	blks := fts.Blocks()[:]
	seen := map[cid.Cid]struct{}{}
	for _, blk := range blks {
		seen[blk.Cid()] = struct{}{}
	}
	if m.head == nil || (m.head.Height() == fts.Height() && m.head.Parents().Equals(fts.Parents()) && m.head.ParentWeight().Equals((fts.ParentWeight()))) {
		for _, blk := range m.head.Blocks() {
			if _, ok := seen[blk.Cid()]; !ok {
				blks = append(blks, blk)
				seen[blk.Cid()] = struct{}{}
			}
		}
	}

	newHead, err := types.NewTipSet(blks)
	assert.Nil(m.t, err)
	if err != nil {
		panic(err)
	}

	//reorg
	revert, apply, err := ReorgOps(func(ctx context.Context, key types.TipSetKey) (*types.TipSet, error) {
		var blks []*types.BlockHeader
		for _, blkCid := range key.Cids() {
			blk, ok := m.blockStore[blkCid]
			assert.True(m.t, ok)
			blks = append(blks, blk)
		}
		return types.NewTipSet(blks)
	}, m.head, newHead)
	assert.Nil(m.t, err)

	for _, ts := range revert {
		for _, blk := range ts.Blocks() {
			m.dropBlks[blk.Cid()] = blk
		}
	}
	for _, ts := range apply {
		for _, blk := range ts.Blocks() {
			delete(m.dropBlks, blk.Cid())
		}
	}
	m.head = newHead
	m.key2Ts[newHead.Key()] = newHead
}

func (m *mockChain) getHead() *types.TipSet {
	m.lk.Lock()
	defer m.lk.Unlock()
	return m.head
}

func (m *mockChain) getTipset(tsk types.TipSetKey) *types.TipSet {
	m.lk.Lock()
	defer m.lk.Unlock()
	return m.key2Ts[tsk]
}

func (m *mockChain) getTipsetByHeight(h abi.ChainEpoch) *types.TipSet {
	m.lk.Lock()
	defer m.lk.Unlock()
	ts := m.head
	for {
		if ts.Height() <= h {
			return ts
		}
		ts = m.key2Ts[ts.Parents()]
	}
}

func (m *mockChain) mockFork(lbHeight abi.ChainEpoch, changeTicket bool) {
	m.lk.Lock()
	m.additionWeight += 10
	toHeight := m.head.Height() - lbHeight
	rand.Seed(build.Clock.Now().Unix())
	var revertTs []*types.TipSet
	ts := m.head
	for {
		if ts.Height() <= toHeight {
			break
		}
		revertTs = append(revertTs, ts)
		ts = m.key2Ts[ts.Parents()]
	}

	parent := m.key2Ts[revertTs[len(revertTs)-1].Parents()]
	for i := len(revertTs) - 1; i >= 0; i-- {
		var blks []*types.BlockHeader
		for _, blk := range revertTs[i].Blocks() {
			blkCopy := *blk
			blkCopy.Miner = m.createMiner()
			blkCopy.Parents = parent.Cids()
			blkCopy.ParentWeight = types.NewInt(100 + uint64(len(ts.Parents().Cids())) + uint64(blk.Height)*5 + m.additionWeight)
			if changeTicket {
				ticket := make([]byte, 32)
				rand.Read(ticket)
				blkCopy.Ticket = &types.Ticket{VRFProof: ticket}
			}
			m.blockStore[blkCopy.Cid()] = &blkCopy
			blks = append(blks, &blkCopy)
		}
		var err error
		parent, err = types.NewTipSet(blks)
		assert.Nil(m.t, err)
	}
	m.lk.Unlock()

	m.setHead(parent)
	assert.Equal(m.t, parent, m.getHead()) //confirm mock fork chain success
}

func (m *mockChain) fallBack(lbHeight abi.ChainEpoch) {
	head := m.getHead()
	ts := m.getTipsetByHeight(head.Height() - lbHeight)
	rand.Seed(build.Clock.Now().Unix())
	var blks []*types.BlockHeader
	for _, blk := range ts.Blocks() {
		blkCopy := *blk
		blkCopy.Miner = m.createMiner()
		blkCopy.ParentWeight = big.Add(head.ParentWeight(), big.NewInt(1000))
		ticket := make([]byte, 32)
		rand.Read(ticket)
		blkCopy.Ticket = &types.Ticket{VRFProof: ticket}
		m.blockStore[blkCopy.Cid()] = &blkCopy
		blks = append(blks, &blkCopy)
	}
	newHead, err := types.NewTipSet(blks)
	assert.Nil(m.t, err)
	m.setHead(newHead)
	assert.Equal(m.t, newHead, m.getHead()) //confirm mock fork chain success
}

func (m *mockChain) nextBlock() {
	// when the head is unchanged for two consecutive rounds, add a round of tipset(one block)
	head := m.getHead()
	nullRounds := (uint64(build.Clock.Now().Unix()) - head.MinTimestamp()) / m.params.BlockDelaySecs

	if nullRounds < 2 {
		return
	}

	epoch := head.Height() + abi.ChainEpoch(nullRounds)
	ticket := make([]byte, 32)
	rand.Read(ticket)
	next := &types.BlockHeader{
		Miner:                 m.createMiner(),
		ParentWeight:          types.NewInt(100 + uint64(len(head.Cids())) + uint64(epoch)*5 + m.additionWeight),
		ParentStateRoot:       m.genRand.Cid(),
		ParentMessageReceipts: m.genRand.Cid(),
		Messages:              m.genRand.Cid(),
		Parents:               head.Cids(),
		Ticket:                &types.Ticket{VRFProof: ticket},
		BeaconEntries: []types.BeaconEntry{
			{
				Round: nullRounds,
				Data:  ticket,
			},
		},
		Height:    epoch,
		Timestamp: head.MinTimestamp() + nullRounds*m.params.BlockDelaySecs,
	}

	m.setBlock(next)
	m.t.Log("insert new block, height", next.Height, "weight", next.ParentWeight, "time", next.Timestamp)
}

func (m *mockChain) replaceWithWeightHead() {
	head := m.getHead()
	blkCopy := *(head.At(0))
	blkCopy.Miner = m.createMiner()
	m.lk.Lock()
	m.blockStore[blkCopy.Cid()] = &blkCopy
	m.lk.Unlock()

	var blks []*types.BlockHeader
	blks = append(blks, head.Blocks()...)
	blks = append(blks, &blkCopy)

	curTs, err := types.NewTipSet(blks)
	assert.Nil(m.t, err)

	m.setHead(curTs)
	assert.Equal(m.t, curTs, m.getHead()) //confirm mock fork chain success
}

func (m *mockChain) setAfterEvent(fn func(round abi.ChainEpoch)) {
	m.callLk.Lock()
	defer m.callLk.Unlock()

	m.eventCall = append(m.eventCall, fn)
}

func (m *mockChain) changeMiner(nMiner int) []address.Address {
	m.callLk.Lock()
	defer m.callLk.Unlock()

	var addrs []address.Address
	m.pcController.clearMiner()
	for i := 0; i < nMiner; i++ {
		addr := m.createMiner()
		addrs = append(addrs, addr)
		m.pcController.setPower(addr, 0)
	}
	m.pcController.randPower()
	return addrs
}

func (m *mockChain) keepChainGoing(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Duration(m.params.BlockDelaySecs) * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				m.nextBlock()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (m *mockChain) processEvent(ctx context.Context) {
	round := abi.ChainEpoch(1)
	go func() {
		t := time.NewTicker(time.Duration(m.params.BlockDelaySecs) * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				m.callLk.Lock()
				callCopy := m.eventCall[:]
				m.callLk.Unlock()
				for _, fn := range callCopy {
					fn(round)
				}
				round++
			case <-ctx.Done():
				return
			}
		}
	}()
}

type randGen struct {
	count int64
	lk    sync.Mutex
}

type logMatcher struct {
	lk      sync.Mutex
	logMsgs []string
	t       *testing.T
}

func newLogMatcher(ctx context.Context, t *testing.T) *logMatcher {
	matcher := &logMatcher{t: t}
	reader := logging.NewPipeReader(logging.PipeFormat(logging.PlaintextOutput))
	sc := bufio.NewScanner(reader)

	go func() {
		for sc.Scan() {
			msg := sc.Text() // GET the line string
			matcher.lk.Lock()
			matcher.logMsgs = append(matcher.logMsgs, msg)
			matcher.lk.Unlock()
			select {
			case <-ctx.Done():
				reader.Close()
				return
			default:
			}
		}
	}()
	return matcher
}

func (m *logMatcher) match(expectMsg string) {
	m.lk.Lock()
	defer m.lk.Unlock()
	for _, logMsg := range m.logMsgs {
		if strings.Contains(logMsg, expectMsg) {
			return
		}
	}
	m.t.Errorf("not found match log %s", expectMsg)
}

func (r *randGen) Cid() cid.Cid {
	r.lk.Lock()
	defer r.lk.Unlock()
	r.count++
	rand.Seed(r.count)
	data := make([]byte, 32)
	rand.Read(data[:])
	c, _ := abi.CidBuilder.Sum(data)
	return c
}

func getMockNetworkParams(t *testing.T, cfg config2.NetworkParamsConfig) *types.NetworkParams {
	return &types.NetworkParams{
		NetworkName:             utils.NetworkTypeToNetworkName(cfg.NetworkType),
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

func init() {
	constants.InsecurePoStValidation = true
}
