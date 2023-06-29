package slashfilter

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/sophon-miner/node/config"
	"github.com/ipfs-force-community/sophon-miner/types"

	venusTypes "github.com/filecoin-project/venus/venus-shared/types"
)

var log = logging.Logger("mysql_slashFilter")

type mysqlSlashFilter struct {
	_db *gorm.DB
}

type MinedBlock = types.MinedBlock

var _ SlashFilterAPI = (*mysqlSlashFilter)(nil)

func NewMysql(cfg *config.MySQLConfig) (SlashFilterAPI, error) {
	db, err := gorm.Open(mysql.Open(cfg.Conn))
	if err != nil {
		return nil, fmt.Errorf("mysql open %s: %w", cfg.Conn, err)
	}

	db.Set("gorm:table_options", "CHARSET=utf8mb4")
	if cfg.Debug {
		db = db.Debug()
	}

	if err := db.AutoMigrate(MinedBlock{}); err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Set the maximum number of idle connections in the connection pool.
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConn)
	// Set the maximum number of open database connections.
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConn)
	// The maximum time that the connection can be reused is set.
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifeTime))

	log.Info("init mysql success for mysqlSlashFilter!")
	return &mysqlSlashFilter{
		_db: db,
	}, nil
}

// double-fork mining (2 blocks at one epoch)
func (f *mysqlSlashFilter) checkSameHeightFault(bh *venusTypes.BlockHeader) error { // nolint: unused
	var blk MinedBlock
	err := f._db.Model(&MinedBlock{}).Take(&blk, "miner=? and epoch=?", bh.Miner.String(), bh.Height).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	if 0 >= len(blk.Cid) {
		return nil
	}

	other, err := cid.Decode(blk.Cid)
	if err != nil {
		return err
	}

	if other == bh.Cid() {
		return nil
	}

	return fmt.Errorf("produced block would trigger double-fork mining faults consensus fault; miner: %s; bh: %s, other: %s", bh.Miner, bh.Cid(), other)
}

// time-offset mining faults (2 blocks with the same parents)
func (f *mysqlSlashFilter) checkSameParentFault(bh *venusTypes.BlockHeader) error {
	var blk MinedBlock
	err := f._db.Model(&MinedBlock{}).Take(&blk, "miner=? and parent_key=?", bh.Miner.String(), venusTypes.NewTipSetKey(bh.Parents...).String()).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	if 0 >= len(blk.Cid) {
		return nil
	}

	other, err := cid.Decode(blk.Cid)
	if err != nil {
		return err
	}

	if other == bh.Cid() {
		return nil
	}

	return fmt.Errorf("produced block would trigger time-offset mining faults consensus fault; miner: %s; bh: %s, other: %s", bh.Miner, bh.Cid(), other)
}

func (f *mysqlSlashFilter) HasBlock(ctx context.Context, bh *venusTypes.BlockHeader) (bool, error) {
	var blk MinedBlock
	err := f._db.Model(&MinedBlock{}).Take(&blk, "miner=? and epoch=?", bh.Miner.String(), bh.Height).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return len(blk.Cid) > 0, nil
}

func (f *mysqlSlashFilter) PutBlock(ctx context.Context, bh *venusTypes.BlockHeader, parentEpoch abi.ChainEpoch, t time.Time, state types.StateMining) error {
	var blk MinedBlock
	err := f._db.Model(&MinedBlock{}).Take(&blk, "miner=? and epoch=?", bh.Miner.String(), bh.Height).Error
	if err != nil {
		// Timeout may not be the winner when it happens, once the winner database must be recorded.
		if err == gorm.ErrRecordNotFound && state != types.Timeout {
			mblk := &MinedBlock{
				ParentEpoch: int64(parentEpoch),
				ParentKey:   venusTypes.NewTipSetKey(bh.Parents...).String(),
				Epoch:       int64(bh.Height),
				Miner:       bh.Miner.String(),

				MineState: state,
			}

			if bh.Ticket != nil {
				mblk.Cid = bh.Cid().String()
			}

			if !t.IsZero() {
				mblk.WinningAt = t
			}

			return f._db.Create(mblk).Error
		}

		return fmt.Errorf("query record failed: %w", err)
	}

	updateColumns := make(map[string]interface{})
	updateColumns["parent_epoch"] = parentEpoch
	if len(bh.Parents) > 0 {
		updateColumns["parent_key"] = venusTypes.NewTipSetKey(bh.Parents...).String()
	}
	updateColumns["mine_state"] = state
	if bh.Ticket != nil {
		updateColumns["cid"] = bh.Cid().String()
	}

	return f._db.Model(&MinedBlock{}).Where("miner=? and epoch=?", bh.Miner.String(), bh.Height).UpdateColumns(updateColumns).Error
}

func (f *mysqlSlashFilter) MinedBlock(ctx context.Context, bh *venusTypes.BlockHeader, parentEpoch abi.ChainEpoch) error {
	// double-fork mining (2 blocks at one epoch) --> HasBlock
	//if err := f.checkSameHeightFault(bh); err != nil {
	//	return err
	//}

	if err := f.checkSameParentFault(bh); err != nil {
		return err
	}

	{
		// parent-grinding fault (didn't mine on top of our own block)

		// First check if we have mined a block on the parent epoch
		var blk MinedBlock
		err := f._db.Model(&MinedBlock{}).Take(&blk, "miner=? and epoch=?", bh.Miner.String(), parentEpoch).Error
		if err == nil {
			// If we had, make sure it's in our parent tipset
			if len(blk.Cid) > 0 {
				parent, err := cid.Decode(blk.Cid)
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
					return errors.Wrapf(ParentGrindingFaults, "produced block would trigger consensus fault; miner: %s; bh: %s, expected parent: %s", bh.Miner, bh.Cid(), parent)
				}
			}
		} else if err != gorm.ErrRecordNotFound {
			//other error except not found
			return err
		}
		//if not exit good block
	}

	return nil
}

func (f *mysqlSlashFilter) ListBlock(ctx context.Context, params *types.BlocksQueryParams) ([]MinedBlock, error) {
	var blks []MinedBlock
	query := f._db.Order("epoch desc")

	if len(params.Miners) > 0 {
		temp := make([]string, 0, len(params.Miners))
		for _, miner := range params.Miners {
			temp = append(temp, miner.String())
		}
		query = query.Where("miner in (?)", temp)
	}
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	err := query.Find(&blks).Error
	if err != nil {
		return nil, err
	}

	return blks, nil
}
