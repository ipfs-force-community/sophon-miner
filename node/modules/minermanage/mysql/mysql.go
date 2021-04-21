package mysql

import (
	"fmt"
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
	Addr        string `gorm:"column:addr;type:varchar(20);uniqueIndex;NOT NULL"`
	SealerAPI   string `gorm:"column:sealer_api;type:varchar(256);"`
	SealerToken string `gorm:"column:sealer_token;type:varchar(256);"`
	WalletAPI   string `gorm:"column:wallet_api;type:varchar(256);"`
	WalletToken string `gorm:"column:wallet_token;type:varchar(256);"`
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
	return func() (minermanage.MinerManageAPI, error){
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%s",
			cfg.User,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.DbName,
			"10s")

		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			//Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			return nil, xerrors.Errorf("[db connection failed] Database name: %s %w", cfg.DbName, err)
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
		sqlDB.SetConnMaxLifetime(time.Minute * cfg.ConnMaxLifeTime)

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
	m.lk.Lock()
	defer m.lk.Unlock()

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
			Sealer: dtypes.SealerNode{
				ListenAPI: val.SealerAPI,
				Token:     val.SealerToken,
			},
			Wallet: dtypes.WalletNode{
				ListenAPI: val.WalletAPI,
				Token:     val.WalletToken,
			},
		})
	}

	return nil
}

func (m *MinerManagerForMySQL) Put(miner dtypes.MinerInfo) error {
	m.lk.Lock()
	defer m.lk.Unlock()

	err := m._db.Create(&mysqlMinerInfo{Addr: miner.Addr.String(), SealerAPI: miner.Sealer.ListenAPI,
		SealerToken: miner.Sealer.Token, WalletAPI: miner.Wallet.ListenAPI, WalletToken: miner.Wallet.Token}).Error
	if err != nil {
		return err
	}

	m.miners = append(m.miners, miner)
	return nil
}

func (m *MinerManagerForMySQL) Set(miner dtypes.MinerInfo) error {
	m.lk.Lock()
	defer m.lk.Unlock()

	for k, addr := range m.miners {
		if addr.Addr.String() == miner.Addr.String() {
			if miner.Sealer.ListenAPI != "" && miner.Sealer.ListenAPI != m.miners[k].Sealer.ListenAPI {
				m.miners[k].Sealer.ListenAPI = miner.Sealer.ListenAPI
			}

			if miner.Sealer.Token != "" && miner.Sealer.Token != m.miners[k].Sealer.Token {
				m.miners[k].Sealer.Token = miner.Sealer.Token
			}

			if miner.Wallet.ListenAPI != "" && miner.Wallet.ListenAPI != m.miners[k].Wallet.ListenAPI {
				m.miners[k].Wallet.ListenAPI = miner.Wallet.ListenAPI
			}

			if miner.Wallet.Token != "" && miner.Wallet.Token != m.miners[k].Wallet.Token {
				m.miners[k].Wallet.Token = miner.Wallet.Token
			}

			err := m.Put(miner)
			if err != nil {
				return err
			}
			break
		}
	}

	return nil
}

func (m *MinerManagerForMySQL) Has(addr address.Address) bool {
	for _, miner := range m.miners {
		if miner.Addr.String() == addr.String() {
			return true
		}
	}

	return false
}

func (m *MinerManagerForMySQL) Get(addr address.Address) *dtypes.MinerInfo {
	m.lk.Lock()
	defer m.lk.Unlock()

	for k := range m.miners {
		if m.miners[k].Addr.String() == addr.String() {
			return &m.miners[k]
		}
	}

	return nil
}

func (m *MinerManagerForMySQL) List() ([]dtypes.MinerInfo, error) {
	m.lk.Lock()
	defer m.lk.Unlock()

	return m.miners, nil
}

func (m *MinerManagerForMySQL) Remove(rmAddr address.Address) error {
	m.lk.Lock()
	defer m.lk.Unlock()

	if !m.Has(rmAddr) {
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

func (m *MinerManagerForMySQL) Count() int {
	m.lk.Lock()
	defer m.lk.Unlock()

	return len(m.miners)
}

var _ minermanage.MinerManageAPI = &MinerManagerForMySQL{}
