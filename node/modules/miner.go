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
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/modules/helpers"
	"github.com/filecoin-project/venus-miner/node/modules/posterkeymgr"
	"github.com/filecoin-project/venus-miner/node/modules/prover_ctor"
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
	fullApi     api.FullNode
	eppCtor     prover_ctor.WinningPostConstructor
	ds          dtypes.MetadataDS
	sf          *slashfilter.SlashFilter
	actorMgr    posterkeymgr.IActorMgr
	minerCfg   *config.MinerConfig
	minerList   map[address.Address]miner.IMiner
	lk          sync.Mutex
	journal     journal.Journal
	//blockRecord block_recorder.IBlockRecord
}

func NewMultiMiner(
	fullApi api.FullNode,
	ds dtypes.MetadataDS,
	cfg *config.MinerConfig,
	eppCtor prover_ctor.WinningPostConstructor,
	actorMgr posterkeymgr.IActorMgr,
	j journal.Journal,
) (*MultiMiner, error) {
	mMiner := &MultiMiner{
		fullApi:     fullApi,
		eppCtor:     eppCtor,
		minerCfg:   cfg,
		actorMgr:    actorMgr,
		ds:          ds,
		sf:          slashfilter.New(ds),
		minerList:   make(map[address.Address]miner.IMiner),
		journal:     j,
	}

	minerAddr, err := minerAddrFromDS(ds)
	if err != nil {
		return nil, err
	}

	if !actorMgr.ExistKey(minerAddr) {
		log.Infof("add miner: %w", minerAddr)
		err := actorMgr.AddKey(config.MinerInfo{Addr: minerAddr, ListenAPI: ""})
		if err != nil {
			return nil, err
		}
	}

	return mMiner, nil
}

func (m *MultiMiner) Start(ctx context.Context) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	log.Info("start to do winning poster")
	posterAddrs, err := m.actorMgr.ListKey()
	if err != nil {
		return err
	}
	for _, addr := range posterAddrs {
		epp, err := m.eppCtor(addr)
		if err != nil {
			return err
		}

		m.minerList[addr.Addr] = miner.NewMiner(m.fullApi, epp, addr.Addr, m.sf, m.journal)
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

func (m *MultiMiner) AddAddress(posterAddr config.MinerInfo) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	if _, ok := m.minerList[posterAddr.Addr]; ok {
		return xerrors.Errorf("exit mining %s", posterAddr.Addr)
	} else {
		epp, err := m.eppCtor(posterAddr)
		if err != nil {
			return err
		}

		innerMiner := miner.NewMiner(m.fullApi, epp, posterAddr.Addr, m.sf, m.journal)
		m.minerList[posterAddr.Addr] = innerMiner
		err = innerMiner.Start(context.Background())
		if err != nil {
			return err
		}
	}
	return m.actorMgr.AddKey(posterAddr)
}

func (m *MultiMiner) RemoveAddress(addr address.Address) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	if mining, ok := m.minerList[addr]; ok {
		err := mining.Stop(context.Background())
		if err != nil {
			return err
		}
		delete(m.minerList, addr)
		return m.actorMgr.RemoveKey(addr)
	} else {
		return xerrors.Errorf("%s not exist", addr)
	}
}

func (m *MultiMiner) ListAddress() ([]config.MinerInfo, error) {
	m.lk.Lock()
	defer m.lk.Unlock()
	return m.actorMgr.ListKey()
}

func SetupPostBlockProducer(lc fx.Lifecycle,
	ds dtypes.MetadataDS,
	api api.FullNode,
	eppCtor prover_ctor.WinningPostConstructor,
	cfg *config.MinerConfig,
	actorMgr posterkeymgr.IActorMgr,
	j journal.Journal,
   //blockRecord block_recorder.IBlockRecord
) (miner.BlockMinerApi, error) {
	var m miner.BlockMinerApi
	m, err := NewMultiMiner(api, ds, cfg, eppCtor, actorMgr, j)
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
