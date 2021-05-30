package mysql

import (
	"context"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
	"github.com/filecoin-project/venus-miner/node/modules/minermanage"
)

var log = logging.Logger("mysql")

type mysqlMinerInfo struct {
	Addr       string `gorm:"column:addr;type:varchar(20);uniqueIndex;NOT NULL"`
	Name       string `gorm:"column:name;type:varchar(50);"`
	Comment    string `gorm:"column:comment;type:varchar(50);"`
	State      int    `gorm:"column:state;type:int(10);comment:'0 for init,1 for active'"`
	CreateTime uint64 `gorm:"column:create_time;not null;type:bigint(20) unsigned" json:"createTime"`
	UpdateTime uint64 `gorm:"column:update_time;not null;type:bigint(20) unsigned" json:"updateTime"`
}

func (m *mysqlMinerInfo) TableName() string {
	return "miner_infos"
}

type MinerManagerForMySQL struct {
	miners []dtypes.MinerInfo

	_db *gorm.DB
	lk  sync.Mutex
}

func NewMinerManger(cfg *config.MySQLConfig) func() (minermanage.MinerManageAPI, error) {
	return func() (minermanage.MinerManageAPI, error) {
		// root:123456@tcp(ip:3306)/venus_miner?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s
		db, err := gorm.Open(mysql.Open(cfg.Conn), &gorm.Config{
			//Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			return nil, xerrors.Errorf("[db connection failed] conn: %s %w", cfg.Conn, err)
		}

		db.Set("gorm:table_options", "CHARSET=utf8mb4")
		if cfg.Debug {
			db = db.Debug()
		}

		if err := db.AutoMigrate(mysqlMinerInfo{}); err != nil {
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

		log.Info("init mysql success for MinerManger!")
		m := &MinerManagerForMySQL{
			_db: db,
		}

		if err := m.init(); err != nil {
			return nil, err
		}

		return m, nil
	}
}

func (m *MinerManagerForMySQL) init() error {
	var res []mysqlMinerInfo
	if err := m._db.Find(&res).Error; err != nil {
		return err
	}

	for _, val := range res {
		addr, err := address.NewFromString(val.Addr)
		if err != nil {
			return err
		}
		m.miners = append(m.miners, dtypes.MinerInfo{
			Addr: addr,
		})
	}

	return nil
}

func (m *MinerManagerForMySQL) Put(ctx context.Context, miner dtypes.MinerInfo) error {
	m.lk.Lock()
	defer m.lk.Unlock()

	err := m._db.Create(&mysqlMinerInfo{Addr: miner.Addr.String()}).Error
	if err != nil {
		return err
	}

	m.miners = append(m.miners, miner)
	return nil
}

func (m *MinerManagerForMySQL) Set(ctx context.Context, miner dtypes.MinerInfo) error {
	m.lk.Lock()
	defer m.lk.Unlock()

	for _, addr := range m.miners {
		if addr.Addr.String() == miner.Addr.String() {
			// ToDo other changes

			err := m.Put(ctx, miner)
			if err != nil {
				return err
			}
			break
		}
	}

	return nil
}

func (m *MinerManagerForMySQL) Has(ctx context.Context, addr address.Address) bool {
	for _, miner := range m.miners {
		if miner.Addr.String() == addr.String() {
			return true
		}
	}

	return false
}

func (m *MinerManagerForMySQL) Get(ctx context.Context, addr address.Address) *dtypes.MinerInfo {
	m.lk.Lock()
	defer m.lk.Unlock()

	for k := range m.miners {
		if m.miners[k].Addr.String() == addr.String() {
			return &m.miners[k]
		}
	}

	return nil
}

func (m *MinerManagerForMySQL) List(ctx context.Context) ([]dtypes.MinerInfo, error) {
	m.lk.Lock()
	defer m.lk.Unlock()

	return m.miners, nil
}

func (m *MinerManagerForMySQL) Remove(ctx context.Context, rmAddr address.Address) error {
	m.lk.Lock()
	defer m.lk.Unlock()

	if !m.Has(ctx, rmAddr) {
		return nil
	}

	var res mysqlMinerInfo
	m._db.Where("`addr` = ?", rmAddr.String()).Take(&res)

	if err := m._db.Delete(&res).Error; err != nil {
		return err
	}

	var newMiners []dtypes.MinerInfo
	for _, miner := range m.miners {
		if miner.Addr.String() != rmAddr.String() {
			newMiners = append(newMiners, miner)
		}
	}

	m.miners = newMiners

	return nil
}

func (m *MinerManagerForMySQL) Update(ctx context.Context, skip, limit int64) ([]dtypes.MinerInfo, error) {
	return nil, nil
}

func (m *MinerManagerForMySQL) Count(ctx context.Context) int {
	m.lk.Lock()
	defer m.lk.Unlock()

	return len(m.miners)
}

var _ minermanage.MinerManageAPI = &MinerManagerForMySQL{}
