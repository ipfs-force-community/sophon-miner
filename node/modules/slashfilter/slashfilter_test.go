package slashfilter

import (
	"context"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/venus/venus-shared/testutil"
	"github.com/filecoin-project/venus/venus-shared/types"
)

func wrapper(f func(*testing.T, SlashFilterAPI, func() error), sf SlashFilterAPI, clear func() error) func(t *testing.T) {
	return func(t *testing.T) {
		f(t, sf, clear)
	}
}

func makeMysqlTestCtx(t *testing.T) (SlashFilterAPI, func() error) {
	sf, db, err := NewMysqlMock()
	require.NoError(t, err)
	clear := func() error {
		return db.Delete(&MinedBlock{}, "1 = 1").Error
	}
	return sf, clear
}

func makeLocalTestCtx(t *testing.T) (SlashFilterAPI, func() error) {
	sf, ds, err := NewLocalMock()
	require.NoError(t, err)

	ctx := context.TODO()

	clear := func() error {
		rs, err := ds.Query(ctx, query.Query{Prefix: "/"})
		if err != nil {
			return err
		}
		for {
			res, ok := rs.NextSync()
			if !ok {
				break
			}
			if err := ds.Delete(ctx, datastore.NewKey(res.Key)); err != nil {
				return err
			}
		}
		return nil
	}
	return sf, clear
}

func checkHasBlock(t *testing.T, api SlashFilterAPI, bh *types.BlockHeader, expect bool) {
	ctx := context.TODO()
	exists, err := api.HasBlock(ctx, bh)
	require.NoError(t, err)
	require.Equal(t, exists, expect)
}

func TestSlashFilter(t *testing.T) {
	t.Run("mysql slash filter tests", func(t *testing.T) {
		sf, clear := makeMysqlTestCtx(t)
		t.Run("put-block", wrapper(testPutBlock, sf, clear))
		t.Run("mined-block", wrapper(testMinedBlock, sf, clear))
	})
	t.Run("local slash filter tests", func(t *testing.T) {
		sf, clear := makeLocalTestCtx(t)
		t.Run("put-block", wrapper(testPutBlock, sf, clear))
		t.Run("mined-block", wrapper(testMinedBlock, sf, clear))
	})
}

func testPutBlock(t *testing.T, sf SlashFilterAPI, clear func() error) {
	require.NoError(t, clear())

	const mdata = "f021344"
	maddr, err := address.NewFromString(mdata)
	require.NoErrorf(t, err, "parse miner address %s", mdata)

	var parents []cid.Cid
	testutil.Provide(t, &parents, testutil.WithSliceLen(2))

	mockCid, _ := cid.Parse("bafkqaaa")

	h := abi.ChainEpoch(100)
	ph := abi.ChainEpoch(99)

	cases := []struct {
		bh          *types.BlockHeader
		parentEpoch abi.ChainEpoch
		t           time.Time
		state       StateMining
	}{
		{
			bh: &types.BlockHeader{
				Parents: parents,
				Height:  h,
				Miner:   maddr,
				Ticket:  nil,
			},
			parentEpoch: ph,
			t:           time.Now(),
			state:       Mining,
		},
		{
			bh: &types.BlockHeader{
				Parents: parents,
				Height:  h,
				Miner:   maddr,
				Ticket: &types.Ticket{
					VRFProof: []byte("====1====="),
				},
				ParentStateRoot:       mockCid,
				ParentMessageReceipts: mockCid,
				Messages:              mockCid,
			},
			parentEpoch: ph,
			t:           time.Time{},
			state:       Success,
		},
	}

	ctx := context.Background()

	t.Run("new block", func(t *testing.T) {
		cs := cases[0]
		checkHasBlock(t, sf, cs.bh, false)
		require.NoError(t, sf.PutBlock(ctx, cs.bh, cs.parentEpoch, cs.t, cs.state))
		// BlockHeader in case[0], don't have a ticket, `HasBlock` should return false.
		checkHasBlock(t, sf, cs.bh, false)
	})

	t.Run("update block", func(t *testing.T) {
		cs := cases[1]
		err := sf.PutBlock(ctx, cs.bh, cs.parentEpoch, cs.t, cs.state)
		require.NoError(t, err)
		// After updated,  block.Ticket is not nil, and its len(Cids) won't be zero.
		// 	hasBlock should be 'true'
		checkHasBlock(t, sf, cs.bh, true)
	})
}

func testMinedBlock(t *testing.T, sf SlashFilterAPI, clear func() error) {
	const mdata = "f021344"
	maddr, err := address.NewFromString(mdata)
	require.NoErrorf(t, err, "parse miner address %s", mdata)

	var parents []cid.Cid
	testutil.Provide(t, &parents, testutil.WithSliceLen(2))

	mockCid, _ := cid.Parse("bafkqaaa")
	bh := &types.BlockHeader{
		Height: abi.ChainEpoch(100),
		Miner:  maddr,
		Ticket: &types.Ticket{
			VRFProof: []byte("====1====="),
		},
		Parents:               parents,
		ParentStateRoot:       mockCid,
		ParentMessageReceipts: mockCid,
		Messages:              mockCid,
	}
	ph := abi.ChainEpoch(99)

	ctx := context.Background()

	t.Run("general", func(t *testing.T) {
		require.NoError(t, clear())
		err = sf.MinedBlock(ctx, bh, ph)
		require.Nil(t, err)
	})

	t.Run("time-offset mining", func(t *testing.T) {
		require.NoError(t, clear())
		require.NoError(t, sf.PutBlock(ctx, bh, ph, time.Now(), Success))

		// change bh.Message to simulate case of time-offset mining fault(2 blocks with the same parents)
		testutil.Provide(t, &bh.Messages)
		err = sf.MinedBlock(ctx, bh, ph)
		require.Error(t, err)
		require.Greater(t, strings.Index(err.Error(), "time-offset"), 0)
	})

	t.Run("parent-grinding", func(t *testing.T) {
		require.NoError(t, clear())
		err = sf.PutBlock(ctx, bh, ph, time.Now(), Success)
		require.NoError(t, err)

		blockId := bh.Cid()

		// simulate case of "parent griding" fault
		// 	parents of new block header don't include previous block,
		testutil.Provide(t, &bh.Parents, testutil.WithSliceLen(2))

		bh.Height = bh.Height + 1
		err = sf.MinedBlock(ctx, bh, ph+1)
		require.Error(t, err)
		require.Greater(t, strings.Index(err.Error(), "parent-grinding fault"), 0)

		// include parent block into its `parents`, expect success..
		bh.Parents = append(bh.Parents, blockId)
		err = sf.MinedBlock(ctx, bh, ph+1)
		require.NoError(t, err)
	})
}
