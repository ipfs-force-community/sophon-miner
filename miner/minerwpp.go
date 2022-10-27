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
	"github.com/filecoin-project/venus-miner/node/config"

	"github.com/filecoin-project/venus/pkg/constants"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	v1 "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	"github.com/filecoin-project/venus/venus-shared/types"
)

type MiningWpp struct {
	mAddr       address.Address
	gatewayNode *config.GatewayNode

	winnRpt abi.RegisteredPoStProof
}

func NewWinningPoStProver(api v1.FullNode, gatewayNode *config.GatewayNode, mAddr address.Address) (*MiningWpp, error) {
	mi, err := api.StateMinerInfo(context.TODO(), mAddr, types.EmptyTSK)
	if err != nil {
		return nil, fmt.Errorf("getting sector size: %w", err)
	}

	if constants.InsecurePoStValidation {
		log.Warn("*****************************************************************************")
		log.Warn(" Generating fake PoSt proof! You should only see this while running tests! ")
		log.Warn("*****************************************************************************")
	}

	return &MiningWpp{gatewayNode: gatewayNode, mAddr: mAddr, winnRpt: mi.WindowPoStProofType}, nil
}

var _ WinningPoStProver = (*MiningWpp)(nil)

func (wpp *MiningWpp) GenerateCandidates(ctx context.Context, randomness abi.PoStRandomness, eligibleSectorCount uint64) ([]uint64, error) {
	panic("should not be called")
}

func (wpp *MiningWpp) ComputeProof(ctx context.Context, ssi []builtin.ExtendedSectorInfo, rand abi.PoStRandomness, currEpoch abi.ChainEpoch, nv network.Version) ([]builtin.PoStProof, error) {
	if constants.InsecurePoStValidation {
		return []builtin.PoStProof{{ProofBytes: []byte("valid proof")}}, nil
	}

	log.Infof("Computing WinningPoSt ;%+v; %v", ssi, rand)

	start := build.Clock.Now()

	api, closer, err := client.NewGatewayRPC(ctx, wpp.gatewayNode)
	if err != nil {
		return nil, err
	}
	defer closer()

	proofBuf, err := api.ComputeProof(ctx, wpp.mAddr, ssi, rand, currEpoch, nv)
	if err != nil {
		return nil, err
	}

	log.Infof("GenerateWinningPoSt took %s", time.Since(start))
	return proofBuf, nil
}
