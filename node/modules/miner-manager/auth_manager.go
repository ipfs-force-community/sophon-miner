package miner_manager

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/go-address"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/venus-auth/jwtclient"

	"github.com/filecoin-project/venus-miner/types"
)

const CoMinersLimit = 20000

var (
	ErrNotFound = fmt.Errorf("not found")
)

var log = logging.Logger("auth-miners")

type MinerManage struct {
	authClient jwtclient.IAuthClient

	miners map[address.Address]*types.MinerInfo
	lk     sync.Mutex
}

func NewVenusAuth(url, token string) func() (jwtclient.IAuthClient, error) {
	return func() (jwtclient.IAuthClient, error) {
		return jwtclient.NewAuthClient(url, token)
	}
}

func NewMinerManager(authClient jwtclient.IAuthClient) (MinerManageAPI, error) {
	m := &MinerManage{authClient: authClient}
	_, err := m.Update(context.TODO(), 0, 0)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *MinerManage) Has(ctx context.Context, mAddr address.Address) bool {
	m.lk.Lock()
	defer m.lk.Unlock()

	_, ok := m.miners[mAddr]
	return ok
}

func (m *MinerManage) Get(ctx context.Context, mAddr address.Address) (*types.MinerInfo, error) {
	m.lk.Lock()
	defer m.lk.Unlock()

	if _, ok := m.miners[mAddr]; ok {
		return m.miners[mAddr], nil
	}

	return nil, ErrNotFound
}

func (m *MinerManage) IsOpenMining(ctx context.Context, mAddr address.Address) bool {
	m.lk.Lock()
	defer m.lk.Unlock()

	if _, ok := m.miners[mAddr]; ok {
		return m.miners[mAddr].OpenMining
	}

	return false
}

func (m *MinerManage) OpenMining(ctx context.Context, mAddr address.Address) (*types.MinerInfo, error) {
	m.lk.Lock()
	defer m.lk.Unlock()

	if minerInfo, ok := m.miners[mAddr]; ok {
		_, err := m.authClient.UpsertMiner(ctx, minerInfo.Name, minerInfo.Addr.String(), true)
		if err != nil {
			return nil, err
		}
		minerInfo.OpenMining = true
		return minerInfo, nil
	}

	return nil, ErrNotFound
}

func (m *MinerManage) CloseMining(ctx context.Context, mAddr address.Address) error {
	m.lk.Lock()
	defer m.lk.Unlock()

	if minerInfo, ok := m.miners[mAddr]; ok {
		_, err := m.authClient.UpsertMiner(ctx, minerInfo.Name, minerInfo.Addr.String(), false)
		if err != nil {
			return err
		}
		minerInfo.OpenMining = false
		return nil
	}

	return ErrNotFound
}

func (m *MinerManage) List(ctx context.Context) (map[address.Address]*types.MinerInfo, error) {
	m.lk.Lock()
	defer m.lk.Unlock()

	return m.miners, nil
}

func (m *MinerManage) Update(ctx context.Context, skip, limit int64) (map[address.Address]*types.MinerInfo, error) {
	m.lk.Lock()
	defer m.lk.Unlock()

	if limit == 0 {
		limit = CoMinersLimit
	}

	users, err := m.authClient.ListUsersWithMiners(ctx, skip, limit, 0)
	if err != nil {
		return nil, err
	}

	miners := make(map[address.Address]*types.MinerInfo, 0)
	for _, user := range users {
		if user.State != 1 {
			log.Warnf("user: %s state is disabled, it's miners won't be updated", user.Name)
			continue
		}

		for _, miner := range user.Miners {
			miners[miner.Miner] = &types.MinerInfo{
				Addr:       miner.Miner,
				Id:         user.Id,
				Name:       miner.User,
				OpenMining: miner.OpenMining,
			}
		}
	}
	m.miners = miners

	return m.miners, nil
}

var _ MinerManageAPI = &MinerManage{}
