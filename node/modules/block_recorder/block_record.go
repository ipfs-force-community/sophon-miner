package block_recorder

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-datastore"

	"github.com/filecoin-project/venus-miner/node/modules/dtypes"
)

const (
	Local = "local"
	Cache = "cache"
	MySQL = "mysql"
)

type IBlockRecord interface {
	MarkAsProduced(miner address.Address, height uint64) error
	Has(miner address.Address, height uint64) bool
}

type LocalDBRecord struct {
	da dtypes.MetadataDS
}

func NewLocalDBRecord(da dtypes.MetadataDS) *LocalDBRecord {
	return &LocalDBRecord{da: da}
}

func (l *LocalDBRecord) MarkAsProduced(ctx context.Context, miner address.Address, height uint64) error {
	blkKey := datastore.NewKey(fmt.Sprintf("%s-%d", miner, height))
	return l.da.Put(ctx, blkKey, []byte{1})
}

func (l *LocalDBRecord) Has(ctx context.Context, miner address.Address, height uint64) bool {
	blkKey := datastore.NewKey(fmt.Sprintf("%s-%d", miner, height))
	has, err := l.da.Has(ctx, blkKey)
	if err != nil {
		return false
	}
	return has
}

type CacheRecord struct {
	cache *lru.ARCCache
}

func NewCacheRecord() (*CacheRecord, error) {
	cache, err := lru.NewARC(100000)
	if err != nil {
		return nil, err
	}
	return &CacheRecord{cache: cache}, nil
}

func (c *CacheRecord) MarkAsProduced(miner address.Address, height uint64) error {
	blkKey := datastore.NewKey(fmt.Sprintf("%s-%d", miner, height))
	c.cache.Add(blkKey, true)
	return nil
}

func (c *CacheRecord) Has(miner address.Address, height uint64) bool {
	blkKey := datastore.NewKey(fmt.Sprintf("%s-%d", miner, height))
	_, has := c.cache.Get(blkKey)
	return has
}
