package app

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// CommitStore handles loading from a KVCommitStore, maintaining different
// CacheWraps for Deliver and Check, and returning useful state info.
type CommitStore struct {
	committed weave.CommitKVStore
	deliver   weave.KVCacheWrap
	check     weave.KVCacheWrap
}

// NewCommitStore loads the CommitKVStore from disk or panics. It sets up the
// deliver and check caches.
func NewCommitStore(store weave.CommitKVStore) *CommitStore {
	err := store.LoadLatestVersion()
	if err != nil {
		panic(err)
	}
	return &CommitStore{
		committed: store,
		deliver:   store.CacheWrap(),
		check:     store.CacheWrap(),
	}
}

// CommitInfo returns the current height and hash
func (cs *CommitStore) CommitInfo() (version int64, hash []byte) {
	id := cs.committed.LatestVersion()
	return id.Version, id.Hash
}

// Commit will flush deliver to the underlying store and commit it
// to disk. It then regenerates new deliver/check caches
//
// TODO: this should probably be protected by a mutex....
// need to think what concurrency we expect
func (cs *CommitStore) Commit() weave.CommitID {
	// flush deliver to store and discard check
	cs.deliver.Write()
	cs.check.Discard()

	// write the store to disk
	res := cs.committed.Commit()

	// set up new caches
	cs.deliver = cs.committed.CacheWrap()
	cs.check = cs.committed.CacheWrap()
	return res
}

// CheckStore returns a store implementation that must be used during the
// checking phase.
func (cs *CommitStore) CheckStore() weave.CacheableKVStore {
	return cs.check
}

// DeliverStore returns a store implementation that must be used during the
// delivery phase.
func (cs *CommitStore) DeliverStore() weave.CacheableKVStore {
	return cs.deliver
}

//------- storing chainID ---------

// _wv: is a prefix for weave internal data
const chainIDKey = "_wv:chainID"

// loadChainID returns the chain id stored if any
func loadChainID(kv weave.KVStore) string {
	v := kv.Get([]byte(chainIDKey))
	return string(v)
}

// saveChainID stores a chain id in the kv store.
// Returns error if already set, or invalid name
func saveChainID(kv weave.KVStore, chainID string) error {
	if !weave.IsValidChainID(chainID) {
		return errors.ErrInvalidChainID(chainID)
	}
	k := []byte(chainIDKey)
	if kv.Has(k) {
		return errors.ErrModifyChainID()
	}
	kv.Set(k, []byte(chainID))
	return nil
}
