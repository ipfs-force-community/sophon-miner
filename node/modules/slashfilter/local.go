package slashfilter

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"

	"github.com/filecoin-project/go-state-types/abi"

	vtypes "github.com/filecoin-project/venus/venus-shared/types"

	"github.com/filecoin-project/venus-miner/types"
)

type localSlashFilter struct {
	byEpoch   datastore.Datastore // double-fork mining faults, parent-grinding fault
	byParents datastore.Datastore // time-offset mining faults
}

func NewLocal(ds types.MetadataDS) SlashFilterAPI {
	return &localSlashFilter{
		byEpoch:   namespace.Wrap(ds, datastore.NewKey("/slashfilter/epoch")),
		byParents: namespace.Wrap(ds, datastore.NewKey("/slashfilter/parents")),
	}
}

func (f *localSlashFilter) HasBlock(ctx context.Context, bh *vtypes.BlockHeader) (bool, error) {
	epochKey := datastore.NewKey(fmt.Sprintf("/%s/%d", bh.Miner, bh.Height))

	return f.byEpoch.Has(ctx, epochKey)
}

func (f *localSlashFilter) PutBlock(ctx context.Context, bh *vtypes.BlockHeader, parentEpoch abi.ChainEpoch, t time.Time, state StateMining) error {
	// Only successful block generation is recorded locally
	if state != Success {
		return nil
	}

	parentsKey := datastore.NewKey(fmt.Sprintf("/%s/%x", bh.Miner, vtypes.NewTipSetKey(bh.Parents...).Bytes()))
	if err := f.byParents.Put(ctx, parentsKey, bh.Cid().Bytes()); err != nil {
		return fmt.Errorf("putting byEpoch entry: %w", err)
	}

	epochKey := datastore.NewKey(fmt.Sprintf("/%s/%d", bh.Miner, bh.Height))
	if err := f.byEpoch.Put(ctx, epochKey, bh.Cid().Bytes()); err != nil {
		return fmt.Errorf("putting byEpoch entry: %w", err)
	}

	return nil
}

func (f *localSlashFilter) MinedBlock(ctx context.Context, bh *vtypes.BlockHeader, parentEpoch abi.ChainEpoch) error {
	// double-fork mining (2 blocks at one epoch) --> HasBlock

	parentsKey := datastore.NewKey(fmt.Sprintf("/%s/%x", bh.Miner, vtypes.NewTipSetKey(bh.Parents...).Bytes()))
	{
		// time-offset mining faults (2 blocks with the same parents)
		if err := checkFault(ctx, f.byParents, parentsKey, bh, "time-offset mining faults"); err != nil {
			return err
		}
	}

	{
		// parent-grinding fault (didn't mine on top of our own block)

		// First check if we have mined a block on the parent epoch
		parentEpochKey := datastore.NewKey(fmt.Sprintf("/%s/%d", bh.Miner, parentEpoch))
		have, err := f.byEpoch.Has(ctx, parentEpochKey)
		if err != nil {
			return err
		}

		if have {
			// If we had, make sure it's in our parent tipset
			cidb, err := f.byEpoch.Get(ctx, parentEpochKey)
			if err != nil {
				return fmt.Errorf("getting other block cid: %w", err)
			}

			_, parent, err := cid.CidFromBytes(cidb)
			if err != nil {
				return err
			}

			var found bool
			for _, c := range bh.Parents {
				if c.Equals(parent) {
					found = true
				}
			}

			if !found {
				return fmt.Errorf("produced block would trigger 'parent-grinding fault' consensus fault; miner: %s; bh: %s, expected parent: %s", bh.Miner, bh.Cid(), parent)
			}
		}
	}

	return nil
}

func checkFault(ctx context.Context, t datastore.Datastore, key datastore.Key, bh *vtypes.BlockHeader, faultType string) error {
	fault, err := t.Has(ctx, key)
	if err != nil {
		return err
	}

	if fault {
		cidb, err := t.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("getting other block cid: %w", err)
		}

		_, other, err := cid.CidFromBytes(cidb)
		if err != nil {
			return err
		}

		if other == bh.Cid() {
			return nil
		}

		return fmt.Errorf("produced block would trigger '%s' consensus fault; miner: %s; bh: %s, other: %s", faultType, bh.Miner, bh.Cid(), other)
	}

	return nil
}
