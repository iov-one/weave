package utils

import (
	"github.com/tendermint/tmlibs/common"

	"github.com/confio/weave"
)

// KeyTagger is a decorate that records all Set/Delete
// operations performed by it's children and adds all those keys
// as DeliverTx tags
type KeyTagger struct{}

var _ weave.Decorator = KeyTagger{}

// NewKeyTagger creates a KeyTagger decorator
func NewKeyTagger() KeyTagger {
	return KeyTagger{}
}

// Check does nothing
func (KeyTagger) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Checker) (weave.CheckResult, error) {
	return next.Check(ctx, store, tx)
}

// Deliver passes in a recording KVStore into the child and
// uses that to calculate tags to add to DeliverResult
func (KeyTagger) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (weave.DeliverResult, error) {

	record := newRecordingStore(store)
	res, err := next.Deliver(ctx, record, tx)
	if err != nil {
		return res, err
	}

	res.Tags = append(res.Tags, kvPairs(record)...)
	return res, nil
}

type kvpairer interface {
	KVPairs() common.KVPairs
}

// kvPairs will get the kvpairs from an underlying store if possible
// use this, so we can use interface for recordingStore
func kvPairs(db weave.KVStore) common.KVPairs {
	r, ok := db.(kvpairer)
	if !ok {
		return nil
	}
	return r.KVPairs()
}

var (
	recordSet    = []byte("s")
	recordDelete = []byte("d")
)

// recordingStore wraps a normal KVStore and records any change operations
type recordingStore struct {
	weave.KVStore
	// changes is a map from key to (recordSet|recordDelete)
	changes map[string][]byte
}

var _ weave.KVStore = (*recordingStore)(nil)

// newRecordingStore initializes a recording store wrapping this
// base store
//
// TODO: return CacheableKVStore if possible
func newRecordingStore(db weave.KVStore) weave.KVStore {
	return &recordingStore{
		KVStore: db,
		changes: make(map[string][]byte),
	}
}

// KVPairs returns the content of changes as KVPairs
// Key is the merkle store key that changes.
// Value is "s" or "d" for set or delete.
func (r *recordingStore) KVPairs() common.KVPairs {
	l := len(r.changes)
	if l == 0 {
		return nil
	}
	res := make(common.KVPairs, 0, l)
	for k, v := range r.changes {
		pair := common.KVPair{
			Key:   []byte(k),
			Value: v,
		}
		res = append(res, pair)
	}
	res.Sort()
	return res
}

// Set records the changes while performing
//
// TODO: record new value???
func (r *recordingStore) Set(key, value []byte) {
	r.changes[string(key)] = recordSet
	r.KVStore.Set(key, value)
}

// Delete records the changes while performing
func (r *recordingStore) Delete(key []byte) {
	r.changes[string(key)] = recordDelete
	r.KVStore.Delete(key)
}
