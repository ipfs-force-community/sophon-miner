package minerecorder

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
)

var (
	ErrDatastoreNotSet = fmt.Errorf("database not set")
)
var innerRecorder = DefaultRecorder{}

func SetDatastore(ds datastore.Datastore) {
	innerRecorder.ds = namespace.Wrap(ds, DatastoreNamespaceKey)
}

func checkAvailable() error {
	if innerRecorder.ds == nil {
		return ErrDatastoreNotSet
	}
	return nil
}

func Record(ctx context.Context, miner address.Address, epoch abi.ChainEpoch, r Records) error {
	err := checkAvailable()
	if err != nil {
		return err
	}
	return innerRecorder.Record(ctx, miner, epoch, r)
}

func Query(ctx context.Context, miner address.Address, epoch abi.ChainEpoch, limit uint) ([]Records, error) {
	err := checkAvailable()
	if err != nil {
		return nil, err
	}
	return innerRecorder.Query(ctx, miner, epoch, limit)
}

type subRecorder struct {
	miner address.Address
	epoch abi.ChainEpoch
}

func (s *subRecorder) Record(ctx context.Context, r Records) {
	err := checkAvailable()
	if err != nil {
		log.Errorf("record failed: %s", err.Error())
	}
	err = innerRecorder.Record(ctx, s.miner, s.epoch, r)
	if err != nil {
		log.Errorf("record failed: %s", err.Error())
	}
}

func Sub(miner address.Address, epoch abi.ChainEpoch) SubRecorder {
	return &subRecorder{
		miner: miner,
		epoch: epoch,
	}
}
