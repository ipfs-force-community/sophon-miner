package miner

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"

	proof2 "github.com/filecoin-project/specs-actors/v2/actors/runtime/proof"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"

	"github.com/filecoin-project/venus-miner/api/client"
	"github.com/filecoin-project/venus-miner/build"
	"github.com/filecoin-project/venus-miner/lib/journal"
	"github.com/filecoin-project/venus-miner/node/config"
	miner_manager "github.com/filecoin-project/venus-miner/node/modules/miner-manager"
	"github.com/filecoin-project/venus-miner/node/modules/slashfilter"
	"github.com/filecoin-project/venus-miner/types"

	"github.com/filecoin-project/venus/pkg/chain"
	"github.com/filecoin-project/venus/pkg/constants"
	v1api "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	types2 "github.com/filecoin-project/venus/venus-shared/types"
)

var log = logging.Logger("miner")

// Journal event types.
const (
	evtTypeBlockMined = iota
)

var DefaultMaxErrCounts = 20

// waitFunc is expected to pace block mining at the configured network rate.
//
// baseTime is the timestamp of the mining base, i.e. the timestamp
// of the tipset we're planning to construct upon.
//
// Upon each mining loop iteration, the returned callback is called reporting
// whether we mined a block in this round or not.
type waitFunc func(ctx context.Context, baseTime uint64) (func(bool, abi.ChainEpoch, error), abi.ChainEpoch, error)

func randTimeOffset(width time.Duration) time.Duration {
	buf := make([]byte, 8)
	rand.Reader.Read(buf) //nolint:errcheck
	val := time.Duration(binary.BigEndian.Uint64(buf) % uint64(width))

	return val - (width / 2)
}

// NewMiner instantiates a miner with a concrete WinningPoStProver and a miner
// address (which can be different from the worker's address).
func NewMiner(api v1api.FullNode,
	gtNode *config.GatewayNode,
	minerManager miner_manager.MinerManageAPI,
	sf slashfilter.SlashFilterAPI,
	j journal.Journal) *Miner {
	networkParams, err := api.StateGetNetworkParams(context.TODO())
	if err != nil {
		return nil
	}
	if networkParams.BlockDelaySecs < 30 {
		build.MinerOnceTimeout = time.Millisecond * 2800
	}

	miner := &Miner{
		api:           api,
		networkParams: networkParams,
		gatewayNode:   gtNode,
		waitFunc: func(ctx context.Context, baseTime uint64) (func(bool, abi.ChainEpoch, error), abi.ChainEpoch, error) {
			// wait around for half the block time in case other parents come in
			//
			// if we're mining a block in the past via catch-up/rush mining,
			// such as when recovering from a network halt, this sleep will be
			// for a negative duration, and therefore **will return
			// immediately**.
			//
			// the result is that we WILL NOT wait, therefore fast-forwarding
			// and thus healing the chain by backfilling it with null rounds
			// rapidly.
			deadline := baseTime + build.PropagationDelaySecs
			baseT := time.Unix(int64(deadline), 0)

			baseT = baseT.Add(randTimeOffset(time.Second))

			build.Clock.Sleep(build.Clock.Until(baseT))

			return func(bool, abi.ChainEpoch, error) {}, 0, nil
		},

		sf: sf,

		evtTypes: [...]journal.EventType{
			evtTypeBlockMined: j.RegisterEventType("miner", "block_mined"),
		},
		journal: j,

		minerManager: minerManager,
		minerWPPMap:  make(map[address.Address]*minerWPP),
	}

	return miner
}

type syncStatus struct {
	heightDiff int64
	err        error
}

type minerWPP struct {
	account  string
	epp      WinningPoStProver
	isMining bool
	err      []string
}

type Miner struct {
	api           v1api.FullNode
	networkParams *types2.NetworkParams

	gatewayNode *config.GatewayNode

	lk       sync.Mutex
	stop     chan struct{}
	stopping chan struct{}

	waitFunc waitFunc

	lastWork *MiningBase

	sf slashfilter.SlashFilterAPI

	evtTypes [1]journal.EventType
	journal  journal.Journal

	st syncStatus

	lkWPP        sync.Mutex
	minerWPPMap  map[address.Address]*minerWPP
	minerManager miner_manager.MinerManageAPI
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
		epp, err := NewWinningPoStProver(m.api, m.gatewayNode, minerInfo)
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

// Stop stops the mining operation. It is not idempotent, and multiple adjacent
// calls to Stop will fail.
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

// mine runs the mining loop. It performs the following:
//
//  1.  Queries our current best currently-known mining candidate (tipset to
//      build upon).
//  2.  Waits until the propagation delay of the network has elapsed (currently
//      6 seconds). The waiting is done relative to the timestamp of the best
//      candidate, which means that if it's way in the past, we won't wait at
//      all (e.g. in catch-up or rush mining).
//  3.  After the wait, we query our best mining candidate. This will be the one
//      we'll work with.
//  4.  Sanity check that we _actually_ have a new mining base to mine on. If
//      not, wait one epoch + propagation delay, and go back to the top.
//  5.  We attempt to mine a block, by calling mineOne (refer to godocs). This
//      method will either return a block if we were eligible to mine, or nil
//      if we weren't.
//  6a. If we mined a block, we update our state and push it out to the network
//      via gossipsub.
//  6b. If we didn't mine a block, we consider this to be a nil round on top of
//      the mining base we selected. If other miner or miners on the network
//      were eligible to mine, we will receive their blocks via gossipsub and
//      we will select that tipset on the next iteration of the loop, thus
//      discarding our null round.
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
			_, err = m.api.StateGetBeaconEntry(ctx, prebase.TipSet.Height()+prebase.NullRounds+1)
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
			if !m.niceSleep(time.Duration(m.networkParams.BlockDelaySecs) * time.Second) {
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
					tCtx, tCtxCancel := context.WithTimeout(ctx, build.MinerOnceTimeout)
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
			var blks []*types2.BlockMsg
			for idx, res := range winPoSts {
				tRes := res
				// TODO: winning post proof
				var b *types2.BlockMsg
				if len(msgs) > idx {
					b, err = m.createBlock(ctx, base, tRes.addr, tRes.waddr, tRes.ticket, tRes.winner, tRes.bvals, tRes.postProof, msgs[idx])
				} else {
					b, err = m.createBlock(ctx, base, tRes.addr, tRes.waddr, tRes.ticket, tRes.winner, tRes.bvals, tRes.postProof, []*types2.SignedMessage{})
				}
				if err != nil {
					log.Errorf("failed to create block: %w", err)
					continue
				}
				blks = append(blks, b)

				tCreateBlock := build.Clock.Now()
				dur := tCreateBlock.Sub(tRes.timetable.tStart)
				parentMiners := make([]address.Address, len(base.TipSet.Blocks()))
				for i, header := range base.TipSet.Blocks() {
					parentMiners[i] = header.Miner
				}
				log.Infow("mined new block", "cid", b.Cid(), "height", b.Header.Height, "miner", b.Header.Miner, "parents", parentMiners, "took", dur)

				if dur > time.Second*time.Duration(m.networkParams.BlockDelaySecs) {
					log.Warnw("CAUTION: block production took longer than the block delay. Your computer may not be fast enough to keep up",
						"miner", tRes.addr,
						"tMinerBaseInfo ", tRes.timetable.tMBI.Sub(tRes.timetable.tStart),
						"tTicket ", tRes.timetable.tTicket.Sub(tRes.timetable.tMBI),
						"tIsWinner ", tRes.timetable.tIsWinner.Sub(tRes.timetable.tTicket),
						"tSeed ", tRes.timetable.tSeed.Sub(tRes.timetable.tIsWinner),
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

			if len(blks) > 0 {
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
					go func(bm *types2.BlockMsg) {
						if exists, err := m.sf.HasBlock(ctx, bm.Header); err != nil {
							log.Errorf("<!!> SLASH FILTER ERROR: %s", err)
							return
						} else if exists {
							log.Error("<!!> SLASH FILTER ERROR: double-fork mining faults")
							return
						}

						if err := m.sf.MinedBlock(ctx, bm.Header, base.TipSet.Height()+base.NullRounds); err != nil {
							log.Errorf("<!!> SLASH FILTER ERROR: %s", err)
							return
						}

						if err := m.api.SyncSubmitBlock(ctx, bm); err != nil {
							log.Errorf("failed to submit newly mined block: %s", err)
							return
						}

						if err := m.sf.PutBlock(ctx, bm.Header, base.TipSet.Height()+base.NullRounds); err != nil {
							log.Errorf("<!!> SLASH FILTER ERROR: %s", err)
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
			}
		} else {
			base.NullRounds++
			log.Info("no block and increase nullround")
			// Wait until the next epoch, plus the propagation delay, so a new tipset
			// has enough time to form.
			//
			// See:  https://github.com/filecoin-project/venus-miner/issues/1845
			nextRound := time.Unix(int64(base.TipSet.MinTimestamp()+m.networkParams.BlockDelaySecs*uint64(base.NullRounds))+int64(build.PropagationDelaySecs), 0)

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

// MiningBase is the tipset on top of which we plan to construct our next block.
// Refer to godocs on GetBestMiningCandidate.
type MiningBase struct {
	TipSet     *types2.TipSet
	NullRounds abi.ChainEpoch
}

// GetBestMiningCandidate implements the fork choice rule from a miner's
// perspective.
//
// It obtains the current chain head (HEAD), and compares it to the last tipset
// we selected as our mining base (LAST). If HEAD's weight is larger than
// LAST's weight, it selects HEAD to build on. Else, it selects LAST.
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

		if types2.BigCmp(btsw, ltsw) <= 0 {
			return m.lastWork, nil
		}
	}

	m.lastWork = &MiningBase{TipSet: bts}
	return m.lastWork, nil
}

type miningTimetable struct {
	tStart, tMBI, tTicket, tIsWinner, tSeed, tProof time.Time
}

type winPoStRes struct {
	addr      address.Address
	waddr     address.Address
	ticket    *types2.Ticket
	winner    *types2.ElectionProof
	bvals     []types2.BeaconEntry
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
func (m *Miner) mineOne(ctx context.Context, base *MiningBase, account string, addr address.Address, epp WinningPoStProver) (<-chan *winPoStRes, error) {
	log.Infow("attempting to mine a block", "tipset", types.LogCids(base.TipSet.Cids()), "miner", addr)
	start := build.Clock.Now()

	round := base.TipSet.Height() + base.NullRounds + 1

	out := make(chan *winPoStRes)

	go func() {
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
		log.Infow("mine one", "miner", addr, "get base info", tMBI.Sub(start))

		beaconPrev := mbi.PrevBeaconEntry
		bvals := mbi.BeaconEntries

		log.Infof("Time delta between now and our mining base: %ds (nulls: %d), miner: %s", uint64(build.Clock.Now().Unix())-base.TipSet.MinTimestamp(), base.NullRounds, addr)

		rbase := beaconPrev
		if len(bvals) > 0 {
			rbase = bvals[len(bvals)-1]
		}

		ticket, err := m.computeTicket(ctx, &rbase, base, mbi, addr)
		if err != nil {
			log.Errorf("scratching ticket for %s failed: %s", addr, err.Error())
			out <- &winPoStRes{addr: addr, err: err}
			return
		}

		tTicket := build.Clock.Now()
		log.Infow("mine one", "miner", addr, "compute ticket", tTicket.Sub(tMBI))

		var sign SignFunc
		if _, ok := m.minerWPPMap[addr]; ok {
			walletAPI, closer, err := client.NewGatewayRPC(ctx, m.gatewayNode)
			if err != nil {
				log.Errorf("create wallet RPC failed: %w", err)
				out <- &winPoStRes{addr: addr, err: err}
				return
			}
			defer closer()
			sign = walletAPI.WalletSign
		} else {
			log.Errorf("[%v] not exist", addr)
			out <- &winPoStRes{addr: addr, err: errors.New("miner not exist")}
			return
		}
		winner, err := IsRoundWinner(ctx, round, account, addr, rbase, mbi, sign)
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

		tIsWinner := build.Clock.Now()
		log.Infow("mine one", "miner", addr, "is winner", tIsWinner.Sub(tTicket))

		buf := new(bytes.Buffer)
		if err := addr.MarshalCBOR(buf); err != nil {
			log.Errorf("failed to marshal miner address: %w", err)
			out <- &winPoStRes{addr: addr, err: err}
			return
		}

		r, err := chain.DrawRandomness(rbase.Data, crypto.DomainSeparationTag_TicketProduction, round-constants.TicketRandomnessLookback, buf.Bytes())
		if err != nil {
			log.Errorf("failed to draw randomness: %w, miner: %s", err, addr)
			out <- &winPoStRes{addr: addr, err: err}
			return
		}
		prand := abi.PoStRandomness(r)

		tSeed := build.Clock.Now()
		log.Infow("mine one", "miner", addr, "seed", tSeed.Sub(tIsWinner))

		nv, err := m.api.StateNetworkVersion(ctx, base.TipSet.Key())
		if err != nil {
			log.Errorf("failed to get network version: %w, miner: %s", err, addr)
			out <- &winPoStRes{addr: addr, err: err}
			return
		}

		postProof, err := epp.ComputeProof(ctx, mbi.Sectors, prand, round, nv)
		if err != nil {
			log.Errorf("failed to compute winning post proof: %w, miner: %s", err, addr)
			out <- &winPoStRes{addr: addr, err: err}
			return
		}

		tProof := build.Clock.Now()
		log.Infow("mine one", "miner", addr, "compute proof", tProof.Sub(tSeed))

		dur := build.Clock.Now().Sub(start)
		tt := miningTimetable{
			tStart: start, tMBI: tMBI, tTicket: tTicket, tIsWinner: tIsWinner, tSeed: tSeed, tProof: tProof,
		}
		out <- &winPoStRes{addr: addr, waddr: mbi.WorkerKey, ticket: ticket, winner: winner, bvals: bvals, postProof: postProof, dur: dur, timetable: tt}
		log.Infow("mined new block ( -> Proof)", "took", dur, "miner", addr)
	}()

	return out, nil
}

func (m *Miner) computeTicket(ctx context.Context, brand *types2.BeaconEntry, base *MiningBase, mbi *types2.MiningBaseInfo, addr address.Address) (*types2.Ticket, error) {
	buf := new(bytes.Buffer)
	if err := addr.MarshalCBOR(buf); err != nil {
		return nil, fmt.Errorf("failed to marshal address to cbor: %w", err)
	}

	round := base.TipSet.Height() + base.NullRounds + 1
	if round > m.networkParams.ForkUpgradeParams.UpgradeSmokeHeight {
		buf.Write(base.TipSet.MinTicket().VRFProof)
	}

	input, err := chain.DrawRandomness(brand.Data, crypto.DomainSeparationTag_TicketProduction, round-constants.TicketRandomnessLookback, buf.Bytes())
	if err != nil {
		return nil, err
	}

	var sign SignFunc
	account := ""
	if val, ok := m.minerWPPMap[addr]; ok {
		account = val.account
		walletAPI, closer, err := client.NewGatewayRPC(ctx, m.gatewayNode)
		if err != nil {
			log.Errorf("create wallet RPC failed: %w", err)
			return nil, err
		}
		defer closer()
		sign = walletAPI.WalletSign
	} else {
		log.Errorf("[%v] not exist", addr)
		return nil, errors.New("miner not exist")
	}

	vrfOut, err := ComputeVRF(ctx, sign, account, mbi.WorkerKey, input)
	if err != nil {
		return nil, err
	}

	return &types2.Ticket{
		VRFProof: vrfOut,
	}, nil
}

func (m *Miner) createBlock(ctx context.Context, base *MiningBase, addr, waddr address.Address, ticket *types2.Ticket,
	eproof *types2.ElectionProof, bvals []types2.BeaconEntry, wpostProof []proof2.PoStProof, msgs []*types2.SignedMessage) (*types2.BlockMsg, error) {
	uts := base.TipSet.MinTimestamp() + m.networkParams.BlockDelaySecs*(uint64(base.NullRounds)+1)

	nheight := base.TipSet.Height() + base.NullRounds + 1

	// why even return this? that api call could just submit it for us
	blockMsg, err := m.api.MinerCreateBlock(context.TODO(), &types2.BlockTemplate{
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
		var sign SignFunc
		account := ""
		if val, ok := m.minerWPPMap[addr]; ok {
			account = val.account
			walletAPI, closer, err := client.NewGatewayRPC(ctx, m.gatewayNode)
			if err != nil {
				log.Errorf("create wallet RPC failed: %w", err)
				return nil, err
			}
			defer closer()
			sign = walletAPI.WalletSign
		} else {
			log.Errorf("[%v] not exist", addr)
			return nil, errors.New("miner not exist")
		}

		nosigbytes, err := blockMsg.Header.SignatureData()
		if err != nil {
			return nil, fmt.Errorf("failed to get SigningBytes: %v", err)
		}

		sig, err := sign(ctx, account, waddr, nosigbytes, types2.MsgMeta{
			Type: types2.MTBlock,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to sign new block: %v", err)
		}
		blockMsg.Header.BlockSig = sig
	}

	return blockMsg, err
}

func (m *Miner) ManualStart(ctx context.Context, addrs []address.Address) error {
	m.lkWPP.Lock()
	defer m.lkWPP.Unlock()

	if len(addrs) > 0 {
		for _, addr := range addrs {
			if _, ok := m.minerWPPMap[addr]; ok {
				m.minerWPPMap[addr].isMining = true
			} else {
				return fmt.Errorf("%s not exist", addr)
			}
		}
	} else {
		for k := range m.minerWPPMap {
			m.minerWPPMap[k].isMining = true
		}
	}

	return nil
}

func (m *Miner) ManualStop(ctx context.Context, addrs []address.Address) error {
	m.lkWPP.Lock()
	defer m.lkWPP.Unlock()

	if len(addrs) > 0 {
		for _, addr := range addrs {
			if _, ok := m.minerWPPMap[addr]; ok {
				m.minerWPPMap[addr].isMining = false
			} else {
				return fmt.Errorf("%s not exist", addr)
			}
		}
	} else {
		for k := range m.minerWPPMap {
			m.minerWPPMap[k].isMining = false
		}
	}

	return nil
}

func (m *Miner) UpdateAddress(ctx context.Context, skip, limit int64) ([]types.MinerInfo, error) {
	miners, err := m.minerManager.Update(ctx, skip, limit)
	if err != nil {
		return nil, err
	}

	// update minerWPPMap
	m.lkWPP.Lock()
	m.minerWPPMap = make(map[address.Address]*minerWPP)
	for _, minerInfo := range miners {
		epp, err := NewWinningPoStProver(m.api, m.gatewayNode, minerInfo)
		if err != nil {
			log.Errorf("create WinningPoStProver failed for [%v], err: %v", minerInfo.Addr.String(), err)
			continue
		}

		m.minerWPPMap[minerInfo.Addr] = &minerWPP{epp: epp, account: minerInfo.Name, isMining: true}
	}
	m.lkWPP.Unlock()

	return miners, nil
}

func (m *Miner) ListAddress(ctx context.Context) ([]types.MinerInfo, error) {
	m.lkWPP.Lock()
	defer m.lkWPP.Unlock()

	return m.minerManager.List(ctx)
}

func (m *Miner) StatesForMining(ctx context.Context, addrs []address.Address) ([]types.MinerState, error) {
	m.lkWPP.Lock()
	defer m.lkWPP.Unlock()

	res := make([]types.MinerState, 0)
	if len(addrs) > 0 {
		for _, addr := range addrs {
			if val, ok := m.minerWPPMap[addr]; ok {
				res = append(res, types.MinerState{Addr: addr, IsMining: val.isMining, Err: val.err})
			}
		}
	} else {
		for k, v := range m.minerWPPMap {
			res = append(res, types.MinerState{Addr: k, IsMining: v.isMining, Err: v.err})
		}
	}

	return res, nil
}

func (m *Miner) winCountInRound(ctx context.Context, account string, mAddr address.Address, api SignFunc, epoch abi.ChainEpoch) (*types2.ElectionProof, error) {
	ts, err := m.api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(epoch), types2.EmptyTSK)
	if err != nil {
		return nil, err
	}

	mbi, err := m.api.MinerGetBaseInfo(ctx, mAddr, ts.Height()+1, ts.Key())
	if err != nil {
		return nil, err
	}

	if mbi == nil {
		return nil, fmt.Errorf("can't find base info on chain, addr %s should be a new miner or no sector found before chain finality", mAddr.String())
	}

	if !mbi.EligibleForMining {
		return nil, fmt.Errorf("%s slashed or just have no power yet", mAddr.String())
	}

	rbase := mbi.PrevBeaconEntry
	if len(mbi.BeaconEntries) > 0 {
		rbase = mbi.BeaconEntries[len(mbi.BeaconEntries)-1]
	}

	return IsRoundWinner(ctx, ts.Height()+1, account, mAddr, rbase, mbi, api)
}

func (m *Miner) CountWinners(ctx context.Context, addrs []address.Address, start abi.ChainEpoch, end abi.ChainEpoch) ([]types.CountWinners, error) {
	log.Infof("count winners, addrs: %v, start: %v, end: %v", addrs, start, end)

	ts, err := m.api.ChainHead(ctx)
	if err != nil {
		log.Error("get chain head", err)
		return []types.CountWinners{}, err
	}

	if start > ts.Height() || end > ts.Height() {
		return []types.CountWinners{}, fmt.Errorf("start or end greater than cur tipset height: %v", ts.Height())
	}

	res := make([]types.CountWinners, 0)
	wg := sync.WaitGroup{}

	mAddrs := make([]address.Address, 0)
	m.lkWPP.Lock()
	if len(addrs) > 0 {
		for _, addr := range addrs {
			if _, ok := m.minerWPPMap[addr]; ok {
				mAddrs = append(mAddrs, addr)
			} else {
				res = append(res, types.CountWinners{Msg: "miner not exist", Miner: addr})
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

				winInfo := make([]types.SimpleWinInfo, 0)
				totalWinCount := int64(0)

				var sign SignFunc = nil
				account := ""
				if val, ok := m.minerWPPMap[tAddr]; ok {
					account = val.account
					walletAPI, closer, err := client.NewGatewayRPC(ctx, m.gatewayNode)
					if err != nil {
						log.Errorf("[%v] create wallet RPC failed: %w", tAddr, err)
						res = append(res, types.CountWinners{Msg: err.Error(), Miner: tAddr})
						return
					}
					defer closer()
					sign = walletAPI.WalletSign
				} else {
					res = append(res, types.CountWinners{Msg: "miner not exist", Miner: tAddr})
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
							winInfo = append(winInfo, types.SimpleWinInfo{Epoch: epoch + 1, WinCount: winner.WinCount})
						}
					}(epoch)
				}
				wgWin.Wait()
				res = append(res, types.CountWinners{Miner: tAddr, TotalWinCount: totalWinCount, WinEpochList: winInfo})
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
						case types2.StageSyncComplete:
						default:
							working = i
						case types2.StageIdle:
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

					if time.Now().Unix()-int64(head.MinTimestamp()) < int64(m.networkParams.BlockDelaySecs) {
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
