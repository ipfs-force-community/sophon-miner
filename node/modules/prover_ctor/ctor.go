package prover_ctor

import (
	"github.com/filecoin-project/venus-miner/node/modules/helpers"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/chain"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus-miner/sector-storage/stores"
	"github.com/filecoin-project/venus-miner/storage"
	"github.com/filecoin-project/venus-miner/storage/prover/local"
)

type WinningPostConstructor func(minerConfig config.PosterAddr) (chain.WinningPoStProver, error)

func WinningPostProverCCTor(
	mctx helpers.MetricsCtx,
	api api.FullNode,
	verifier ffiwrapper.Verifier,
	ls stores.LocalStorage,
	si stores.SectorIndex,
	cfg *config.MinerConfig,
	urls ffiwrapper.URLs,
) WinningPostConstructor {
	return func(posterAddr config.PosterAddr) (chain.WinningPoStProver, error) {
		minerId, err := address.IDFromAddress(posterAddr.Addr)
		if err != nil {
			return nil, nil
		}
		prover, err := local.NewProver(mctx, ls, si, cfg, urls, dtypes.MinerID(minerId))
		if err != nil {
			return nil, nil
		}

		epp, err := storage.NewWinningPoStProver(api, prover, verifier, posterAddr.Addr)
		if err != nil {
			return nil, nil
		}
		return epp, nil
	}
}
