package posterkeymgr

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ipfs/go-datastore"
	"github.com/prometheus/common/log"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

const actorKey = "miner-actors"
const defaultKey = "default-actor"

var nodefault = xerrors.Errorf("not set default key")
var _ = (IActorMgr)(&ActorMgr{})

type IActorMgr interface {
	AddKey(addr config.MinerInfo) error
	RemoveKey(rmAddr address.Address) error
	ExistKey(checkAddr address.Address) bool
	ListKey() ([]config.MinerInfo, error)
	SetDefault(address.Address) error
	Default() (address.Address, error)
	Count() int
}

type ActorMgr struct {
	da  dtypes.MetadataDS
	cfg *config.MinerConfig
	lk  sync.Mutex
}

func NewActorMgr(ds dtypes.MetadataDS, cfg *config.MinerConfig) (*ActorMgr, error) {
	addrs, err := mergerDbAndConfig(ds, cfg)
	if err != nil {
		return nil, err
	}
	addrBytes, err := json.Marshal(addrs)
	if err != nil {
		return nil, err
	}
	err = ds.Put(datastore.NewKey(actorKey), addrBytes)
	if err != nil {
		return nil, err
	}
	cfg.MinerInfos = addrs
	return &ActorMgr{da: ds, cfg: cfg}, nil
}

func mergerDbAndConfig(ds dtypes.MetadataDS, cfg *config.MinerConfig) ([]config.MinerInfo, error) {
	var addrs []config.MinerInfo
	addrBytes, err := ds.Get(datastore.NewKey(actorKey))
	if err != nil && err != datastore.ErrNotFound {
		return nil, err
	}
	fmt.Printf("miner address: %s\n", string(addrBytes))
	if err == nil {
		err = json.Unmarshal(addrBytes, &addrs)
		if err != nil {
			return nil, err
		}
	}

	mergedAddrs := map[address.Address]config.MinerInfo{}
	for _, addrPoster := range append(addrs, cfg.MinerInfos...) {
		if _, ok := mergedAddrs[addrPoster.Addr]; !ok {
			mergedAddrs[addrPoster.Addr] = addrPoster
		} else {
			//use later replace front
			mergedAddrs[addrPoster.Addr] = addrPoster
		}
	}
	posterAddrs := []config.MinerInfo{}
	for _, val := range mergedAddrs {
		posterAddrs = append(posterAddrs, val)
	}
	return posterAddrs, nil
}

func (actorMgr *ActorMgr) AddKey(posterAddr config.MinerInfo) error {
	actorMgr.lk.Lock()
	defer actorMgr.lk.Unlock()

	if actorMgr.existKey(posterAddr.Addr) {
		log.Warnf("addr %s has exit", posterAddr.Addr)
		return nil
	}

	newAddress := append(actorMgr.cfg.MinerInfos, posterAddr)
	addrBytes, err := json.Marshal(newAddress)
	if err != nil {
		return err
	}
	err = actorMgr.da.Put(datastore.NewKey(actorKey), addrBytes)
	if err != nil {
		return err
	}
	actorMgr.cfg.MinerInfos = newAddress
	return nil
}

func (actorMgr *ActorMgr) RemoveKey(rmAddr address.Address) error {
	actorMgr.lk.Lock()
	defer actorMgr.lk.Unlock()

	if !actorMgr.existKey(rmAddr) {
		return nil
	}
	var newPosterAddr []config.MinerInfo
	for _, posterAddr := range actorMgr.cfg.MinerInfos {
		if posterAddr.Addr.String() != rmAddr.String() {
			newPosterAddr = append(newPosterAddr, posterAddr)
		}
	}
	addrBytes, err := json.Marshal(newPosterAddr)
	if err != nil {
		return err
	}
	err = actorMgr.da.Put(datastore.NewKey(actorKey), addrBytes)
	if err != nil {
		return err
	}
	actorMgr.cfg.MinerInfos = newPosterAddr

	//rm default if rmAddr == defaultAddr
	defaultAddr, err := actorMgr.Default()
	if err != nil {
		if err == nodefault {
			return nil
		}
		return err
	}
	if rmAddr == defaultAddr {
		err := actorMgr.rmfault()
		if err != nil {
			return err
		}
	}
	return nil
}

func (actorMgr *ActorMgr) ExistKey(checkAddr address.Address) bool {
	actorMgr.lk.Lock()
	defer actorMgr.lk.Unlock()

	return actorMgr.existKey(checkAddr)
}

func (actorMgr *ActorMgr) existKey(checkAddr address.Address) bool {

	for _, posterAddr := range actorMgr.cfg.MinerInfos {
		if posterAddr.Addr.String() == checkAddr.String() {
			return true
		}
	}
	return false
}

func (actorMgr *ActorMgr) ListKey() ([]config.MinerInfo, error) {
	actorMgr.lk.Lock()
	defer actorMgr.lk.Unlock()

	return actorMgr.cfg.MinerInfos, nil
}

func (actorMgr *ActorMgr) Count() int {
	actorMgr.lk.Lock()
	defer actorMgr.lk.Unlock()

	return len(actorMgr.cfg.MinerInfos)
}

func (actorMgr *ActorMgr) SetDefault(addr address.Address) error {
	return actorMgr.da.Put(datastore.NewKey(defaultKey), addr.Bytes())
}

func (actorMgr *ActorMgr) Default() (address.Address, error) {
	bytes, err := actorMgr.da.Get(datastore.NewKey(defaultKey))
	if err != nil {
		if len(actorMgr.cfg.MinerInfos) == 0 {
			return address.Undef, nodefault
		}
		return actorMgr.cfg.MinerInfos[0].Addr, nil
	}
	return address.NewFromBytes(bytes)
}

func (actorMgr *ActorMgr) rmfault() error {
	return actorMgr.da.Delete(datastore.NewKey(defaultKey))
}
