package miner

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/venus-miner/api/client"
	"github.com/filecoin-project/venus-miner/build"
	"github.com/filecoin-project/venus-miner/chain"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"

	"github.com/filecoin-project/venus/pkg/util/ffiwrapper"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	v1 "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	"github.com/filecoin-project/venus/venus-shared/types"
)

type MiningWpp struct {
	minerInfo   dtypes.MinerInfo
	gatewayNode *config.GatewayNode

	verifier ffiwrapper.Verifier
	miner    abi.ActorID
	winnRpt  abi.RegisteredPoStProof
}

func NewWinningPoStProver(api v1.FullNode, gatewayNode *config.GatewayNode, minerInfo dtypes.MinerInfo, verifier ffiwrapper.Verifier) (*MiningWpp, error) {
	mi, err := api.StateMinerInfo(context.TODO(), minerInfo.Addr, types.EmptyTSK)
	if err != nil {
		return nil, fmt.Errorf("getting sector size: %w", err)
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

	return &MiningWpp{gatewayNode: gatewayNode, minerInfo: minerInfo, verifier: verifier, miner: abi.ActorID(minerId), winnRpt: mi.WindowPoStProofType}, nil
}

var _ chain.WinningPoStProver = (*MiningWpp)(nil)

func (wpp *MiningWpp) GenerateCandidates(ctx context.Context, randomness abi.PoStRandomness, eligibleSectorCount uint64) ([]uint64, error) {
	start := build.Clock.Now()

	cds, err := wpp.verifier.GenerateWinningPoStSectorChallenge(ctx, wpp.winnRpt, wpp.miner, randomness, eligibleSectorCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate candidates: %w", err)
	}
	log.Infof("Generate candidates took %s (C: %+v)", time.Since(start), cds)
	return cds, nil
}

func (wpp *MiningWpp) ComputeProof(ctx context.Context, ssi []builtin.ExtendedSectorInfo, rand abi.PoStRandomness, currEpoch abi.ChainEpoch, nv network.Version) ([]builtin.PoStProof, error) {
	if build.InsecurePoStValidation {
		return []builtin.PoStProof{{ProofBytes: []byte("valid proof")}}, nil
	}

	log.Infof("Computing WinningPoSt ;%+v; %v", ssi, rand)

	start := build.Clock.Now()

	// todo call gateway api
	api, closer, err := client.NewGatewayRPC(ctx, wpp.gatewayNode)
	if err != nil {
		return nil, err
	}
	defer closer()

	proofBuf, err := api.ComputeProof(ctx, wpp.minerInfo.Addr, ssi, rand, currEpoch, nv)
	if err != nil {
		return nil, err
	}

	log.Infof("GenerateWinningPoSt took %s", time.Since(start))
	return proofBuf, nil
}
