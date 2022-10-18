package repo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/filecoin-project/venus-miner/build"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/config/migrate"
)

type UpgradeFunc func(*fsLockedRepo) error

type versionInfo struct {
	version int
	upgrade UpgradeFunc
}

var versionMap = []versionInfo{
	{version: 180, upgrade: Version180Upgrade},
}

func Version180Upgrade(fsr *fsLockedRepo) error {
	cfgV, err := config.FromFile(fsr.configPath, &migrate.MinerConfig{})
	if err != nil {
		return err
	}
	cfgV170, ok := cfgV.(*migrate.MinerConfig)

	if ok {
		// upgrade
		if err := fsr.SetConfig(func(i interface{}) {
			cfg := i.(*config.MinerConfig)
			cfgV170.ToMinerConfig(cfg)
		}); err != nil {
			return fmt.Errorf("modify config failed: %w", err)
		}
	}

	return nil
}

func (fsr *fsLockedRepo) Migrate() error {
	filePath := filepath.Join(fsr.path, fsVersion)
	f, err := os.Open(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	actVerStr := "0"
	if err == nil {
		defer f.Close() //nolint: errcheck

		data, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filePath, err)
		}
		actVerStr = string(data)
	}

	actVer, _ := strconv.Atoi(actVerStr)
	for _, up := range versionMap {
		if up.version > actVer {
			if up.upgrade == nil {
				log.Infof("version %d no executable function", up.version)
			} else {
				err = up.upgrade(fsr)
				if err != nil {
					return fmt.Errorf("upgrade version to %d: %w", up.version, err)
				}
				log.Infof("success to upgrade version %d to %d", actVer, up.version)
			}
			actVer = up.version
		}
	}

	err = fsr.SetVersion(build.Version)
	if err != nil {
		return fmt.Errorf("modify version failed: %w", err)
	}

	return nil
}
