package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-resty/resty/v2"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage"
)

const CoMinersLimit = 200

var log = logging.Logger("auth-miners")

type MinerManagerForAuth struct {
	cli   *resty.Client
	token string

	miners []dtypes.MinerInfo
	lk     sync.Mutex
}

func NewMinerManager(url, token string) func() (minermanage.MinerManageAPI, error) {
	return func() (minermanage.MinerManageAPI, error) {
		cli := resty.New().SetHostURL(url).SetHeader("Accept", "application/json")
		m := &MinerManagerForAuth{cli: cli, token: token}

		miners, err := m.Update(context.TODO(), 0, 0)
		if err != nil {
			return nil, err
		}

		m.miners = miners
		return m, nil
	}
}

func (m *MinerManagerForAuth) Put(ctx context.Context, mi dtypes.MinerInfo) error {
	m.lk.Lock()
	defer m.lk.Unlock()

	if m.Has(ctx, mi.Addr) {
		log.Warnf("addr %s has exit", mi.Addr)
		return nil
	}

	m.miners = append(m.miners, mi)
	return nil
}

func (m *MinerManagerForAuth) Has(ctx context.Context, addr address.Address) bool {
	m.lk.Lock()
	defer m.lk.Unlock()

	for _, miner := range m.miners {
		if miner.Addr.String() == addr.String() {
			return true
		}
	}

	return false
}

func (m *MinerManagerForAuth) Get(ctx context.Context, addr address.Address) *dtypes.MinerInfo {
	m.lk.Lock()
	defer m.lk.Unlock()

	for k := range m.miners {
		if m.miners[k].Addr.String() == addr.String() {
			return &m.miners[k]
		}
	}

	return nil
}

func (m *MinerManagerForAuth) List(ctx context.Context) ([]dtypes.MinerInfo, error) {
	m.lk.Lock()
	defer m.lk.Unlock()

	return m.miners, nil
}

func (m *MinerManagerForAuth) Update(ctx context.Context, skip, limit int64) ([]dtypes.MinerInfo, error) {
	if limit == 0 {
		limit = CoMinersLimit
	}

	response, err := m.cli.R().SetQueryParams(map[string]string{
		"token": m.token,
		"skip":  fmt.Sprintf("%d", skip),
		"limit": fmt.Sprintf("%d", limit),
	}).Get("/user/list")
	if err != nil {
		return nil, err
	}
	switch response.StatusCode() {
	case http.StatusOK:
		var res []dtypes.User
		err = json.Unmarshal(response.Body(), &res)
		if err != nil {
			return nil, err
		}

		m.lk.Lock()
		m.miners = make([]dtypes.MinerInfo, 0)
		for _, val := range res {
			addr, err := address.NewFromString(val.Miner)
			if err == nil {
				m.miners = append(m.miners, dtypes.MinerInfo{
					Addr: addr,
					Id:   val.ID,
					Name: val.Name,
				})
			} else {
				log.Errorf("miner [%s] is error", val.Miner)
			}
		}
		m.lk.Unlock()
		return m.miners, err
	default:
		response.Result()
		return nil, fmt.Errorf("response code is : %d, msg:%s", response.StatusCode(), response.Body())
	}
}

func (m *MinerManagerForAuth) Count(ctx context.Context) int {
	m.lk.Lock()
	defer m.lk.Unlock()

	return len(m.miners)
}

var _ minermanage.MinerManageAPI = &MinerManagerForAuth{}
