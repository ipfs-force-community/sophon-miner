package modules

import (
	"context"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/build"
	"github.com/filecoin-project/venus-miner/chain"
	"github.com/filecoin-project/venus-miner/chain/actors/builtin"
	"github.com/filecoin-project/venus-miner/chain/types"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/sector-storage/ffiwrapper"
)


type MiningWpp struct {
	minerInfo dtypes.MinerInfo

	verifier ffiwrapper.Verifier
	miner    abi.ActorID
	winnRpt  abi.RegisteredPoStProof
}

func NewWinningPoStProver(api api.FullNode, minerInfo dtypes.MinerInfo, verifier ffiwrapper.Verifier) (*MiningWpp, error) {
	mi, err := api.StateMinerInfo(context.TODO(), minerInfo.Addr, types.EmptyTSK)
	if err != nil {
		return nil, xerrors.Errorf("getting sector size: %w", err)
	}

	if build.InsecurePoStValidation {
		log.Warn("*****************************************************************************")
		log.Warn(" Generating fake PoSt proof! You should only see this while running tests! ")
		log.Warn("*****************************************************************************")
	}

	minerId, err := address.IDFromAddress(minerInfo.Addr)
	if err != nil {
		return nil, err
	}

	return &MiningWpp{minerInfo: minerInfo, verifier: verifier, miner: abi.ActorID(minerId), winnRpt: mi.WindowPoStProofType}, nil
}

var _ chain.WinningPoStProver = (*MiningWpp)(nil)

func (wpp *MiningWpp) GenerateCandidates(ctx context.Context, randomness abi.PoStRandomness, eligibleSectorCount uint64) ([]uint64, error) {
	start := build.Clock.Now()

	cds, err := wpp.verifier.GenerateWinningPoStSectorChallenge(ctx, wpp.winnRpt, wpp.miner, randomness, eligibleSectorCount)
	if err != nil {
		return nil, xerrors.Errorf("failed to generate candidates: %w", err)
	}
	log.Infof("Generate candidates took %s (C: %+v)", time.Since(start), cds)
	return cds, nil
}

func (wpp *MiningWpp) ComputeProof(ctx context.Context, ssi []builtin.SectorInfo, rand abi.PoStRandomness) ([]builtin.PoStProof, error) {
	if build.InsecurePoStValidation {
		return []builtin.PoStProof{{ProofBytes: []byte("valid proof")}}, nil
	}

	log.Infof("Computing WinningPoSt ;%+v; %v", ssi, rand)

	start := build.Clock.Now()

	// todo 调用sealer rpc api

	log.Infof("GenerateWinningPoSt took %s", time.Since(start))
	return nil, nil
}
