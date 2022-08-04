package miner

import (
	"context"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/node/modules/slashfilter"

	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/venus/venus-shared/types"
)

func TestMine(t *testing.T) {
	mtHelper := buildMinerTestHelper(t, []string{"f01010", "f01011"})

	t.Run("test mine common", mtHelper.testMineCommon)
	t.Run("test mine on chain forked, block order not changed", mtHelper.testMineForkKeepBlksOrder)
	t.Run("test mine on chain forked, block order changed", mtHelper.testMineForkDisOrderBlks)
	t.Run("test count winner", mtHelper.testCountWinner)
	t.Run("test mining state", mtHelper.testMiningState)
}

func (mt *minerTestHelper) testMineCommon(t *testing.T) {
	mt.beginMineTillWantedBlockCount(t, miningModeNormal, 5)
	mt.miner.mine(mt.ctx)
	mt.checkAfterMine(t)
}

func (mt *minerTestHelper) testMineForkKeepBlksOrder(t *testing.T) {
	mt.beginMineTillWantedBlockCount(t, miningModeChainForkNoBlkOrderChange, 10)
	mt.miner.mine(mt.ctx)
	mt.checkAfterMine(t)
}

func (mt *minerTestHelper) testMineForkDisOrderBlks(t *testing.T) {
	mt.beginMineTillWantedBlockCount(t, miningModeChainForkBlkOrderChanged, 5)
	mt.miner.mine(mt.ctx)
	mt.checkAfterMine(t)
}

func (mt *minerTestHelper) testMiningState(t *testing.T) {
	states, err := mt.miner.StatesForMining(mt.ctx, mt.minerAddrs)
	require.NoError(t, err)
	for _, s := range states {
		require.Equal(t, s.IsMining, true)
		require.Equal(t, len(s.Err), 0)
	}
}

func (mt *minerTestHelper) testCountWinner(t *testing.T) {
	ctx := context.TODO()
	winner, err := mt.miner.CountWinners(ctx, mt.minerAddrs, 2, 3)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(winner), 2)
}

func (mt *minerTestHelper) beginMineTillWantedBlockCount(t *testing.T, miningMode miningMode, hiCount int64) {
	if !mt.setIsMining(true) {
		t.Logf("have been minning...")
		return
	}
	mt.minningMode = miningMode

	mt.setupMiner(t)

	var tryMineNewTipset = func() {
		mt.lock()
		defer mt.unlock()
		head := mt.mockChainTipsets[len(mt.mockChainTipsets)-1]

		blks, isok := mt.submitedBlocks[head.Height()+1]
		// if no new tipset generated more than `BlockDelaySecs`, generate random tipset..
		if !isok {
			duration := time.Since(time.Unix(int64(head.MinTimestamp()), 0)).Seconds()
			if duration < 10 {
				return
			}
			blks = []*types.BlockHeader{buildTestBlock(t, head, address.Undef)}
			t.Logf("create rondom tipset, at heigth:%d", 1+head.Height())
		}
		newTipset, err := types.NewTipSet(blks)
		require.NoError(t, err)
		mt.mockChainTipsets = append(mt.mockChainTipsets, newTipset)

		t.Logf("--> chain head now is :%d, cids:%s\n", newTipset.Height(), newTipset.Key())
	}

	go func() {
		defer mt.setIsMining(false)
		ticker := time.NewTicker(time.Second * time.Duration(mt.networkParam.BlockDelaySecs))
		startHeight := mt.currentHeight()
	exitFor:
		//loop for generating new chain head with previous generated as its parents, every `blockDelay` seconds,
		//as well as returned `messages` by MpoolSelects
		for int64(mt.currentHeight()-startHeight) < hiCount {
			tryMineNewTipset()
			<-ticker.C
			switch mt.minningMode {
			case miningModeChainForkBlkOrderChanged:
				if len(mt.forkedHeight) >= 3 {
					break exitFor
				}
			default:
				if len(mt.submitedBlocks) >= mt.wantedMinedBlockCount {
					break exitFor
				}
			}
		}
		ticker.Stop()
		require.NoError(t, mt.miner.Stop(mt.ctx))
	}()
}

func (mt *minerTestHelper) checkAfterMine(t *testing.T) {
	var normalMineChecker = func() {

		require.GreaterOrEqual(t, len(mt.submitedBlocks), mt.wantedMinedBlockCount,
			"its impossible, that we cant mined %d blocks", mt.wantedMinedBlockCount)

		t.Logf("check after mine, submited epoch count:%d", len(mt.submitedBlocks))

		for h, blks := range mt.submitedBlocks {
			var minedBlks []*slashfilter.MinedBlock
			require.NoError(t, mt.slashFilterDB.Model(&slashfilter.MinedBlock{}).Find(&minedBlks, "epoch = ?", h).Error)
			require.Equal(t, len(blks), len(minedBlks))

			blkMap := make(map[string]*types.BlockHeader)

			for _, blk := range blks {
				blkMap[blk.Cid().String()] = blk
			}

			for _, mblk := range minedBlks {
				blk, isok := blkMap[mblk.Cid]
				require.Equal(t, isok, true)
				require.Equal(t, blk.Miner.String(), mblk.Miner)
			}
		}
	}

	var forkMineChecker = func() {
		var forkedBlks []*slashfilter.MinedBlock

		require.NoError(t, mt.slashFilterDB.Model((*slashfilter.MinedBlock)(nil)).
			Find(&forkedBlks, "mine_state in (?, ?)", slashfilter.ChainForked, slashfilter.Error).
			Order("epoch desc").Error)

		log.Infof("check after mine, mined but forked or errored block count:%d", len(forkedBlks))

		for _, blk := range forkedBlks {
			t.Logf("mined block but forked: height:%d, miner:%s, mine_state:%s\n", blk.Epoch,
				blk.Miner, blk.MineState.String())
		}

		require.Greater(t, len(forkedBlks), 0)
	}

	switch mt.minningMode {
	case miningModeChainForkBlkOrderChanged:
		forkMineChecker()
	case miningModeNormal, miningModeChainForkNoBlkOrderChange:
		fallthrough
	default:
		normalMineChecker()
	}

}
