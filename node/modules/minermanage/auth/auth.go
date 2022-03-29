package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
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
	users, err := m.listUsers(skip, limit)
	if err != nil {
		return nil, err
	}
	var mInfos []dtypes.MinerInfo

	for _, u := range users {
		if u.State != 1 {
			log.Warnf("user: %s state is disabled, it's miners won't be updated", u.Name)
			continue
		}
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
			mInfos = append(mInfos, dtypes.MinerInfo{
				Addr: addr,
				Id:   u.ID,
				Name: u.Name,
			})
		}
	}
	return mInfos, nil
}

func (m *MinerManagerForAuth) Count(ctx context.Context) int {
	m.lk.Lock()
	defer m.lk.Unlock()

	return len(m.miners)
}

var _ minermanage.MinerManageAPI = &MinerManagerForAuth{}

func (m *MinerManagerForAuth) listUsers(skip, limit int64) ([]*dtypes.User, error) {
	var users []*dtypes.User
	resp, err := m.cli.R().SetQueryParams(map[string]string{
		"skip":  strconv.FormatInt(skip, 10),
		"limit": strconv.FormatInt(limit, 10),
	}).SetResult(&users).SetError(&apiErr{}).Get("/user/list")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusOK {
		return *(resp.Result().(*[]*dtypes.User)), nil
	}
	return nil, resp.Error().(*apiErr).Err()
}

func (m *MinerManagerForAuth) listMiners(user string) ([]*dtypes.Miner, error) {
	var res []*dtypes.Miner
	resp, err := m.cli.R().SetQueryParams(map[string]string{"user": user}).
		SetResult(&res).SetError(&apiErr{}).Get("/miner/list-by-user")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusOK {
		return *(resp.Result().(*[]*dtypes.Miner)), nil
	}
	return nil, resp.Error().(*apiErr).Err()
}

type apiErr struct {
	Error string `json:"error"`
}

func (err *apiErr) Err() error {
	return fmt.Errorf(err.Error)
}
