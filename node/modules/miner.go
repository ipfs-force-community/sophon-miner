package modules

import (
	"context"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs/go-datastore"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/build"
	"github.com/filecoin-project/venus-miner/chain/gen/slashfilter"
	"github.com/filecoin-project/venus-miner/journal"
	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/modules/helpers"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage"
	"github.com/filecoin-project/venus-miner/sector-storage/ffiwrapper"
)

func MinerAddress(ds dtypes.MetadataDS) (dtypes.MinerAddress, error) {
	ma, err := minerAddrFromDS(ds)
	return dtypes.MinerAddress(ma), err
}

func minerAddrFromDS(ds dtypes.MetadataDS) (address.Address, error) {
	maddrb, err := ds.Get(datastore.NewKey("miner-address"))
	if err != nil {
		return address.Undef, err
	}

	return address.NewFromBytes(maddrb)
}

func MinerNetworkName(ctx helpers.MetricsCtx, a api.FullNode) (dtypes.NetworkName, error) {
	if !build.Devnet {
		return "testnetnet", nil
	}
	return a.StateNetworkName(ctx)
}

type MultiMiner struct {
	fullApi      api.FullNode
	verifier     ffiwrapper.Verifier
	ds           dtypes.MetadataDS
	minerList    map[address.Address]miner.IMinerMining
	minerManager minermanage.MinerManageAPI
	sf           *slashfilter.SlashFilter

	lk      sync.Mutex
	journal journal.Journal
	//blockRecord block_recorder.IBlockRecord
}

func NewMultiMiner(
	fullApi api.FullNode,
	verifier ffiwrapper.Verifier,
	ds dtypes.MetadataDS,
	minerManager minermanage.MinerManageAPI,
	j journal.Journal,
) (*MultiMiner, error) {
	mMiner := &MultiMiner{
		fullApi:      fullApi,
		verifier:     verifier,
		minerManager: minerManager,
		ds:           ds,
		sf:           slashfilter.New(ds),
		minerList:    make(map[address.Address]miner.IMinerMining),
		journal:      j,
	}

	return mMiner, nil
}

func (m *MultiMiner) Start(ctx context.Context) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	log.Info("start to do winning poster")

	miners, err := m.minerManager.List()
	if err != nil {
		return err
	}
	for _, minerInfo := range miners {
		epp, err := NewWinningPoStProver(m.fullApi, minerInfo, m.verifier)
		if err != nil {
			return err
		}
		m.minerList[minerInfo.Addr] = miner.NewMiner(m.fullApi, epp, minerInfo.Addr, m.sf, m.journal)
	}

	for _, m := range m.minerList {
		err := m.Start(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiMiner) Stop(ctx context.Context) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	for _, m := range m.minerList {
		err := m.Stop(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiMiner) AddAddress(minerInfo dtypes.MinerInfo) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	if _, ok := m.minerList[minerInfo.Addr]; ok {
		return xerrors.Errorf("exit mining %s", minerInfo.Addr)
	} else {
		epp, err := NewWinningPoStProver(m.fullApi, minerInfo, m.verifier)
		if err != nil {
			return err
		}

		newMiner := miner.NewMiner(m.fullApi, epp, minerInfo.Addr, m.sf, m.journal)

		m.minerList[minerInfo.Addr] = newMiner
		err = newMiner.Start(context.Background())
		if err != nil {
			return err
		}
	}

	return m.minerManager.Add(minerInfo)
}

func (m *MultiMiner) RemoveAddress(addr address.Address) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	if minerAPI, ok := m.minerList[addr]; ok {
		err := minerAPI.Stop(context.Background())
		if err != nil {
			return err
		}

		delete(m.minerList, addr)

		return m.minerManager.Remove(addr)
	} else {
		return xerrors.Errorf("%s not exist", addr)
	}
}

func (m *MultiMiner) ListAddress() ([]dtypes.MinerInfo, error) {
	m.lk.Lock()
	defer m.lk.Unlock()

	return m.minerManager.List()
}

func NewWiningPoster(lc fx.Lifecycle,
	ds dtypes.MetadataDS,
	api api.FullNode,
	verifier ffiwrapper.Verifier,
	minerManager minermanage.MinerManageAPI,
	j journal.Journal,
	//blockRecord block_recorder.IBlockRecord
) (miner.MiningAPI, error) {
	var m miner.MiningAPI
	m, err := NewMultiMiner(api, verifier, ds, minerManager, j)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := m.Start(ctx); err != nil {
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return m.Stop(ctx)
		},
	})

	return m, nil
}
