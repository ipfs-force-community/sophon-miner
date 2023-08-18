package minerecorder

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-datastore"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/stretchr/testify/require"
)

var (
	RecordExample = make(map[string]string)
)

func TestDefaultRecorder(t *testing.T) {
	ctx := context.Background()

	t.Run("record and get", func(t *testing.T) {
		db := createDatastore(t)
		r := NewDefaultRecorder(db)
		defer db.Close()

		err := r.Record(ctx, newIDAddress(1), abi.ChainEpoch(0), RecordExample)
		require.NoError(t, err)
		ret, err := r.get(ctx, newIDAddress(1), abi.ChainEpoch(0))
		require.NoError(t, err)
		require.Equal(t, RecordExample, ret)
	})

	t.Run("query record not exist", func(t *testing.T) {
		db := createDatastore(t)
		r := NewDefaultRecorder(db)
		defer db.Close()

		_, err := r.get(ctx, newIDAddress(1), abi.ChainEpoch(9))
		require.ErrorIs(t, err, ErrorRecordNotFound)

		err = r.Record(ctx, newIDAddress(1), abi.ChainEpoch(9), RecordExample)
		require.NoError(t, err)
		err = r.Record(ctx, newIDAddress(1), abi.ChainEpoch(14), RecordExample)
		require.NoError(t, err)

		rets, err := r.Query(ctx, newIDAddress(1), abi.ChainEpoch(5), 10)
		require.NoError(t, err)
		require.Equal(t, 2, len(rets))
	})

	t.Run("query exceed MaxRecordPerQuery", func(t *testing.T) {
		db := createDatastore(t)
		r := NewDefaultRecorder(db)
		defer db.Close()

		_, err := r.Query(ctx, newIDAddress(1), abi.ChainEpoch(0), MaxRecordPerQuery+1)
		require.ErrorIs(t, err, ErrorExceedMaxRecordPerQuery)
	})

	t.Run("auto clean", func(t *testing.T) {
		ExpireEpoch = abi.ChainEpoch(10)
		db := createDatastore(t)
		r := NewDefaultRecorder(db)
		defer db.Close()

		// pre fill
		err := r.Record(ctx, newIDAddress(1), abi.ChainEpoch(0), RecordExample)
		require.NoError(t, err)
		err = r.Record(ctx, newIDAddress(2), abi.ChainEpoch(0), RecordExample)
		require.NoError(t, err)

		ret, err := r.get(ctx, newIDAddress(1), abi.ChainEpoch(0))
		require.NoError(t, err)
		require.Equal(t, RecordExample, ret)

		// approach expiration
		err = r.Record(ctx, newIDAddress(1), abi.ChainEpoch(ExpireEpoch), RecordExample)
		require.NoError(t, err)

		ret, err = r.get(ctx, newIDAddress(1), abi.ChainEpoch(0))
		require.NoError(t, err)
		require.Equal(t, RecordExample, ret)

		// exceed expiration
		err = r.Record(ctx, newIDAddress(1), abi.ChainEpoch(ExpireEpoch+1), RecordExample)
		require.NoError(t, err)

		_, err = r.get(ctx, newIDAddress(1), abi.ChainEpoch(0))
		require.ErrorIs(t, err, ErrorRecordNotFound)
		_, err = r.get(ctx, newIDAddress(2), abi.ChainEpoch(0))
		require.ErrorIs(t, err, ErrorRecordNotFound)

		ret, err = r.get(ctx, newIDAddress(1), abi.ChainEpoch(ExpireEpoch))
		require.NoError(t, err)
		require.Equal(t, RecordExample, ret)
	})

	t.Run("sub recorder", func(t *testing.T) {
		db := createDatastore(t)
		SetDatastore(db)

		rcd := Sub(newIDAddress(1), abi.ChainEpoch(0))
		rcd.Record(ctx, Records{"key": "val"})

		ret, err := Query(ctx, newIDAddress(1), abi.ChainEpoch(0), 1)
		require.NoError(t, err)
		require.Equal(t, 1, len(ret))
		require.Equal(t, "val", ret[0]["key"])
	})
}

func createDatastore(t testing.TB) datastore.Batching {
	path := t.TempDir() + "/leveldb"
	db, err := leveldb.NewDatastore(path, nil)
	require.NoError(t, err)
	return db
}

func newIDAddress(id uint64) address.Address {
	ret, err := address.NewIDAddress(id)
	if err != nil {
		panic("create id address fail")
	}
	return ret
}

func BenchmarkQuery(b *testing.B) {
	ctx := context.Background()
	db := createDatastore(b)
	r := NewDefaultRecorder(db)
	defer db.Close()

	for i := 0; i < 1000*000*1000; i++ {
		err := r.Record(ctx, newIDAddress(1), abi.ChainEpoch(i), RecordExample)
		require.NoError(b, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := r.Query(ctx, newIDAddress(1), abi.ChainEpoch(0), 2000)
		require.NoError(b, err)
	}
}

func TestRWMutex(t *testing.T) {
	lk := sync.RWMutex{}

	for i := 0; i < 10; i++ {
		go read(&lk, i)
	}
	for i := 0; i < 2; i++ {
		go write(&lk, i)
	}
	time.Sleep(time.Second * 1)
	for i := 0; i < 10; i++ {
		go read(&lk, i+10)
	}

	time.Sleep(time.Second * 10)
}

func read(lk *sync.RWMutex, i int) {
	time.Sleep(time.Millisecond * 100 * time.Duration(i))
	fmt.Println("request read lock", i, time.Now())
	lk.RLock()
	defer lk.RUnlock()
	println("read", i, time.Now().String())
	time.Sleep(time.Second * 2)
}

func write(lk *sync.RWMutex, i int) {
	time.Sleep(time.Millisecond * 100 * time.Duration(i))
	fmt.Println("request write lock", i, time.Now())
	lk.Lock()
	defer lk.Unlock()
	println("write", i, time.Now().String())
	time.Sleep(time.Second * 2)
}
