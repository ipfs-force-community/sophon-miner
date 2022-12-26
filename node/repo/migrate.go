package repo

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/filecoin-project/venus-miner/build"
	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus-miner/node/config/migrate/v170"
	"github.com/filecoin-project/venus-miner/node/config/migrate/v180"
)

type UpgradeFunc func(*fsLockedRepo) error

type versionInfo struct {
	version int
	upgrade UpgradeFunc
}

var versionMap = []versionInfo{
	{version: 180, upgrade: ToVersion180Upgrade},
}

func ToVersion180Upgrade(fsr *fsLockedRepo) error {
	cfgV, err := config.FromFile(fsr.configPath, &v170.MinerConfig{})
	if err != nil {
		return err
	}
	cfgOld, ok := cfgV.(*v170.MinerConfig)
	if ok {
		// upgrade
		if err := fsr.stillValid(); err != nil {
			return err
		}

		fsr.configLk.Lock()
		defer fsr.configLk.Unlock()

		// mutate in-memory representation of config
		cfgV180 := &v180.MinerConfig{}
		cfgOld.ToMinerConfig(cfgV180)

		// buffer into which we write TOML bytes
		buf := new(bytes.Buffer)

		// encode now-mutated config as TOML and write to buffer
		err = toml.NewEncoder(buf).Encode(cfgV180)
		if err != nil {
			return err
		}

		// write buffer of TOML bytes to config file
		err = ioutil.WriteFile(fsr.configPath, buf.Bytes(), 0644)
		if err != nil {
			return err
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

	actVer, _ := strconv.Atoi(strings.Trim(actVerStr, "\n"))
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
