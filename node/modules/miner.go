package modules

import (
	"context"
	"sync"

	"github.com/filecoin-project/go-address"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/chain/gen/slashfilter"
	"github.com/filecoin-project/venus-miner/journal"
	"github.com/filecoin-project/venus-miner/miner"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage"
	"github.com/filecoin-project/venus-miner/sector-storage/ffiwrapper"
)

type minerMiningAPI interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type MultiMiner struct {
	fullApi      api.FullNode
	verifier     ffiwrapper.Verifier
	ds           dtypes.MetadataDS
	minerMap     map[address.Address]minerMiningAPI
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
		sf:           slashfilter.New(ds), // ToDo all miners use the same slashfilter, key: [/f0100/500]
		minerMap:     make(map[address.Address]minerMiningAPI),
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
		m.minerMap[minerInfo.Addr] = miner.NewMiner(m.fullApi, epp, minerInfo.Addr, m.sf, m.journal)
	}

	for _, m := range m.minerMap {
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
	for _, m := range m.minerMap {
		err := m.Stop(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiMiner) ManualStart(ctx context.Context, addr address.Address) error {
	m.lk.Lock()
	defer m.lk.Unlock()

	if mining, ok := m.minerMap[addr]; ok {
		return mining.Start(ctx)
	}

	return xerrors.Errorf("%s not exist", addr)
}

func (m *MultiMiner) ManualStop(ctx context.Context, addr address.Address) error {
	m.lk.Lock()
	defer m.lk.Unlock()

	if mining, ok := m.minerMap[addr]; ok {
		return mining.Stop(ctx)
	}

	return xerrors.Errorf("%s not exist", addr)
}

func (m *MultiMiner) AddAddress(minerInfo dtypes.MinerInfo) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	if _, ok := m.minerMap[minerInfo.Addr]; ok {
		return xerrors.Errorf("exit mining %s", minerInfo.Addr)
	} else {
		epp, err := NewWinningPoStProver(m.fullApi, minerInfo, m.verifier)
		if err != nil {
			return err
		}

		newMiner := miner.NewMiner(m.fullApi, epp, minerInfo.Addr, m.sf, m.journal)

		m.minerMap[minerInfo.Addr] = newMiner
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
	if minerAPI, ok := m.minerMap[addr]; ok {
		err := minerAPI.Stop(context.Background())
		if err != nil {
			return err
		}

		delete(m.minerMap, addr)

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
