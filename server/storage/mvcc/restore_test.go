package mvcc

import (
	"fmt"
	"math"
	"sync"
	"testing"

	"go.etcd.io/etcd/server/v3/storage/backend"
	betesting "go.etcd.io/etcd/server/v3/storage/backend/testing"
	"go.etcd.io/etcd/server/v3/storage/schema"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestRestore(t *testing.T) {
	lg := zaptest.NewLogger(t)
	cfg := backend.DefaultBackendConfig(lg)
	cfg.Path = "/home/skyjiang/go-project/etcd-badger/bin/etcd.etcd/member/snap/db"

	b := betesting.NewBackendFromCfg(t, cfg)

	min, max := newRevBytes(), newRevBytes()
	revToBytes(revision{main: 1}, min)
	revToBytes(revision{main: math.MaxInt64, sub: math.MaxInt64}, max)

	tx := b.ReadTx()
	tx.RLock()
	defer tx.RUnlock()

	rkvc := make(chan revKeyValue, restoreChunkKeys)

	var wg sync.WaitGroup
	wg.Add(1)

	idx := NewTreeIndex(lg)
	go restoreIndex(lg, rkvc, idx, &wg)

	for {
		keys, vals := tx.UnsafeRange(schema.Key, min, max, int64(restoreChunkKeys))
		if len(keys) == 0 {
			break
		}

		// rkvc blocks if the total pending keys exceeds the restore
		// chunk size to keep keys from consuming too much memory.
		for i, key := range keys {
			rkv := revKeyValue{key: key}
			if err := rkv.kv.Unmarshal(vals[i]); err != nil {
				fmt.Printf("failed to unmarshal mvccpb.KeyValue, %s\n", err.Error())
			}
			rkv.kstr = string(rkv.kv.Key)
			/* if isTombstone(key) {
				delete(keyToLease, rkv.kstr)
			} else if lid := lease.LeaseID(rkv.kv.Lease); lid != lease.NoLease {
				keyToLease[rkv.kstr] = lid
			} else {
				delete(keyToLease, rkv.kstr)
			} */
			rkvc <- rkv
		}

		if len(keys) < restoreChunkKeys {
			// partial set implies final set
			break
		}
		// next set begins after where this one ended
		newMin := bytesToRev(keys[len(keys)-1][:revBytesLen])
		newMin.sub++
		revToBytes(newMin, min)
	}

	wg.Wait()
}


func restoreIndex(lg *zap.Logger, rkvc <-chan revKeyValue, idx index, wg *sync.WaitGroup) {
	// restore the tree index from streaming the unordered index.
	kiCache := make(map[string]*keyIndex, restoreChunkKeys)
	wg.Done()

	for rkv := range rkvc {
		fmt.Printf("--> %s : %d\n", rkv.kstr, rkv.kv.ModRevision)
		ki, ok := kiCache[rkv.kstr]
		// purge kiCache if many keys but still missing in the cache
		if !ok && len(kiCache) >= restoreChunkKeys {
			i := 10
			for k := range kiCache {
				delete(kiCache, k)
				if i--; i == 0 {
					break
				}
			}
		}
		// cache miss, fetch from tree index if there
		if !ok {
			ki = &keyIndex{key: rkv.kv.Key}
			if idxKey := idx.KeyIndex(ki); idxKey != nil {
				kiCache[rkv.kstr], ki = idxKey, idxKey
				ok = true
			}
		}
		rev := bytesToRev(rkv.key)
		if ok {
			if isTombstone(rkv.key) {
				if err := ki.tombstone(lg, rev.main, rev.sub); err != nil {
					lg.Warn("tombstone encountered error", zap.Error(err))
				}
				continue
			}
			ki.put(lg, rev.main, rev.sub)
		} else if !isTombstone(rkv.key) {
			ki.restore(lg, revision{rkv.kv.CreateRevision, 0}, rev, rkv.kv.Version)
			idx.Insert(ki)
			kiCache[rkv.kstr] = ki
		}
	}
}
