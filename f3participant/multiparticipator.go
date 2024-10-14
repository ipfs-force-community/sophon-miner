package f3participant

import (
	"context"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	v1api "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	"github.com/ipfs-force-community/sophon-miner/node/modules/helpers"
	miner_manager "github.com/ipfs-force-community/sophon-miner/node/modules/miner-manager"
	"github.com/jpillora/backoff"
	"go.uber.org/fx"
)

const (
	// maxCheckProgressAttempts defines the maximum number of failed attempts
	// before we abandon the current lease and restart the participation process.
	//
	// The default backoff takes 12 attempts to reach a maximum delay of 1 minute.
	// Allowing for 13 failures results in approximately 2 minutes of backoff since
	// the lease was granted. Given a lease validity of up to 5 instances, this means
	// we would give up on checking the lease during its mid-validity period;
	// typically when we would try to renew the participation ticket. Hence, the value
	// to 13.
	checkProgressMaxAttempts = 13

	// F3LeaseTerm The number of instances the miner will attempt to lease from nodes.
	F3LeaseTerm = 5
)

type MultiParticipant struct {
	participants map[address.Address]*Participant
	minerManager miner_manager.MinerManageAPI

	newParticipant func(context.Context, address.Address) *Participant
	lk             sync.Mutex
}

func NewMultiParticipant(lc fx.Lifecycle,
	mctx helpers.MetricsCtx,
	node v1api.FullNode,
	minerManager miner_manager.MinerManageAPI,
) (*MultiParticipant, error) {
	newParticipant := func(ctx context.Context, participant address.Address) *Participant {
		return NewParticipant(
			ctx,
			node,
			participant,
			&backoff.Backoff{
				Min:    1 * time.Second,
				Max:    1 * time.Minute,
				Factor: 1.5,
			},
			checkProgressMaxAttempts,
			F3LeaseTerm,
		)
	}

	miners, err := minerManager.List(mctx)
	if err != nil {
		return nil, err
	}

	mp := &MultiParticipant{
		participants:   make(map[address.Address]*Participant),
		minerManager:   minerManager,
		newParticipant: newParticipant,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			for _, minerInfo := range miners {
				if !minerManager.IsOpenMining(ctx, minerInfo.Addr) {
					continue
				}
				p := newParticipant(ctx, minerInfo.Addr)
				mp.participants[minerInfo.Addr] = p
				if err := p.Start(ctx); err != nil {
					return err
				}
			}
			go mp.MonitorMiner(ctx)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			for _, p := range mp.participants {
				if err := p.Stop(ctx); err != nil {
					log.Errorf("failed to stop participant %v, err: %v", p, err)
				}
			}
			return nil
		},
	})

	return mp, nil
}

func (mp *MultiParticipant) addParticipant(ctx helpers.MetricsCtx, participant address.Address) error {
	mp.lk.Lock()
	defer mp.lk.Unlock()

	p, ok := mp.participants[participant]
	if ok {
		return nil
	}

	mp.participants[participant] = mp.newParticipant(ctx, participant)
	p.Start(ctx)
	log.Infof("add participate %s", participant)

	return nil
}

func (mp *MultiParticipant) removeParticipant(ctx context.Context, participant address.Address) error {
	mp.lk.Lock()
	defer mp.lk.Unlock()

	p, ok := mp.participants[participant]
	if !ok {
		return nil
	}
	delete(mp.participants, participant)
	log.Infof("remove participate %s", participant)

	return p.Stop(ctx)
}

func (mp *MultiParticipant) MonitorMiner(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			miners, err := mp.minerManager.List(ctx)
			if err != nil {
				log.Errorf("failed to list miners: %v", err)
				continue
			}

			for _, minerInfo := range miners {
				if !mp.minerManager.IsOpenMining(ctx, minerInfo.Addr) {
					if err := mp.removeParticipant(ctx, minerInfo.Addr); err != nil {
						log.Errorf("failed to remove participate %s: %v", minerInfo.Addr, err)
					}
					continue
				}

				if _, ok := mp.participants[minerInfo.Addr]; !ok {
					if err := mp.addParticipant(ctx, minerInfo.Addr); err != nil {
						log.Errorf("failed to add participate %s: %v", minerInfo.Addr, err)
					}
				}
			}
		}
	}
}
