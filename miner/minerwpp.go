package miner

import (
	"context"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/venus-miner/api/client"
	"github.com/filecoin-project/venus-miner/build"
	"github.com/filecoin-project/venus-miner/chain"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/sector-storage/ffiwrapper"

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

	return &MiningWpp{gatewayNode: gatewayNode, minerInfo: minerInfo, verifier: verifier, miner: abi.ActorID(minerId), winnRpt: mi.WindowPoStProofType}, nil
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

func (wpp *MiningWpp) ComputeProof(ctx context.Context, ssi []builtin.ExtendedSectorInfo, rand abi.PoStRandomness, currEpoch abi.ChainEpoch, nv network.Version) ([]builtin.PoStProof, error) {
	if build.InsecurePoStValidation {
		return []builtin.PoStProof{{ProofBytes: []byte("valid proof")}}, nil
	}

	log.Infof("Computing WinningPoSt ;%+v; %v", ssi, rand)

	start := build.Clock.Now()

	// todo call gateway api
	api, closer, err := client.NewGatewayRPC(wpp.gatewayNode)
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

//// sealerStruct
//type sealerStruct struct {
//	Internal struct {
//		ComputeProof func(context.Context, []proof2.SectorInfo, abi.PoStRandomness) ([]proof2.PoStProof, error) `perm:"read"`
//	}
//}
//
//func (s *sealerStruct) ComputeProof(ctx context.Context, ssi []proof2.SectorInfo, rand abi.PoStRandomness) ([]proof2.PoStProof, error) {
//	return s.Internal.ComputeProof(ctx, ssi, rand)
//}
//
//func newSealerRPC(addr string, requestHeader http.Header) (sealerAPI, jsonrpc.ClientCloser, error) {
//	var res sealerStruct
//	closer, err := jsonrpc.NewMergeClient(context.Background(), addr, "Filecoin",
//		[]interface{}{
//			&res.Internal,
//		},
//		requestHeader,
//	)
//
//	return &res, closer, err
//}
//
//func getSealerAPI(ctx context.Context, sealer dtypes.SealerNode) (sealerAPI, jsonrpc.ClientCloser, error) {
//	addr, err := sealer.DialArgs()
//	if err != nil {
//		return nil, nil, xerrors.Errorf("could not get DialArgs: %w", err)
//	}
//
//
//	return newSealerRPC(addr, sealer.AuthHeader())
//}
