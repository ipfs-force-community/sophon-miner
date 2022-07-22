package miner

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-miner/api/client"
	"github.com/filecoin-project/venus-miner/types"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
)

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

func (m *Miner) winCountInRound(ctx context.Context, account string, mAddr address.Address, api SignFunc, epoch abi.ChainEpoch) (*sharedTypes.ElectionProof, error) {
	ts, err := m.api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(epoch), sharedTypes.EmptyTSK)
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
						log.Errorf("[%v] create wallet RPC failed: %s", tAddr, err)
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
							log.Errorf("generate winner met error %s", err)
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
