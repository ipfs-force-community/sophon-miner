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

	"github.com/filecoin-project/venus-wallet/core"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"

	"github.com/filecoin-project/venus-miner/api"
	"github.com/filecoin-project/venus-miner/api/client"
	"github.com/filecoin-project/venus-miner/build"
	"github.com/filecoin-project/venus-miner/chain"
	"github.com/filecoin-project/venus-miner/chain/gen/slashfilter"
	"github.com/filecoin-project/venus-miner/chain/types"
	"github.com/filecoin-project/venus-miner/journal"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/modules/block_recorder"
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

var DefaultMaxErrCounts = 20

// returns a callback reporting whether we mined a blocks in this round
type waitFunc func(ctx context.Context, baseTime uint64) (func(bool, abi.ChainEpoch, error), abi.ChainEpoch, error)

func randTimeOffset(width time.Duration) time.Duration {
	buf := make([]byte, 8)
	rand.Reader.Read(buf) //nolint:errcheck
	val := time.Duration(binary.BigEndian.Uint64(buf) % uint64(width))

	return val - (width / 2)
}

func NewMiner(api api.FullNode, gtNode *config.GatewayNode, verifier ffiwrapper.Verifier, minerManager minermanage.MinerManageAPI,
	sf slashfilter.SlashFilterAPI, j journal.Journal, blockRecord block_recorder.IBlockRecord) *Miner {
	miner := &Miner{
		api:         api,
		gatewayNode: gtNode,
		waitFunc: func(ctx context.Context, baseTime uint64) (func(bool, abi.ChainEpoch, error), abi.ChainEpoch, error) {
			// Wait around for half the block time in case other parents come in
			deadline := baseTime + build.PropagationDelaySecs
			baseT := time.Unix(int64(deadline), 0)

			baseT = baseT.Add(randTimeOffset(time.Second))

			build.Clock.Sleep(build.Clock.Until(baseT))

			return func(bool, abi.ChainEpoch, error) {}, 0, nil
		},

		sf:          sf,
		blockRecord: blockRecord,

		evtTypes: [...]journal.EventType{
			evtTypeBlockMined: j.RegisterEventType("miner", "block_mined"),
		},
		journal: j,

		minerManager: minerManager,
		minerWPPMap:  make(map[address.Address]*minerWPP),

		verifier: verifier,
	}

	switch build.BuildType {
	case build.BuildMainnet, build.BuildCalibnet, build.BuildNerpanet: // The time to wait for the latest block is counted in
		miner.mineTimeout = 12 * time.Second
		// miner.mineTimeout = time.Duration(build.BlockDelaySecs-build.PropagationDelaySecs*2) * time.Second
	default:
		miner.mineTimeout = time.Millisecond * 2800 // 0.2S is used to select messages and generate blocks
	}

	return miner
}

type syncStatus struct {
	heightDiff int64
	err        error
}

type minerWPP struct {
	account  string
	epp      chain.WinningPoStProver
	isMining bool
	err      []string
}

type Miner struct {
	api api.FullNode

	gatewayNode *config.GatewayNode

	lk       sync.Mutex
	stop     chan struct{}
	stopping chan struct{}

	waitFunc waitFunc

	lastWork *MiningBase

	sf          slashfilter.SlashFilterAPI
	blockRecord block_recorder.IBlockRecord
	// minedBlockHeights *lru.ARCCache

	evtTypes [1]journal.EventType
	journal  journal.Journal

	st syncStatus

	lkWPP        sync.Mutex
	minerWPPMap  map[address.Address]*minerWPP
	minerManager minermanage.MinerManageAPI

	verifier ffiwrapper.Verifier

	mineTimeout time.Duration // the timeout of mining once

}

func (m *Miner) Start(ctx context.Context) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	if m.stop != nil {
		return fmt.Errorf("miner already started")
	}

	// init miners
	miners, err := m.minerManager.List(ctx)
	if err != nil {
		return err
	}
	for _, minerInfo := range miners {
		epp, err := NewWinningPoStProver(m.api, m.gatewayNode, minerInfo, m.verifier)
		if err != nil {
			log.Errorf("create WinningPoStProver failed for [%v], err: %v", minerInfo.Addr.String(), err)
			continue
		}
		m.minerWPPMap[minerInfo.Addr] = &minerWPP{epp: epp, account: minerInfo.Name, isMining: true}
	}

	m.stop = make(chan struct{})
	go m.mine(context.TODO())
	go m.SyncStatus(ctx)
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

func (m *Miner) hasMinersNeedMining() bool {
	if len(m.minerWPPMap) == 0 {
		return false
	}

	for _, mw := range m.minerWPPMap {
		if mw.isMining {
			return true
		}
	}

	return false
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

		log.Infow("sync status", "HeightDiff", m.st.heightDiff, "err:", m.st.err)

		// if there is no miner to be mined, wait
		if !m.hasMinersNeedMining() {
			log.Warn("no miner is configured for mining, please check ... ")
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
				tMining := mining
				tAddr := addr

				go func() {
					defer wg.Done()

					// set timeout for miner once
					tCtx, tCtxCancel := context.WithTimeout(ctx, m.mineTimeout)
					defer tCtxCancel()

					resChan, err := m.mineOne(tCtx, base, tMining.account, tAddr, tMining.epp)
					if err != nil { // ToDo retry or continue minerLoop ? currently err is always nil
						log.Errorf("mining block failed for %s: %+v", tAddr.String(), err)
						return
					}

					// waiting for mining results
					select {
					case <-tCtx.Done():
						log.Errorf("mining timeout for %s", tAddr.String())
						if len(tMining.err) >= DefaultMaxErrCounts {
							tMining.err = tMining.err[:DefaultMaxErrCounts-2]
						}
						tMining.err = append(tMining.err, time.Now().Format("2006-01-02 15:04:05 ")+"mining timeout!")
						return
					case res := <-resChan:
						if res != nil && res.winner != nil {
							winPoSts = append(winPoSts, res) //nolint:staticcheck
						} else if res.err != nil {
							if len(tMining.err) > DefaultMaxErrCounts {
								tMining.err = tMining.err[:DefaultMaxErrCounts-2]
							}
							tMining.err = append(tMining.err, time.Now().Format("2006-01-02 15:04:05 ")+res.err.Error())
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
				b, err := m.createBlock(ctx, base, tRes.addr, tRes.waddr, tRes.ticket, tRes.winner, tRes.bvals, tRes.postProof, msgs[idx])
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

					has := m.blockRecord.Has(bm.Header.Miner, uint64(bm.Header.Height))
					if has {
						log.Warnw("Created a block at the same height as another block we've created", "height", bm.Header.Height, "miner", bm.Header.Miner, "parents", bm.Header.Parents)
						return
					}

					err = m.blockRecord.MarkAsProduced(bm.Header.Miner, uint64(bm.Header.Height))
					if err != nil {
						log.Errorf("failed to write db: %s", err)
					}

					if err := m.api.SyncSubmitBlock(ctx, bm); err != nil {
						log.Errorf("failed to submit newly mined block: %s", err)
					}
				}(b)
			}

			// ToDo Under normal circumstances, when the block is created in a cycle,
			// the block is broadcast at the time of (timestamp),
			// and the latest block is often not received directly from the next round,
			// resulting in  lastbase==base staying in the previous cycle,
			// so that it will be broadcast again. Jump one cycle (L280), miss a cycle and possibly produce blocks.

			nextRound := time.Unix(int64(blks[0].Header.Timestamp)+int64(build.PropagationDelaySecs), 0)

			select {
			case <-build.Clock.After(build.Clock.Until(nextRound)):
			case <-m.stop:
				stopping := m.stopping
				m.stop = nil
				m.stopping = nil
				close(stopping)
				return
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
	waddr     address.Address
	ticket    *types.Ticket
	winner    *types.ElectionProof
	bvals     []types.BeaconEntry
	postProof []proof2.PoStProof
	dur       time.Duration
	timetable miningTimetable
	err       error
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
func (m *Miner) mineOne(ctx context.Context, base *MiningBase, account string, addr address.Address, epp chain.WinningPoStProver) (<-chan *winPoStRes, error) {
	log.Infow("attempting to mine a block", "tipset", types.LogCids(base.TipSet.Cids()), "miner", addr)

	out := make(chan *winPoStRes)

	go func() {
		start := build.Clock.Now()

		round := base.TipSet.Height() + base.NullRounds + 1

		mbi, err := m.api.MinerGetBaseInfo(ctx, addr, round, base.TipSet.Key())
		if err != nil {
			log.Errorf("failed to get mining base info: %w, miner: %s", err, addr)
			out <- &winPoStRes{addr: addr, err: err}
			return
		}
		if mbi == nil {
			log.Infow("get nil MinerGetBaseInfo", "miner", addr)
			out <- &winPoStRes{addr: addr, winner: nil}
			return
		}
		if !mbi.EligibleForMining {
			// slashed or just have no power yet
			log.Warnw("slashed or just have no power yet", "miner", addr)
			out <- &winPoStRes{addr: addr, winner: nil}
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
			out <- &winPoStRes{addr: addr, err: err}
			return
		}

		var sign chain.SignFunc
		if _, ok := m.minerWPPMap[addr]; ok {
			walletAPI, closer, err := client.NewGatewayRPC(m.gatewayNode)
			if err != nil {
				log.Errorf("create wallet RPC failed: %w", err)
				out <- &winPoStRes{addr: addr, err: err}
				return
			}
			defer closer()
			sign = walletAPI.WalletSign
		} else {
			log.Errorf("[%v] not exist", addr)
			out <- &winPoStRes{addr: addr, err: xerrors.New("miner not exist")}
			return
		}
		winner, err := chain.IsRoundWinner(ctx, round, account, addr, rbase, mbi, sign)
		if err != nil {
			log.Errorf("failed to check for %s if we win next round: %w", addr, err)
			out <- &winPoStRes{addr: addr, err: err}
			return
		}

		if winner == nil {
			log.Infow("not to be winner", "miner", addr)
			out <- &winPoStRes{addr: addr, winner: nil}
			return
		}

		tTicket := build.Clock.Now()

		buf := new(bytes.Buffer)
		if err := addr.MarshalCBOR(buf); err != nil {
			log.Errorf("failed to marshal miner address: %w", err)
			out <- &winPoStRes{addr: addr, err: err}
			return
		}

		r, err := chain.DrawRandomness(rbase.Data, crypto.DomainSeparationTag_WinningPoStChallengeSeed, round, buf.Bytes())
		if err != nil {
			log.Errorf("failed to get randomness for winning post: %w, miner: %s", err, addr)
			out <- &winPoStRes{addr: addr, err: err}
			return
		}

		prand := abi.PoStRandomness(r)

		tSeed := build.Clock.Now()

		postProof, err := epp.ComputeProof(ctx, mbi.Sectors, prand)
		if err != nil {
			log.Errorf("failed to compute winning post proof: %w, miner: %s", err, addr)
			out <- &winPoStRes{addr: addr, err: err}
			return
		}

		tProof := build.Clock.Now()

		dur := build.Clock.Now().Sub(start)
		tt := miningTimetable{
			tStart: start, tMBI: tMBI, tDrand: tDrand, tPowercheck: tPowercheck, tTicket: tTicket, tSeed: tSeed, tProof: tProof,
		}
		out <- &winPoStRes{addr: addr, waddr: mbi.WorkerKey, ticket: ticket, winner: winner, bvals: bvals, postProof: postProof, dur: dur, timetable: tt}
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

	input := new(bytes.Buffer)
	drp := &core.DrawRandomParams{
		Rbase:   brand.Data,
		Pers:    crypto.DomainSeparationTag_TicketProduction,
		Round:   round - build.TicketRandomnessLookback,
		Entropy: buf.Bytes(),
	}
	err := drp.MarshalCBOR(input)
	if err != nil {
		return nil, xerrors.Errorf("failed to marshal randomness: %w", err)
	}
	//input, err := chain.DrawRandomness(brand.Data, crypto.DomainSeparationTag_TicketProduction, round-build.TicketRandomnessLookback, buf.Bytes())
	//if err != nil {
	//	return nil, err
	//}

	var sign chain.SignFunc
	accout := ""
	if val, ok := m.minerWPPMap[addr]; ok {
		accout = val.account
		walletAPI, closer, err := client.NewGatewayRPC(m.gatewayNode)
		if err != nil {
			log.Errorf("create wallet RPC failed: %w", err)
			return nil, err
		}
		defer closer()
		sign = walletAPI.WalletSign
	} else {
		log.Errorf("[%v] not exist", addr)
		return nil, xerrors.New("miner not exist")
	}

	vrfOut, err := chain.ComputeVRF(ctx, sign, accout, mbi.WorkerKey, input.Bytes())
	if err != nil {
		return nil, err
	}

	return &types.Ticket{
		VRFProof: vrfOut,
	}, nil
}

func (m *Miner) createBlock(ctx context.Context, base *MiningBase, addr, waddr address.Address, ticket *types.Ticket,
	eproof *types.ElectionProof, bvals []types.BeaconEntry, wpostProof []proof2.PoStProof, msgs []*types.SignedMessage) (*types.BlockMsg, error) {
	uts := base.TipSet.MinTimestamp() + build.BlockDelaySecs*(uint64(base.NullRounds)+1)

	nheight := base.TipSet.Height() + base.NullRounds + 1

	// why even return this? that api call could just submit it for us
	blockMsg, err := m.api.MinerCreateBlock(context.TODO(), &api.BlockTemplate{
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

	if err != nil {
		return blockMsg, err
	}

	// ToDo check if BlockHeader is signed
	if blockMsg.Header.BlockSig == nil {
		var sign chain.SignFunc
		account := ""
		if val, ok := m.minerWPPMap[addr]; ok {
			account = val.account
			walletAPI, closer, err := client.NewGatewayRPC(m.gatewayNode)
			if err != nil {
				log.Errorf("create wallet RPC failed: %w", err)
				return nil, err
			}
			defer closer()
			sign = walletAPI.WalletSign
		} else {
			log.Errorf("[%v] not exist", addr)
			return nil, xerrors.New("miner not exist")
		}

		nosigbytes, err := blockMsg.Header.SigningBytes()
		if err != nil {
			return nil, xerrors.Errorf("failed to get SigningBytes: %v", err)
		}

		sig, err := sign(ctx, account, waddr, nosigbytes, core.MsgMeta{
			Type: core.MTBlock,
		})
		if err != nil {
			return nil, xerrors.Errorf("failed to sign new block: %v", err)
		}
		blockMsg.Header.BlockSig = sig
	}

	return blockMsg, err
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
		mining.err = []string{}
		return nil
	}

	return xerrors.Errorf("%s not exist", addr)
}

func (m *Miner) UpdateAddress(ctx context.Context, skip, limit int64) ([]dtypes.MinerInfo, error) {
	miners, err := m.minerManager.Update(ctx, skip, limit)
	if err != nil {
		return nil, err
	}

	// update minerWPPMap
	m.lkWPP.Lock()
	m.minerWPPMap = make(map[address.Address]*minerWPP)
	for _, minerInfo := range miners {
		epp, err := NewWinningPoStProver(m.api, m.gatewayNode, minerInfo, m.verifier)
		if err != nil {
			log.Errorf("create WinningPoStProver failed for [%v], err: %v", minerInfo.Addr.String(), err)
			continue
		}

		m.minerWPPMap[minerInfo.Addr] = &minerWPP{epp: epp, account: minerInfo.Name, isMining: true}
	}
	m.lkWPP.Unlock()

	return miners, nil
}

func (m *Miner) ListAddress(ctx context.Context) ([]dtypes.MinerInfo, error) {
	m.lkWPP.Lock()
	defer m.lkWPP.Unlock()

	return m.minerManager.List(ctx)
}

func (m *Miner) StatesForMining(ctx context.Context, addrs []address.Address) ([]dtypes.MinerState, error) {
	m.lkWPP.Lock()
	defer m.lkWPP.Unlock()

	res := make([]dtypes.MinerState, 0)
	if len(addrs) > 0 {
		for _, addr := range addrs {
			if val, ok := m.minerWPPMap[addr]; ok {
				res = append(res, dtypes.MinerState{Addr: addr, IsMining: val.isMining, Err: val.err})
			}
		}
	} else {
		for k, v := range m.minerWPPMap {
			res = append(res, dtypes.MinerState{Addr: k, IsMining: v.isMining, Err: v.err})
		}
	}

	return res, nil
}

func (m *Miner) winCountInRound(ctx context.Context, account string, mAddr address.Address, api chain.SignFunc, epoch abi.ChainEpoch) (*types.ElectionProof, error) {
	ts, err := m.api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(epoch), types.EmptyTSK)
	if err != nil {
		log.Error("chain get tipset by height error", err)
		return nil, err
	}

	mbi, err := m.api.MinerGetBaseInfo(ctx, mAddr, ts.Height()+1, ts.Key())
	if err != nil {
		log.Error("miner get base info", err)
		return nil, err
	}

	if mbi == nil {
		return nil, xerrors.Errorf("can't find base info on chain, addr %s should be a new miner or no sector found before chain finality", mAddr.String())
	}

	rbase := mbi.PrevBeaconEntry
	if len(mbi.BeaconEntries) > 0 {
		rbase = mbi.BeaconEntries[len(mbi.BeaconEntries)-1]
	}

	return chain.IsRoundWinner(ctx, ts.Height()+1, account, mAddr, rbase, mbi, api)
}

func (m *Miner) CountWinners(ctx context.Context, addrs []address.Address, start abi.ChainEpoch, end abi.ChainEpoch) ([]dtypes.CountWinners, error) {
	res := make([]dtypes.CountWinners, 0)
	wg := sync.WaitGroup{}

	mAddrs := make([]address.Address, 0)
	m.lkWPP.Lock()
	if len(addrs) > 0 {
		for _, addr := range addrs {
			if _, ok := m.minerWPPMap[addr]; ok {
				mAddrs = append(mAddrs, addr)
			} else {
				res = append(res, dtypes.CountWinners{Msg: "miner not exist", Miner: addr})
			}
		}
	} else {
		for k := range m.minerWPPMap {
			mAddrs = append(mAddrs, k)
		}
	}
	m.lkWPP.Unlock()

	if len(mAddrs) > 0 {
		wg.Add(len(mAddrs))
		for _, addr := range mAddrs {
			tAddr := addr

			go func() {
				defer wg.Done()

				winInfo := make([]dtypes.SimpleWinInfo, 0)
				totalWinCount := int64(0)

				var sign chain.SignFunc = nil
				account := ""
				if val, ok := m.minerWPPMap[tAddr]; ok {
					account = val.account
					walletAPI, closer, err := client.NewGatewayRPC(m.gatewayNode)
					if err != nil {
						log.Errorf("[%v] create wallet RPC failed: %w", tAddr, err)
						res = append(res, dtypes.CountWinners{Msg: err.Error(), Miner: tAddr})
						return
					}
					defer closer()
					sign = walletAPI.WalletSign
				} else {
					res = append(res, dtypes.CountWinners{Msg: "miner not exist", Miner: tAddr})
					return
				}

				wgWin := sync.WaitGroup{}
				for epoch := start; epoch <= end; epoch++ {
					wgWin.Add(1)

					go func(epoch abi.ChainEpoch) {
						defer wgWin.Done()

						winner, err := m.winCountInRound(ctx, account, tAddr, sign, epoch)
						if err != nil {
							log.Errorf("generate winner met error %w", err)
							return
						}

						if winner != nil {
							totalWinCount += winner.WinCount
							winInfo = append(winInfo, dtypes.SimpleWinInfo{Epoch: epoch + 1, WinCount: winner.WinCount})
						}
					}(epoch)
				}
				wgWin.Wait()
				res = append(res, dtypes.CountWinners{Miner: tAddr, TotalWinCount: totalWinCount, WinEpochList: winInfo})
			}()
		}
		wg.Wait()
	}

	return res, nil
}

func (m *Miner) SyncStatus(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)

	log.Info("check sync start ...")
	for {
		select {
		case <-ticker.C:
			{
				for {
					state, err := m.api.SyncState(ctx)
					if err != nil {
						m.st = syncStatus{heightDiff: -1, err: err}
						break
					}

					if len(state.ActiveSyncs) == 0 {
						time.Sleep(time.Second)
						continue
					}

					head, err := m.api.ChainHead(ctx)
					if err != nil {
						m.st = syncStatus{heightDiff: -1, err: err}
						break
					}

					working := -1
					for i, ss := range state.ActiveSyncs {
						switch ss.Stage {
						case api.StageSyncComplete:
						default:
							working = i
						case api.StageIdle:
							// not complete, not actively working
						}
					}

					if working == -1 {
						working = len(state.ActiveSyncs) - 1
					}

					ss := state.ActiveSyncs[working]
					var heightDiff int64 = -1
					if ss.Target != nil {
						heightDiff = int64(ss.Target.Height() - head.Height())
					}

					if time.Now().Unix()-int64(head.MinTimestamp()) < int64(build.BlockDelaySecs) {
						heightDiff = 0
					}
					m.st = syncStatus{heightDiff: heightDiff}
					break
				}
			}
		case <-ctx.Done():
			log.Info("check sync exit ...")
			return
		}
	}
}
