package repo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/ipfs/go-datastore"
	fslock "github.com/ipfs/go-fs-lock"
	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/venus-miner/node/config"
)

const (
	fsAPI       = "api"
	fsAPIToken  = "token"
	fsConfig    = "config.toml"
	fsDatastore = "datastore"
	fsVersion   = "version"
	fsLock      = "repo.lock"
)

var log = logging.Logger("repo")

var ErrRepoExists = errors.New("repo exists")

// FsRepo is struct for repo, use NewFS to create
type FsRepo struct {
	path       string
	configPath string
}

var _ Repo = &FsRepo{}

// NewFS creates a repo instance based on a path on file system
func NewFS(path string) (*FsRepo, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	return &FsRepo{
		path:       path,
		configPath: filepath.Join(path, fsConfig),
	}, nil
}

func (fsr *FsRepo) SetConfigPath(cfgPath string) {
	fsr.configPath = cfgPath
}

func (fsr *FsRepo) Exists() (bool, error) { //nolint
	var err error
	_, err = os.Stat(filepath.Join(fsr.path, fsConfig))
	notexist := os.IsNotExist(err)
	return !notexist, nil
}

func (fsr *FsRepo) Init() error {
	exist, err := fsr.Exists()
	if err != nil {
		return err
	}
	if exist {
		return nil
	}

	log.Infof("Initializing repo at '%s'", fsr.path)
	err = os.MkdirAll(fsr.path, 0755) //nolint: gosec
	if err != nil && !os.IsExist(err) {
		return err
	}

	if err := fsr.initConfig(); err != nil {
		return fmt.Errorf("init config: %w", err)
	}
	return nil
}

func (fsr *FsRepo) initConfig() error {
	_, err := os.Stat(fsr.configPath)
	if err == nil {
		// exists
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	c, err := os.Create(fsr.configPath)
	if err != nil {
		return err
	}

	comm, err := config.ConfigComment(config.DefaultMinerConfig())
	if err != nil {
		return fmt.Errorf("comment: %w", err)
	}
	_, err = c.Write(comm)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	if err := c.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	return nil
}

func (fsr *FsRepo) Update(cfg *config.MinerConfig) error {
	f, err := os.OpenFile(fsr.configPath, os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	comm, err := config.ConfigComment(cfg)
	if err != nil {
		return fmt.Errorf("comment: %w", err)
	}
	_, err = f.Write(comm)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}

	return nil

}

// APIEndpoint returns endpoint of API in this repo
func (fsr *FsRepo) APIEndpoint() (multiaddr.Multiaddr, error) {
	p := filepath.Join(fsr.path, fsAPI)

	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return nil, ErrNoAPIEndpoint
	} else if err != nil {
		return nil, err
	}
	defer f.Close() //nolint: errcheck // Read only op

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %w", p, err)
	}
	strma := string(data)
	strma = strings.TrimSpace(strma)

	apima, err := multiaddr.NewMultiaddr(strma)
	if err != nil {
		return nil, err
	}
	return apima, nil
}

func (fsr *FsRepo) APIToken() ([]byte, error) {
	p := filepath.Join(fsr.path, fsAPIToken)
	f, err := os.Open(p)

	if os.IsNotExist(err) {
		//return nil, ErrNoAPIEndpoint
		log.Warnf("api token not exit , wont use token auth")
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close() //nolint: errcheck // Read only op

	tb, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.TrimSpace(tb), nil
}

func (fsr *FsRepo) Config() (interface{}, error) {
	return config.FromFile(fsr.configPath, config.DefaultMinerConfig())
}

// Lock acquires exclusive lock on this repo
func (fsr *FsRepo) Lock() (LockedRepo, error) {
	locked, err := fslock.Locked(fsr.path, fsLock)
	if err != nil {
		return nil, fmt.Errorf("could not check lock status: %w", err)
	}
	if locked {
		return nil, ErrRepoAlreadyLocked
	}

	closer, err := fslock.Lock(fsr.path, fsLock)
	if err != nil {
		return nil, fmt.Errorf("could not lock the repo: %w", err)
	}
	return &fsLockedRepo{
		path:       fsr.path,
		configPath: fsr.configPath,
		closer:     closer,
	}, nil
}

type fsLockedRepo struct {
	path       string
	configPath string
	closer     io.Closer
	readonly   bool

	ds     map[string]datastore.Batching
	dsErr  error
	dsOnce sync.Once

	configLk sync.Mutex
}

func (fsr *fsLockedRepo) Path() string {
	return fsr.path
}

func (fsr *fsLockedRepo) Close() error {
	if fsr.ds != nil {
		for _, ds := range fsr.ds {
			if err := ds.Close(); err != nil {
				return fmt.Errorf("could not close datastore: %w", err)
			}
		}
	}

	var err error
	if fsr.closer != nil {
		err = fsr.closer.Close()
		fsr.closer = nil
	}

	return err
}

// join joins path elements with fsr.path
func (fsr *fsLockedRepo) join(paths ...string) string {
	return filepath.Join(append([]string{fsr.path}, paths...)...)
}

func (fsr *fsLockedRepo) stillValid() error {
	if fsr.closer == nil {
		return ErrClosedRepo
	}
	return nil
}

func (fsr *fsLockedRepo) Config() (interface{}, error) {
	fsr.configLk.Lock()
	defer fsr.configLk.Unlock()

	return fsr.loadConfigFromDisk()
}

func (fsr *fsLockedRepo) loadConfigFromDisk() (interface{}, error) {
	return config.FromFile(fsr.configPath, config.DefaultMinerConfig())
}

func (fsr *fsLockedRepo) SetConfig(c func(interface{})) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}

	fsr.configLk.Lock()
	defer fsr.configLk.Unlock()

	cfg, err := fsr.loadConfigFromDisk()
	if err != nil {
		return err
	}

	// mutate in-memory representation of config
	c(cfg)

	// buffer into which we write TOML bytes
	buf := new(bytes.Buffer)

	// encode now-mutated config as TOML and write to buffer
	err = toml.NewEncoder(buf).Encode(cfg)
	if err != nil {
		return err
	}

	// write buffer of TOML bytes to config file
	err = os.WriteFile(fsr.configPath, buf.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (fsr *fsLockedRepo) SetAPIEndpoint(ma multiaddr.Multiaddr) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}
	return os.WriteFile(fsr.join(fsAPI), []byte(ma.String()), 0644)
}

func (fsr *fsLockedRepo) SetVersion(version string) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}
	return os.WriteFile(fsr.join(fsVersion), []byte(version), 0644)
}

func (fsr *fsLockedRepo) SetAPIToken(token []byte) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}
	return os.WriteFile(fsr.join(fsAPIToken), token, 0600)
}
