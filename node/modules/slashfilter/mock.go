package slashfilter

import (
	"github.com/ipfs/go-datastore"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewMysqlMock() (SlashFilterAPI, *gorm.DB, error) {
	db, err := gorm.Open(
		sqlite.Open(":memory:"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})

	if err != nil {
		return nil, nil, err
	}

	sqlDb, err := db.DB()
	if err != nil {
		return nil, nil, err
	}

	// mock db work only allow one connection
	sqlDb.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&MinedBlock{}); err != nil {
		return nil, nil, err
	}

	return &mysqlSlashFilter{_db: db}, db, nil
}

func NewLocalMock() (SlashFilterAPI, datastore.Datastore, error) {
	ds := datastore.NewMapDatastore()
	return NewLocal(ds), ds, nil
}
