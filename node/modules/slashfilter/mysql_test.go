package slashfilter

import (
	"context"
	"regexp"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/venus/venus-shared/testutil"
	"github.com/filecoin-project/venus/venus-shared/types"
)

func wrapper(f func(*testing.T, *mysqlSlashFilter, sqlmock.Sqlmock), sf *mysqlSlashFilter, mock sqlmock.Sqlmock) func(t *testing.T) {
	return func(t *testing.T) {
		f(t, sf, mock)
	}
}

func TestMysqlSlashFilter(t *testing.T) {
	sf, mock := mysqlSetup(t)

	t.Run("put-block", wrapper(testPutBlock, sf, mock))
	t.Run("has-block", wrapper(testHasBlock, sf, mock))
	t.Run("mined-block", wrapper(testMinedBlock, sf, mock))
}

func testPutBlock(t *testing.T, sf *mysqlSlashFilter, mock sqlmock.Sqlmock) {
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
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `miner_blocks` WHERE miner=? and epoch=? LIMIT 1")).
			WithArgs(cases[0].bh.Miner.String(), cases[0].bh.Height).WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `miner_blocks` (`parent_epoch`,`parent_key`,`epoch`,`miner`,`cid`,`winning_at`,`mine_state`,`consuming`) VALUES (?,?,?,?,?,?,?,?)")).
			WithArgs(cases[0].parentEpoch, types.NewTipSetKey(cases[0].bh.Parents...).String(),
				cases[0].bh.Height, cases[0].bh.Miner.String(), "", cases[0].t, cases[0].state, 0).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := sf.PutBlock(ctx, cases[0].bh, cases[0].parentEpoch, cases[0].t, cases[0].state)
		if err != nil {
			t.Fatalf("put block err: %s", err.Error())
		}
	})

	t.Run("update block", func(t *testing.T) {
		columns := []string{"parent_epoch", "parent_key", "epoch", "miner", "winning_at", `mine_state`}
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `miner_blocks` WHERE miner=? and epoch=? LIMIT 1")).
			WithArgs(cases[1].bh.Miner.String(), cases[1].bh.Height).WillReturnRows(
			sqlmock.NewRows(columns).AddRow(cases[0].parentEpoch, types.NewTipSetKey(cases[0].bh.Parents...).String(),
				cases[0].bh.Height, cases[0].bh.Miner.String(), cases[0].t, cases[0].state))
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("UPDATE `miner_blocks` SET `cid`=?,`mine_state`=?,`parent_epoch`=?,`parent_key`=? WHERE miner=? and epoch=?")).
			WithArgs(cases[1].bh.Cid().String(), cases[1].state, cases[1].parentEpoch, types.NewTipSetKey(cases[1].bh.Parents...).String(),
				cases[1].bh.Miner.String(), cases[1].bh.Height).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := sf.PutBlock(ctx, cases[1].bh, cases[1].parentEpoch, cases[1].t, cases[1].state)
		if err != nil {
			t.Fatalf("put block err: %s", err.Error())
		}
	})
}

func testHasBlock(t *testing.T, sf *mysqlSlashFilter, mock sqlmock.Sqlmock) {
	const mdata = "f021344"
	maddr, err := address.NewFromString(mdata)
	require.NoErrorf(t, err, "parse miner address %s", mdata)

	mockCid, _ := cid.Parse("bafkqaaa")

	bh := &types.BlockHeader{
		Height: abi.ChainEpoch(100),
		Miner:  maddr,
		Ticket: &types.Ticket{
			VRFProof: []byte("====1====="),
		},
		ParentStateRoot:       mockCid,
		ParentMessageReceipts: mockCid,
		Messages:              mockCid,
	}

	ctx := context.Background()

	t.Run("block not exist", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `miner_blocks` WHERE miner=? and epoch=? LIMIT 1")).
			WithArgs(bh.Miner.String(), bh.Height).WillReturnError(gorm.ErrRecordNotFound)

		has, err := sf.HasBlock(ctx, bh)
		require.Nil(t, err)
		require.False(t, has)
	})

	t.Run("has block", func(t *testing.T) {
		columns := []string{"epoch", "miner", "cid"}
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `miner_blocks` WHERE miner=? and epoch=? LIMIT 1")).
			WithArgs(bh.Miner.String(), bh.Height).
			WillReturnRows(sqlmock.NewRows(columns).AddRow(bh.Height, bh.Miner.String(), bh.Cid().String()))
		has, err := sf.HasBlock(ctx, bh)
		require.Nil(t, err)
		require.True(t, has)
	})
}

func testMinedBlock(t *testing.T, sf *mysqlSlashFilter, mock sqlmock.Sqlmock) {
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
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `miner_blocks` WHERE miner=? and parent_key=? LIMIT 1")).
			WithArgs(bh.Miner.String(), types.NewTipSetKey(bh.Parents...).String()).WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `miner_blocks` WHERE miner=? and epoch=? LIMIT 1")).
			WithArgs(bh.Miner.String(), ph).WillReturnError(gorm.ErrRecordNotFound)

		err = sf.MinedBlock(ctx, bh, ph)
		require.Nil(t, err)
	})

	t.Run("time-offset mining", func(t *testing.T) {
		columns := []string{"epoch", "miner", "cid"}
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `miner_blocks` WHERE miner=? and parent_key=? LIMIT 1")).
			WithArgs(bh.Miner.String(), types.NewTipSetKey(bh.Parents...).String()).
			WillReturnRows(sqlmock.NewRows(columns).AddRow(bh.Height, bh.Miner.String(), bh.Cid().String()))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `miner_blocks` WHERE miner=? and epoch=? LIMIT 1")).
			WithArgs(bh.Miner.String(), ph).WillReturnError(gorm.ErrRecordNotFound)

		err = sf.MinedBlock(ctx, bh, ph)
		require.Nil(t, err)
	})

	t.Run("parent-grinding", func(t *testing.T) {
		columns := []string{"epoch", "miner", "cid"}
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `miner_blocks` WHERE miner=? and parent_key=? LIMIT 1")).
			WithArgs(bh.Miner.String(), types.NewTipSetKey(bh.Parents...).String()).WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `miner_blocks` WHERE miner=? and epoch=? LIMIT 1")).
			WithArgs(bh.Miner.String(), ph).
			WillReturnRows(sqlmock.NewRows(columns).AddRow(bh.Height, bh.Miner.String(), parents[0].String()))

		err = sf.MinedBlock(ctx, bh, ph)
		require.Nil(t, err)
	})
}

func mysqlSetup(t *testing.T) (*mysqlSlashFilter, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})

	if err != nil {
		t.Fatal(err)
	}

	return &mysqlSlashFilter{_db: gormDB}, mock
}
