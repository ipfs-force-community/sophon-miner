package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	types2 "github.com/filecoin-project/venus-miner/types"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/ipfs/go-datastore"
	fslock "github.com/ipfs/go-fs-lock"
	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-base32"
	"github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/venus-miner/node/config"
	"github.com/filecoin-project/venus/venus-shared/types"
)

const (
	fsAPI       = "api"
	fsAPIToken  = "token"
	fsConfig    = "config.toml"
	fsDatastore = "datastore"
	fsLock      = "repo.lock"
	fsKeystore  = "keystore"
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
	_, err = os.Stat(filepath.Join(fsr.path, fsDatastore))
	notexist := os.IsNotExist(err)
	if notexist {
		_, err = os.Stat(filepath.Join(fsr.path, fsKeystore))
		notexist = os.IsNotExist(err)
		if notexist {
			err = nil
		}
	}
	return !notexist, err
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

	return fsr.initKeystore()

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
	f, err := os.Open(fsr.configPath)
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

func (fsr *FsRepo) initKeystore() error {
	kstorePath := filepath.Join(fsr.path, fsKeystore)
	if _, err := os.Stat(kstorePath); err == nil {
		return ErrRepoExists
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Mkdir(kstorePath, 0700)
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

	data, err := ioutil.ReadAll(f)
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

	tb, err := ioutil.ReadAll(f)
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
	err = ioutil.WriteFile(fsr.configPath, buf.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (fsr *fsLockedRepo) SetAPIEndpoint(ma multiaddr.Multiaddr) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}
	return ioutil.WriteFile(fsr.join(fsAPI), []byte(ma.String()), 0644)
}

func (fsr *fsLockedRepo) SetAPIToken(token []byte) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}
	return ioutil.WriteFile(fsr.join(fsAPIToken), token, 0600)
}

func (fsr *fsLockedRepo) KeyStore() (types2.KeyStore, error) {
	if err := fsr.stillValid(); err != nil {
		return nil, err
	}
	return fsr, nil
}

var kstrPermissionMsg = "permissions of key: '%s' are too relaxed, " +
	"required: 0600, got: %#o"

// List lists all the keys stored in the KeyStore
func (fsr *fsLockedRepo) List() ([]string, error) {
	if err := fsr.stillValid(); err != nil {
		return nil, err
	}

	kstorePath := fsr.join(fsKeystore)
	dir, err := os.Open(kstorePath)
	if err != nil {
		return nil, fmt.Errorf("opening dir to list keystore: %w", err)
	}
	defer dir.Close() //nolint:errcheck
	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("reading keystore dir: %w", err)
	}
	keys := make([]string, 0, len(files))
	for _, f := range files {
		if f.Mode()&0077 != 0 {
			return nil, fmt.Errorf(kstrPermissionMsg, f.Name(), f.Mode())
		}
		name, err := base32.RawStdEncoding.DecodeString(f.Name())
		if err != nil {
			return nil, fmt.Errorf("decoding key: '%s': %w", f.Name(), err)
		}
		keys = append(keys, string(name))
	}
	return keys, nil
}

// Get gets a key out of keystore and returns types.KeyInfo coresponding to named key
func (fsr *fsLockedRepo) Get(name string) (types.KeyInfo, error) {
	if err := fsr.stillValid(); err != nil {
		return types.KeyInfo{}, err
	}

	encName := base32.RawStdEncoding.EncodeToString([]byte(name))
	keyPath := fsr.join(fsKeystore, encName)

	fstat, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		return types.KeyInfo{}, fmt.Errorf("opening key '%s': %w", name, types.ErrKeyInfoNotFound)
	} else if err != nil {
		return types.KeyInfo{}, fmt.Errorf("opening key '%s': %w", name, err)
	}

	if fstat.Mode()&0077 != 0 {
		return types.KeyInfo{}, fmt.Errorf(kstrPermissionMsg, name, fstat.Mode())
	}

	file, err := os.Open(keyPath)
	if err != nil {
		return types.KeyInfo{}, fmt.Errorf("opening key '%s': %w", name, err)
	}
	defer file.Close() //nolint: errcheck // read only op

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return types.KeyInfo{}, fmt.Errorf("reading key '%s': %w", name, err)
	}

	var res types.KeyInfo
	err = json.Unmarshal(data, &res)
	if err != nil {
		return types.KeyInfo{}, fmt.Errorf("decoding key '%s': %w", name, err)
	}

	return res, nil
}

// Put saves key info under given name
func (fsr *fsLockedRepo) Put(name string, info types.KeyInfo) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}

	encName := base32.RawStdEncoding.EncodeToString([]byte(name))
	keyPath := fsr.join(fsKeystore, encName)

	_, err := os.Stat(keyPath)
	if err == nil {
		return fmt.Errorf("checking key before put '%s': %w", name, types.ErrKeyExists)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking key before put '%s': %w", name, err)
	}

	keyData, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("encoding key '%s': %w", name, err)
	}

	err = ioutil.WriteFile(keyPath, keyData, 0600)
	if err != nil {
		return fmt.Errorf("writing key '%s': %w", name, err)
	}
	return nil
}

func (fsr *fsLockedRepo) Delete(name string) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}

	encName := base32.RawStdEncoding.EncodeToString([]byte(name))
	keyPath := fsr.join(fsKeystore, encName)

	_, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("checking key before delete '%s': %w", name, types.ErrKeyInfoNotFound)
	} else if err != nil {
		return fmt.Errorf("checking key before delete '%s': %w", name, err)
	}

	err = os.Remove(keyPath)
	if err != nil {
		return fmt.Errorf("deleting key '%s': %w", name, err)
	}
	return nil
}
