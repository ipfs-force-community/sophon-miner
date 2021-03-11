package miner

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	proof2 "github.com/filecoin-project/specs-actors/v2/actors/runtime/proof"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	lru "github.com/hashicorp/golang-lru"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/build"
	"github.com/filecoin-project/venus-miner/chain"
	"github.com/filecoin-project/venus-miner/chain/gen/slashfilter"
	"github.com/filecoin-project/venus-miner/chain/types"
	"github.com/filecoin-project/venus-miner/journal"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage"
	"github.com/filecoin-project/venus-miner/sector-storage/ffiwrapper"

	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"
)

var log = logging.Logger("miner")

// Journal event types.
const (
	evtTypeBlockMined = iota
)

// returns a callback reporting whether we mined a blocks in this round
type waitFunc func(ctx context.Context, baseTime uint64) (func(bool, abi.ChainEpoch, error), abi.ChainEpoch, error)

func randTimeOffset(width time.Duration) time.Duration {
	buf := make([]byte, 8)
	rand.Reader.Read(buf) //nolint:errcheck
	val := time.Duration(binary.BigEndian.Uint64(buf) % uint64(width))

	return val - (width / 2)
}

func NewMiner(api api.FullNode, verifier ffiwrapper.Verifier, minerManager minermanage.MinerManageAPI,
	sf *slashfilter.SlashFilter /*blockRecord block_recorder.IBlockRecord,*/, j journal.Journal) *Miner {
	arc, err := lru.NewARC(10000)
	if err != nil {
		panic(err)
	}

	return &Miner{
		api: api,
		waitFunc: func(ctx context.Context, baseTime uint64) (func(bool, abi.ChainEpoch, error), abi.ChainEpoch, error) {
			// Wait around for half the block time in case other parents come in
			deadline := baseTime + build.PropagationDelaySecs
			baseT := time.Unix(int64(deadline), 0)

			baseT = baseT.Add(randTimeOffset(time.Second))

			build.Clock.Sleep(build.Clock.Until(baseT))

			return func(bool, abi.ChainEpoch, error) {}, 0, nil
		},

		sf:                sf,
		minedBlockHeights: arc,

		//blockRecord: blockRecord,

		evtTypes: [...]journal.EventType{
			evtTypeBlockMined: j.RegisterEventType("miner", "block_mined"),
		},
		journal: j,

		minerManager: minerManager,
		minerWPPMap:  make(map[address.Address]*minerWPP),

		verifier: verifier,

		mineTimeout: build.BlockDelaySecs - build.PropagationDelaySecs,
	}
}

type minerWPP struct {
	epp      chain.WinningPoStProver
	isMining bool
}

type Miner struct {
	api api.FullNode

	lk       sync.Mutex
	stop     chan struct{}
	stopping chan struct{}

	waitFunc waitFunc

	lastWork *MiningBase

	sf                *slashfilter.SlashFilter
	minedBlockHeights *lru.ARCCache

	evtTypes [1]journal.EventType
	journal  journal.Journal

	lkWPP        sync.Mutex
	minerWPPMap  map[address.Address]*minerWPP
	minerManager minermanage.MinerManageAPI

	verifier ffiwrapper.Verifier

	mineTimeout uint64 // the timeout of mining once

	//blockRecord block_recorder.IBlockRecord
}

func (m *Miner) Start(ctx context.Context) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	if m.stop != nil {
		return fmt.Errorf("miner already started")
	}

	// init miners
	miners, err := m.minerManager.List()
	if err != nil {
		return err
	}
	for _, minerInfo := range miners {
		epp, err := NewWinningPoStProver(m.api, minerInfo, m.verifier)
		if err != nil {
			return err
		}
		m.minerWPPMap[minerInfo.Addr] = &minerWPP{epp: epp, isMining: true}
	}

	m.stop = make(chan struct{})
	go m.mine(context.TODO())
	return nil
}

func (m *Miner) Stop(ctx context.Context) error {
	m.lk.Lock()

	m.stopping = make(chan struct{})
	stopping := m.stopping
	close(m.stop)

	m.lk.Unlock()

	select {
	case <-stopping:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Miner) niceSleep(d time.Duration) bool {
	select {
	case <-build.Clock.After(d):
		return true
	case <-m.stop:
		log.Infow("received interrupt while trying to sleep in mining cycle")
		return false
	}
}

func (m *Miner) mine(ctx context.Context) {
	log.Info("start to do winning poster")
	ctx, span := trace.StartSpan(ctx, "/mine")
	defer span.End()

	go m.doWinPoStWarmup(ctx)

	var lastBase MiningBase
minerLoop:
	for {
		select {
		case <-m.stop:
			stopping := m.stopping
			m.stop = nil
			m.stopping = nil
			close(stopping)
			return

		default:
		}

		// if there are no miners, wait
		if len(m.minerWPPMap) <=0 {
			log.Warn("no miner is configured, please check ... ")
			if !m.niceSleep(time.Second * 5) {
				continue minerLoop
			}
			continue
		}

		var base *MiningBase
		var onDone func(bool, abi.ChainEpoch, error)
		var injectNulls abi.ChainEpoch

		for {
			prebase, err := m.GetBestMiningCandidate(ctx)
			if err != nil {
				log.Errorf("failed to get best mining candidate: %s", err)
				if !m.niceSleep(time.Second * 5) {
					continue minerLoop
				}
				continue
			}

			if base != nil && base.TipSet.Height() == prebase.TipSet.Height() && base.NullRounds == prebase.NullRounds {
				base = prebase
				break
			}
			if base != nil {
				onDone(false, 0, nil)
			}

			// TODO: need to change the orchestration here. the problem is that
			// we are waiting *after* we enter this loop and selecta mining
			// candidate, which is almost certain to change in multiminer
			// tests. Instead, we should block before entering the loop, so
			// that when the test 'MineOne' function is triggered, we pull our
			// best mining candidate at that time.

			// Wait until propagation delay period after block we plan to mine on
			onDone, injectNulls, err = m.waitFunc(ctx, prebase.TipSet.MinTimestamp())
			if err != nil {
				log.Error(err)
				continue
			}

			// just wait for the beacon entry to become available before we select our final mining base
			_, err = m.api.BeaconGetEntry(ctx, prebase.TipSet.Height()+prebase.NullRounds+1)
			if err != nil {
				log.Errorf("failed getting beacon entry: %s", err)
				if !m.niceSleep(time.Second) {
					continue minerLoop
				}
				continue
			}

			base = prebase
		}

		base.NullRounds += injectNulls // testing

		if base.TipSet.Equals(lastBase.TipSet) && lastBase.NullRounds == base.NullRounds {
			log.Warnf("BestMiningCandidate from the previous round: %s (nulls:%d)", lastBase.TipSet.Cids(), lastBase.NullRounds)
			if !m.niceSleep(time.Duration(build.BlockDelaySecs) * time.Second) {
				continue minerLoop
			}
			continue
		}

		// ToDo each miner mine once in each round, need to judge the timeout !!!
		var (
			winPoSts []*winPoStRes
			wg       sync.WaitGroup
		)
		m.lkWPP.Lock()
		for addr, mining := range m.minerWPPMap {
			if mining.isMining {
				wg.Add(1)
				epp := mining.epp
				tAddr := addr

				go func() {
					defer wg.Done()

					// set timeout for miner once
					tCtx, tCtxCancel := context.WithTimeout(ctx, time.Second*time.Duration(m.mineTimeout))
					defer tCtxCancel()

					resChan, err := m.mineOne(tCtx, base, tAddr, epp)
					if err != nil { // ToDo retry or continue minerLoop?
						log.Errorf("mining block failed for %s: %+v", tAddr.String(), err)
						return
					}

					// waiting for mining results
					for {
						select {
						case <-tCtx.Done():
							log.Errorf("mining block timeout for %s", tAddr.String())
							return
						case res := <-resChan:
							if res != nil && res.winner != nil {
								winPoSts = append(winPoSts, res) //nolint:staticcheck
							}
						}
					}
				}()

			}
		}
		wg.Wait()
		m.lkWPP.Unlock()

		log.Infow("mining compute end", "number of wins", len(winPoSts), "total miner", len(m.minerWPPMap))
		lastBase = *base

		if len(winPoSts) > 0 { // the size of winPoSts indicates the number of blocks
			// get pending messages early,
			ticketQualitys := make([]float64, len(winPoSts))
			for idx, res := range winPoSts {
				ticketQualitys[idx] = res.ticket.Quality()
			}
			log.Infow("select message", "tickets", len(ticketQualitys))
			msgs, err := m.api.MpoolSelects(context.TODO(), base.TipSet.Key(), ticketQualitys)
			if err != nil {
				log.Errorf("failed to select messages for block: %w", err)
				return
			}
			tPending := build.Clock.Now()

			// create blocks
			var blks []*types.BlockMsg
			for idx, res := range winPoSts {
				tRes := res
				// TODO: winning post proof
				b, err := m.createBlock(base, tRes.addr, tRes.ticket, tRes.winner, tRes.bvals, tRes.postProof, msgs[idx])
				if err != nil {
					log.Errorf("failed to create block: %w", err)
					return
				}
				blks = append(blks, b)

				tCreateBlock := build.Clock.Now()
				dur := tCreateBlock.Sub(tRes.timetable.tStart)
				parentMiners := make([]address.Address, len(base.TipSet.Blocks()))
				for i, header := range base.TipSet.Blocks() {
					parentMiners[i] = header.Miner
				}
				log.Infow("mined new block", "cid", b.Cid(), "height", b.Header.Height, "miner", b.Header.Miner, "parents", parentMiners, "took", dur)

				if dur > time.Second*time.Duration(build.BlockDelaySecs) {
					log.Warnw("CAUTION: block production took longer than the block delay. Your computer may not be fast enough to keep up",
						"tMinerBaseInfo ", tRes.timetable.tMBI.Sub(tRes.timetable.tStart),
						"tDrand ", tRes.timetable.tDrand.Sub(tRes.timetable.tMBI),
						"tPowercheck ", tRes.timetable.tPowercheck.Sub(tRes.timetable.tDrand),
						"tTicket ", tRes.timetable.tTicket.Sub(tRes.timetable.tPowercheck),
						"tSeed ", tRes.timetable.tSeed.Sub(tRes.timetable.tTicket),
						"tProof ", tRes.timetable.tProof.Sub(tRes.timetable.tSeed),
						"tPending ", tPending.Sub(tRes.timetable.tProof),
						"tCreateBlock ", tCreateBlock.Sub(tPending))
				}

				m.journal.RecordEvent(m.evtTypes[evtTypeBlockMined], func() interface{} {
					return map[string]interface{}{
						"parents":   base.TipSet.Cids(),
						"nulls":     base.NullRounds,
						"epoch":     b.Header.Height,
						"timestamp": b.Header.Timestamp,
						"cid":       b.Header.Cid(),
						"miner":     b.Header.Miner,
					}
				})
			}

			btime := time.Unix(int64(blks[0].Header.Timestamp), 0)
			now := build.Clock.Now()
			switch {
			case btime == now:
				// block timestamp is perfectly aligned with time.
			case btime.After(now):
				if !m.niceSleep(build.Clock.Until(btime)) {
					log.Warnf("received interrupt while waiting to broadcast block, will shutdown after block is sent out")
					build.Clock.Sleep(build.Clock.Until(btime))
				}
			default:
				log.Warnw("mined block in the past",
					"block-time", btime, "time", build.Clock.Now(), "difference", build.Clock.Since(btime))
			}

			// broadcast all blocks
			for _, b := range blks {
				go func(bm *types.BlockMsg) {
					if err := m.sf.MinedBlock(bm.Header, base.TipSet.Height()+base.NullRounds); err != nil {
						log.Errorf("<!!> SLASH FILTER ERROR: %s", err)
						return
					}

					blkKey := fmt.Sprintf("%s-%d", bm.Header.Miner, bm.Header.Height)
					if _, ok := m.minedBlockHeights.Get(blkKey); ok {
						log.Warnw("Created a block at the same height as another block we've created", "height", bm.Header.Height, "miner", bm.Header.Miner, "parents", bm.Header.Parents)
						return
					}

					m.minedBlockHeights.Add(blkKey, true)

					// TODO: should do better 'anti slash' protection here
					//has := m.blockRecord.Has(m.address, uint64(b.Header.Height))
					//if has {
					//	log.Warnw("Created a block at the same height as another block we've created", "height", b.Header.Height, "miner", b.Header.Miner, "parents", b.Header.Parents)
					//	continue
					//}
					//
					//err = m.blockRecord.MarkAsProduced(m.address, uint64(b.Header.Height))
					//if err != nil {
					//	log.Errorf("failed to write db: %s", err)
					//}

					if err := m.api.SyncSubmitBlock(ctx, bm); err != nil {
						log.Errorf("failed to submit newly mined block: %s", err)
					}
				}(b)
			}
		} else {
			base.NullRounds++
			log.Info("no block and increase nullround")
			// Wait until the next epoch, plus the propagation delay, so a new tipset
			// has enough time to form.
			//
			// See:  https://github.com/filecoin-project/venus-miner/issues/1845
			nextRound := time.Unix(int64(base.TipSet.MinTimestamp()+build.BlockDelaySecs*uint64(base.NullRounds))+int64(build.PropagationDelaySecs), 0)

			select {
			case <-build.Clock.After(build.Clock.Until(nextRound)):
			case <-m.stop:
				stopping := m.stopping
				m.stop = nil
				m.stopping = nil
				close(stopping)
				return
			}
		}
	}
}

type MiningBase struct {
	TipSet     *types.TipSet
	NullRounds abi.ChainEpoch
}

func (m *Miner) GetBestMiningCandidate(ctx context.Context) (*MiningBase, error) {
	m.lk.Lock()
	defer m.lk.Unlock()

	bts, err := m.api.ChainHead(ctx)
	if err != nil {
		return nil, err
	}

	if m.lastWork != nil {
		if m.lastWork.TipSet.Equals(bts) {
			return m.lastWork, nil
		}

		btsw, err := m.api.ChainTipSetWeight(ctx, bts.Key())
		if err != nil {
			return nil, err
		}
		ltsw, err := m.api.ChainTipSetWeight(ctx, m.lastWork.TipSet.Key())
		if err != nil {
			m.lastWork = nil
			return nil, err
		}

		if types.BigCmp(btsw, ltsw) <= 0 {
			return m.lastWork, nil
		}
	}

	m.lastWork = &MiningBase{TipSet: bts}
	return m.lastWork, nil
}

type miningTimetable struct {
	tStart, tMBI, tDrand, tPowercheck, tTicket, tSeed, tProof time.Time
}

type winPoStRes struct {
	addr      address.Address
	ticket    *types.Ticket
	winner    *types.ElectionProof
	bvals     []types.BeaconEntry
	postProof []proof2.PoStProof
	dur       time.Duration
	timetable miningTimetable
}

// mineOne attempts to mine a single block, and does so synchronously, if and
// only if we are eligible to mine.
//
// {hint/landmark}: This method coordinates all the steps involved in mining a
// block, including the condition of whether mine or not at all depending on
// whether we win the round or not.
//
// This method does the following:
//
//  1.
func (m *Miner) mineOne(ctx context.Context, base *MiningBase, addr address.Address, epp chain.WinningPoStProver) (<-chan *winPoStRes, error) {
	log.Infow("attempting to mine a block", "tipset", types.LogCids(base.TipSet.Cids()), "miner", addr)

	out := make(chan *winPoStRes)

	go func() {
		start := build.Clock.Now()

		round := base.TipSet.Height() + base.NullRounds + 1

		mbi, err := m.api.MinerGetBaseInfo(ctx, addr, round, base.TipSet.Key())
		if err != nil {
			log.Errorf("failed to get mining base info: %w, miner: %s", err, addr)
			return
		}
		if mbi == nil {
			log.Infow("get nil MinerGetBaseInfo", "miner", addr)
			return
		}
		if !mbi.EligibleForMining {
			// slashed or just have no power yet
			return
		}

		tMBI := build.Clock.Now()

		beaconPrev := mbi.PrevBeaconEntry

		tDrand := build.Clock.Now()
		bvals := mbi.BeaconEntries

		tPowercheck := build.Clock.Now()

		log.Infof("Time delta between now and our mining base: %ds (nulls: %d), miner: %s", uint64(build.Clock.Now().Unix())-base.TipSet.MinTimestamp(), base.NullRounds, addr)

		rbase := beaconPrev
		if len(bvals) > 0 {
			rbase = bvals[len(bvals)-1]
		}

		ticket, err := m.computeTicket(ctx, &rbase, base, mbi, addr)
		if err != nil {
			log.Errorf("scratching ticket for %s failed: %w", addr, err)
			return
		}

		winner, err := chain.IsRoundWinner(ctx, base.TipSet, round, addr, rbase, mbi, m.api)
		if err != nil {
			log.Errorf("failed to check for %s if we win next round: %w", addr, err)
			return
		}

		if winner == nil {
			log.Infow("not to be winner", "miner", addr)
			return
		}

		tTicket := build.Clock.Now()

		buf := new(bytes.Buffer)
		if err := addr.MarshalCBOR(buf); err != nil {
			log.Errorf("failed to marshal miner address: %w", err)
			return
		}

		r, err := chain.DrawRandomness(rbase.Data, crypto.DomainSeparationTag_WinningPoStChallengeSeed, round, buf.Bytes())
		if err != nil {
			log.Errorf("failed to get randomness for winning post: %w, miner: %s", err, addr)
			return
		}

		prand := abi.PoStRandomness(r)

		tSeed := build.Clock.Now()

		postProof, err := epp.ComputeProof(ctx, mbi.Sectors, prand)
		if err != nil {
			log.Errorf("failed to compute winning post proof: %w, miner: %s", err, addr)
			return
		}

		tProof := build.Clock.Now()

		dur := build.Clock.Now().Sub(start)
		tt := miningTimetable{
			tStart: start, tMBI: tMBI, tDrand: tDrand, tPowercheck: tPowercheck, tTicket: tTicket, tSeed: tSeed, tProof: tProof,
		}
		out <- &winPoStRes{addr: addr, ticket: ticket, winner: winner, bvals: bvals, postProof: postProof, dur: dur, timetable: tt}
		log.Infow("mined new block ( -> Proof)", "took", dur, "miner", addr)
	}()

	return out, nil
}

func (m *Miner) computeTicket(ctx context.Context, brand *types.BeaconEntry, base *MiningBase, mbi *api.MiningBaseInfo, addr address.Address) (*types.Ticket, error) {
	buf := new(bytes.Buffer)
	if err := addr.MarshalCBOR(buf); err != nil {
		return nil, xerrors.Errorf("failed to marshal address to cbor: %w", err)
	}

	round := base.TipSet.Height() + base.NullRounds + 1
	if round > build.UpgradeSmokeHeight {
		buf.Write(base.TipSet.MinTicket().VRFProof)
	}

	input, err := chain.DrawRandomness(brand.Data, crypto.DomainSeparationTag_TicketProduction, round-build.TicketRandomnessLookback, buf.Bytes())
	if err != nil {
		return nil, err
	}

	vrfOut, err := chain.ComputeVRF(ctx, m.api.WalletSign, mbi.WorkerKey, input)
	if err != nil {
		return nil, err
	}

	return &types.Ticket{
		VRFProof: vrfOut,
	}, nil
}

func (m *Miner) createBlock(base *MiningBase, addr address.Address, ticket *types.Ticket,
	eproof *types.ElectionProof, bvals []types.BeaconEntry, wpostProof []proof2.PoStProof, msgs []*types.SignedMessage) (*types.BlockMsg, error) {
	uts := base.TipSet.MinTimestamp() + build.BlockDelaySecs*(uint64(base.NullRounds)+1)

	nheight := base.TipSet.Height() + base.NullRounds + 1

	// why even return this? that api call could just submit it for us
	return m.api.MinerCreateBlock(context.TODO(), &api.BlockTemplate{
		Miner:            addr,
		Parents:          base.TipSet.Key(),
		Ticket:           ticket,
		Eproof:           eproof,
		BeaconValues:     bvals,
		Messages:         msgs,
		Epoch:            nheight,
		Timestamp:        uts,
		WinningPoStProof: wpostProof,
	})
}

func (m *Miner) ManualStart(ctx context.Context, addr address.Address) error {
	m.lkWPP.Lock()
	defer m.lkWPP.Unlock()

	if mining, ok := m.minerWPPMap[addr]; ok {
		mining.isMining = true
		return nil
	}

	return xerrors.Errorf("%s not exist", addr)
}

func (m *Miner) ManualStop(ctx context.Context, addr address.Address) error {
	m.lkWPP.Lock()
	defer m.lkWPP.Unlock()

	if mining, ok := m.minerWPPMap[addr]; ok {
		mining.isMining = false
		return nil
	}

	return xerrors.Errorf("%s not exist", addr)
}

func (m *Miner) AddAddress(minerInfo dtypes.MinerInfo) error {
	m.lkWPP.Lock()
	defer m.lkWPP.Unlock()
	if _, ok := m.minerWPPMap[minerInfo.Addr]; ok {
		return xerrors.Errorf("exit mining %s", minerInfo.Addr)
	} else {
		epp, err := NewWinningPoStProver(m.api, minerInfo, m.verifier)
		if err != nil {
			return err
		}

		m.minerWPPMap[minerInfo.Addr] = &minerWPP{epp: epp, isMining: true}
	}

	return m.minerManager.Add(minerInfo)
}

func (m *Miner) RemoveAddress(addr address.Address) error {
	m.lkWPP.Lock()
	defer m.lkWPP.Unlock()
	if _, ok := m.minerWPPMap[addr]; ok {
		delete(m.minerWPPMap, addr)

		return m.minerManager.Remove(addr)
	}

	return xerrors.Errorf("%s not exist", addr)
}

func (m *Miner) ListAddress() ([]dtypes.MinerInfo, error) {
	m.lkWPP.Lock()
	defer m.lkWPP.Unlock()

	return m.minerManager.List()
}
