package miner_manager

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/go-resty/resty/v2"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/venus-miner/types"
)

const CoMinersLimit = 200

var log = logging.Logger("auth-miners")

type MinerManage struct {
	cli   *resty.Client
	token string

	miners []types.MinerInfo
	lk     sync.Mutex
}

func NewMinerManager(url, token string) func() (MinerManageAPI, error) {
	return func() (MinerManageAPI, error) {
		cli := resty.New().SetHostURL(url).SetHeader("Accept", "application/json")
		m := &MinerManage{cli: cli, token: token}

		miners, err := m.Update(context.TODO(), 0, 0)
		if err != nil {
			return nil, err
		}

		m.miners = miners
		return m, nil
	}
}

func (m *MinerManage) Has(ctx context.Context, addr address.Address) bool {
	m.lk.Lock()
	defer m.lk.Unlock()

	for _, miner := range m.miners {
		if miner.Addr.String() == addr.String() {
			return true
		}
	}

	return false
}

func (m *MinerManage) Get(ctx context.Context, addr address.Address) *types.MinerInfo {
	m.lk.Lock()
	defer m.lk.Unlock()

	for k := range m.miners {
		if m.miners[k].Addr.String() == addr.String() {
			return &m.miners[k]
		}
	}

	return nil
}

func (m *MinerManage) List(ctx context.Context) ([]types.MinerInfo, error) {
	m.lk.Lock()
	defer m.lk.Unlock()

	return m.miners, nil
}

func (m *MinerManage) Update(ctx context.Context, skip, limit int64) ([]types.MinerInfo, error) {
	if limit == 0 {
		limit = CoMinersLimit
	}
	users, err := m.listUsers(skip, limit)
	if err != nil {
		return nil, err
	}
	var mInfos = make([]types.MinerInfo, 0)

	for _, u := range users {
		miners, err := m.listMiners(u.Name)
		if err != nil {
			log.Errorf("list user:%s minres failed:%s", u.Name, err.Error())
			continue
		}
		for _, val := range miners {
			addr, err := address.NewFromString(val.Miner)
			if err != nil {
				log.Errorf("invalid user:%s miner:%s, %s", u.Name, val.Miner, err.Error())
				continue

			}
			mInfos = append(mInfos, types.MinerInfo{
				Addr: addr,
				Id:   u.ID,
				Name: u.Name,
			})
		}
	}

	m.miners = mInfos

	return m.miners, nil
}

func (m *MinerManage) Count(ctx context.Context) int {
	m.lk.Lock()
	defer m.lk.Unlock()

	return len(m.miners)
}

func (m *MinerManage) listUsers(skip, limit int64) ([]*types.User, error) {
	var users []*types.User
	resp, err := m.cli.R().SetQueryParams(map[string]string{
		"skip":  strconv.FormatInt(skip, 10),
		"limit": strconv.FormatInt(limit, 10),
		"state": "1",
	}).SetResult(&users).SetError(&apiErr{}).Get("/user/list")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusOK {
		return *(resp.Result().(*[]*types.User)), nil
	}
	return nil, resp.Error().(*apiErr).Err()
}

func (m *MinerManage) listMiners(user string) ([]*types.Miner, error) {
	var res []*types.Miner
	resp, err := m.cli.R().SetQueryParams(map[string]string{"user": user}).
		SetResult(&res).SetError(&apiErr{}).Get("/user/miner/list")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusOK {
		return *(resp.Result().(*[]*types.Miner)), nil
	}
	return nil, resp.Error().(*apiErr).Err()
}

type apiErr struct {
	Error string `json:"error"`
}

func (err *apiErr) Err() error {
	return fmt.Errorf(err.Error)
}

var _ MinerManageAPI = &MinerManage{}
