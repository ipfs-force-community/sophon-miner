package slashfilter

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus-miner/node/config"

	"github.com/filecoin-project/venus/venus-shared/types"
)

var log = logging.Logger("mysql_slashFilter")

type mysqlSlashFilter struct {
	_db *gorm.DB
}

type MinedBlock struct {
	ParentEpoch int64  `gorm:"column:parent_epoch;type:bigint(20);NOT NULL"`
	ParentKey   string `gorm:"column:parent_key;type:varchar(1000);NOT NULL"`

	Epoch int64  `gorm:"column:epoch;type:bigint(20);NOT NULL;primary_key"`
	Miner string `gorm:"column:miner;type:varchar(256);NOT NULL;primary_key"`
	Cid   string `gorm:"column:cid;type:varchar(256)"`
}

func (m *MinedBlock) TableName() string {
	return "miner_blocks"
}

var _ SlashFilterAPI = (*mysqlSlashFilter)(nil)

func NewMysql(cfg *config.MySQLConfig) func() (SlashFilterAPI, error) {
	return func() (SlashFilterAPI, error) {
		db, err := gorm.Open(mysql.Open(cfg.Conn))
		if err != nil {
			return nil, fmt.Errorf("[db connection failed] Connection : %s %w", cfg.Conn, err)
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
		sqlDB.SetConnMaxLifetime(time.Second * cfg.ConnMaxLifeTime)

		log.Info("init mysql success for mysqlSlashFilter!")
		return &mysqlSlashFilter{
			_db: db,
		}, nil
	}
}

// double-fork mining (2 blocks at one epoch)
func (f *mysqlSlashFilter) checkSameHeightFault(bh *types.BlockHeader) error { // nolint: unused
	var bk MinedBlock
	err := f._db.Model(&MinedBlock{}).Take(&bk, "miner=? and epoch=?", bh.Miner.String(), bh.Height).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	other, err := cid.Decode(bk.Cid)
	if err != nil {
		return err
	}

	if other == bh.Cid() {
		return nil
	}

	return fmt.Errorf("produced block would trigger double-fork mining faults consensus fault; miner: %s; bh: %s, other: %s", bh.Miner, bh.Cid(), other)
}

// time-offset mining faults (2 blocks with the same parents)
func (f *mysqlSlashFilter) checkSameParentFault(bh *types.BlockHeader) error {
	var bk MinedBlock
	err := f._db.Model(&MinedBlock{}).Take(&bk, "miner=? and parent_key=?", bh.Miner.String(), types.NewTipSetKey(bh.Parents...).String()).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	other, err := cid.Decode(bk.Cid)
	if err != nil {
		return err
	}

	if other == bh.Cid() {
		return nil
	}

	return fmt.Errorf("produced block would trigger time-offset mining faults consensus fault; miner: %s; bh: %s, other: %s", bh.Miner, bh.Cid(), other)
}

func (f *mysqlSlashFilter) HasBlock(ctx context.Context, bh *types.BlockHeader) (bool, error) {
	var bk MinedBlock
	err := f._db.Model(&MinedBlock{}).Take(&bk, "miner=? and epoch=?", bh.Miner.String(), bh.Height).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (f *mysqlSlashFilter) PutBlock(ctx context.Context, bh *types.BlockHeader, parentEpoch abi.ChainEpoch) error {
	return f._db.Save(&MinedBlock{
		ParentEpoch: int64(parentEpoch),
		ParentKey:   types.NewTipSetKey(bh.Parents...).String(),
		Epoch:       int64(bh.Height),
		Miner:       bh.Miner.String(),
		Cid:         bh.Cid().String(),
	}).Error
}

func (f *mysqlSlashFilter) MinedBlock(ctx context.Context, bh *types.BlockHeader, parentEpoch abi.ChainEpoch) error {
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
		err := f._db.Model(&MinedBlock{}).Take(&blk, "miner=? and parent_epoch=?", bh.Miner.String(), parentEpoch).Error
		if err == nil {
			// If we had, make sure it's in our parent tipset
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
				return fmt.Errorf("produced block would trigger 'parent-grinding fault' consensus fault; miner: %s; bh: %s, expected parent: %s", bh.Miner, bh.Cid(), parent)
			}
		} else if err != gorm.ErrRecordNotFound {
			//other error except not found
			return err
		}
		//if not exit good block
	}

	return nil
}
