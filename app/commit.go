package app

import (
	"github.com/confio/weave"
)

// commitStore is an internal type to handle loading from a
// KVCommitStore, maintaining different CacheWraps for
// Deliver and Check, and returning useful state info.
type commitStore struct {
	committed weave.CommitKVStore
	deliver   weave.KVCacheWrap
	check     weave.KVCacheWrap
}

// newCommitStore loads the CommitKVStore from disk or panics
// Sets up the deliver and check caches
//
// TODO: where is chain?????
func newCommitStore(store weave.CommitKVStore) *commitStore {
	err := store.LoadLatestVersion()
	if err != nil {
		panic(err)
	}
	// TODO: get chain ID???? or from where????
	return &commitStore{
		committed: store,
		deliver:   store.CacheWrap(),
		check:     store.CacheWrap(),
	}
}

// CommitInfo returns the current height and hash
func (cs *commitStore) CommitInfo() (version int64, hash []byte) {
	id := cs.committed.LatestVersion()
	return id.Version, id.Hash
}

// Commit will flush deliver to the underlying store and commit it
// to disk. It then regenerates new deliver/check caches
//
// TODO: this should probably be protected by a mutex....
// need to think what concurrency we expect
func (cs *commitStore) Commit() weave.CommitID {
	// flush deliver to store and discard check
	cs.deliver.Write()
	cs.check.Discard()

	// write the store to disk
	res := cs.committed.Commit()
	// // TODO: release an old version
	// if version > s.historySize {
	//   s.committed.Tree.DeleteVersion(uint64(version - s.historySize))
	// }

	// set up new caches
	cs.deliver = cs.committed.CacheWrap()
	cs.check = cs.committed.CacheWrap()
	return res
}
