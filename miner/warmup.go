package miner

import (
	"context"
	"crypto/rand"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/venus-miner/chain"
	"math"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"

	proof2 "github.com/filecoin-project/specs-actors/v2/actors/runtime/proof"

	"github.com/filecoin-project/venus-miner/chain/types"
)

func (m *Miner) winPoStWarmup(ctx context.Context) error {
	for addr, wpp := range m.minerWPPMap {
		tAddr := addr
		epp := wpp.epp
		go func() {
			err := m.winPostWarmupForMiner(ctx, tAddr, epp)
			if err != nil {
				log.Infow("mining warm up failed", "miner", tAddr, "err", err.Error())
				m.minerWPPMap[tAddr].isMining = false // Those who fail to pass cannot be mined
				m.minerWPPMap[tAddr].err = append(m.minerWPPMap[tAddr].err, time.Now().Format("2006-01-02 15:04:05 ")+err.Error())
			}
		}()
	}

	return nil
}

func (m *Miner) winPostWarmupForMiner(ctx context.Context, addr address.Address, epp chain.WinningPoStProver) error {
	deadlines, err := m.api.StateMinerDeadlines(ctx, addr, types.EmptyTSK)
	if err != nil {
		return xerrors.Errorf("getting deadlines: %w", err)
	}

	var sector abi.SectorNumber = math.MaxUint64

out:
	for dlIdx := range deadlines {
		partitions, err := m.api.StateMinerPartitions(ctx, addr, uint64(dlIdx), types.EmptyTSK)
		if err != nil {
			return xerrors.Errorf("getting partitions for deadline %d: %w", dlIdx, err)
		}

		for _, partition := range partitions {
			b, err := partition.ActiveSectors.First()
			if err == bitfield.ErrNoBitsSet {
				continue
			}
			if err != nil {
				return err
			}

			sector = abi.SectorNumber(b)
			break out
		}
	}

	if sector == math.MaxUint64 {
		log.Info("skipping winning PoSt warmup, no sectors")
		return nil
	}

	log.Infow("starting winning PoSt warmup", "sector", sector)
	start := time.Now()

	var r abi.PoStRandomness = make([]byte, abi.RandomnessLength)
	_, _ = rand.Read(r)

	si, err := m.api.StateSectorGetInfo(ctx, addr, sector, types.EmptyTSK)
	if err != nil {
		return xerrors.Errorf("getting sector info: %w", err)
	}

	_, err = epp.ComputeProof(ctx, []proof2.SectorInfo{
		{
			SealProof:    si.SealProof,
			SectorNumber: sector,
			SealedCID:    si.SealedCID,
		},
	}, r)
	if err != nil {
		return xerrors.Errorf("failed to compute proof: %w", err)
	}

	log.Infow("winning PoSt warmup successful", "took", time.Since(start))
	return nil
}

func (m *Miner) doWinPoStWarmup(ctx context.Context) {
	err := m.winPoStWarmup(ctx)
	if err != nil {
		log.Errorw("winning PoSt warmup failed", "error", err)
	}
}
