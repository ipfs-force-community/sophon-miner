package minerecorder

import (
	"context"
	"errors"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-datastore"
)

var (
	ErrDatastoreNotSet  = fmt.Errorf("database not set")
	ErrRecorderDisabled = fmt.Errorf("recorder disabled")
)
var innerRecorder Recorder
var enable = false

func SetDatastore(ds datastore.Datastore) {
	innerRecorder = NewDefaultRecorder(ds)
	enable = true
}

func checkAvailable() error {
	if !enable {
		return ErrRecorderDisabled
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
	if errors.Is(err, ErrRecorderDisabled) {
		log.Debugf("recorder disabled, skip record")
		return
	} else if err != nil {
		log.Warnf("record failed: %s", err.Error())
	}
	err = innerRecorder.Record(ctx, s.miner, s.epoch, r)
	if err != nil {
		log.Warnf("record failed: %s", err.Error())
	}
}

func Sub(miner address.Address, epoch abi.ChainEpoch) SubRecorder {
	return &subRecorder{
		miner: miner,
		epoch: epoch,
	}
}
