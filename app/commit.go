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
func (cs *CommitStore) CommitInfo() (weave.CommitID, error) {
	return cs.committed.LatestVersion()
}

// Commit will flush deliver to the underlying store and commit it
// to disk. It then regenerates new deliver/check caches
//
// TODO: this should probably be protected by a mutex....
// need to think what concurrency we expect
func (cs *CommitStore) Commit() (weave.CommitID, error) {
	// flush deliver to store and discard check
	if err := cs.deliver.Write(); err != nil {
		return weave.CommitID{}, err
	}
	cs.check.Discard()

	// write the store to disk
	res, err := cs.committed.Commit()
	if err != nil {
		return res, err
	}

	// set up new caches
	cs.deliver = cs.committed.CacheWrap()
	cs.check = cs.committed.CacheWrap()
	return res, nil
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

// mustLoadChainID returns the chain id stored if any
// panics on db error
func mustLoadChainID(kv weave.KVStore) string {
	v, err := kv.Get([]byte(chainIDKey))
	if err != nil {
		panic(err)
	}
	return string(v)
}

// saveChainID stores a chain id in the kv store.
// Returns error if already set, or invalid name
func saveChainID(kv weave.KVStore, chainID string) error {
	if !weave.IsValidChainID(chainID) {
		return errors.Wrapf(errors.ErrInput, "chain id: %v", chainID)
	}
	k := []byte(chainIDKey)
	exists, err := kv.Has(k)
	if err != nil {
		return errors.Wrap(err, "load chainId")
	}
	if exists {
		return errors.Wrap(errors.ErrUnauthorized, "can't modify chain id after genesis init")
	}
	err = kv.Set(k, []byte(chainID))
	if err != nil {
		return errors.Wrap(err, "save chainId")
	}
	return nil
}
