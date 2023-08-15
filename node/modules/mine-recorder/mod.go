package minerecorder

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"github.com/ipfs/go-datastore/query"
)

var log = logging.Logger("mine-recorder")

var (
	// MaxRecordPerQuery is the max record can be query in one time
	MaxRecordPerQuery uint = 288
	// ExpireEpoch is the num of epoch that record will be expired, default is 7 * 2880 epoch, about 7 days
	ExpireEpoch = abi.ChainEpoch(7 * 2880)

	DatastoreNamespaceKey = datastore.NewKey("/mine-record")
)

var (
	ErrorRecordNotFound          = datastore.ErrNotFound
	ErrorExceedMaxRecordPerQuery = fmt.Errorf("query exceed MaxRecordPerQuery(%d)", MaxRecordPerQuery)
)

var (
	metaMinEpochKey = datastore.NewKey("/index/min-epoch")
)

type Recorder interface {
	Record(ctx context.Context, miner address.Address, epoch abi.ChainEpoch, r Records) error
	Query(ctx context.Context, miner address.Address, from abi.ChainEpoch, limit uint) ([]Records, error)
}

type SubRecorder interface {
	Record(ctx context.Context, r Records)
}

type Records = map[string]string

func key(miner address.Address, epoch abi.ChainEpoch) datastore.Key {
	return datastore.NewKey("/" + epoch.String() + "/" + miner.String())
}

type DefaultRecorder struct {
	ds                datastore.Datastore
	latestRecordEpoch abi.ChainEpoch
}

var _ Recorder = (*DefaultRecorder)(nil)

func NewDefaultRecorder(ds datastore.Datastore) *DefaultRecorder {
	return &DefaultRecorder{
		ds: namespace.Wrap(ds, DatastoreNamespaceKey),
	}
}

func (d *DefaultRecorder) Record(ctx context.Context, miner address.Address, epoch abi.ChainEpoch, r Records) error {
	before, err := d.get(ctx, miner, epoch)
	if err != nil && !errors.Is(err, ErrorRecordNotFound) {
		return err
	}

	r = coverMap(before, r)

	err = d.put(ctx, miner, epoch, r)
	if err != nil {
		return err
	}

	err = d.cleanExpire(ctx, epoch)
	if err != nil {
		return err
	}
	return nil
}

func (d *DefaultRecorder) get(ctx context.Context, miner address.Address, epoch abi.ChainEpoch) (Records, error) {
	key := key(miner, epoch)
	val, err := d.ds.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get record (%s) fail: %w", key, err)
	}
	var record Records
	err = json.Unmarshal(val, &record)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (d *DefaultRecorder) put(ctx context.Context, miner address.Address, epoch abi.ChainEpoch, r Records) error {
	key := key(miner, epoch)
	val, err := json.Marshal(&r)
	if err != nil {
		return err
	}
	err = d.ds.Put(ctx, key, val)
	if err != nil {
		return err
	}
	return nil
}

func (d *DefaultRecorder) Query(ctx context.Context, miner address.Address, from abi.ChainEpoch, limit uint) ([]Records, error) {
	if limit > MaxRecordPerQuery {
		return nil, ErrorExceedMaxRecordPerQuery
	}
	if limit == 0 {
		limit = 1
	}
	ret := make([]Records, 0, limit)
	to := from + abi.ChainEpoch(limit)
	for ; from < to; from++ {
		r, err := d.get(ctx, miner, from)
		if errors.Is(err, ErrorRecordNotFound) {
			// ignore record not found
			log.Warnf("query record: %s on %d : %s", miner, from, err.Error())
			continue
		} else if err != nil {
			return nil, err
		}
		covered := coverMap(r, Records{"miner": miner.String(), "epoch": from.String()})
		ret = append(ret, covered)
		from++
	}

	if len(ret) != int(limit) {
		log.Infof("query record: %s from %d to %d ,found %d ", miner, from-abi.ChainEpoch(limit), from, len(ret))
	} else {
		log.Debugf("query record: %s from %d to %d ,found %d ", miner, from-abi.ChainEpoch(limit), limit, len(ret))
	}
	return ret, nil
}

func (d *DefaultRecorder) cleanExpire(ctx context.Context, epoch abi.ChainEpoch) error {
	if epoch > d.latestRecordEpoch {
		d.latestRecordEpoch = epoch
		deadline := epoch - ExpireEpoch
		err := d.cleanBefore(ctx, deadline)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DefaultRecorder) cleanBefore(ctx context.Context, deadline abi.ChainEpoch) error {
	minEpochInByte, err := d.ds.Get(ctx, metaMinEpochKey)
	if errors.Is(err, datastore.ErrNotFound) {
		// no min epoch record
		minEpochInByte = intoBytes(deadline)
		err = d.ds.Put(ctx, metaMinEpochKey, minEpochInByte)
		return err
	} else if err != nil {
		return fmt.Errorf("get min epoch: %w", err)
	}

	var minEpoch abi.ChainEpoch
	fromBytes(minEpochInByte, &minEpoch)

	if deadline > minEpoch {
		// delete record range [minEpoch, deadline)
		for i := minEpoch; i < deadline; i++ {
			res, err := d.ds.Query(ctx, query.Query{
				Prefix:   "/" + i.String(),
				KeysOnly: true,
			})
			if err != nil {
				return fmt.Errorf("query record: %w", err)
			}
			for r := range res.Next() {
				if r.Error != nil {
					return fmt.Errorf("query record: %w", r.Error)
				}
				err := d.ds.Delete(ctx, datastore.NewKey(r.Entry.Key))
				if err != nil {
					return fmt.Errorf("delete record: %w", err)
				}
			}
		}

		deadlineInByte := intoBytes(deadline)
		err = d.ds.Put(ctx, metaMinEpochKey, deadlineInByte)
		if err != nil {
			return err
		}
	}
	return nil
}

func intoBytes(v any) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, v) // nolint: errcheck
	return buf.Bytes()
}

func fromBytes(b []byte, v any) {
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, v) // nolint: errcheck
}

// mergeMap cover src onto dst, the key in dst will overwrite the key in src
func coverMap[K comparable, V any](dst, src map[K]V) map[K]V {
	newMap := make(map[K]V)
	for k, v := range dst {
		newMap[k] = v
	}
	for k, v := range src {
		newMap[k] = v
	}
	return newMap
}
