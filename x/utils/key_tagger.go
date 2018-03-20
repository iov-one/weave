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

	res.Tags = append(res.Tags, record.KVPairs()...)
	return res, nil
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
func newRecordingStore(db weave.KVStore) *recordingStore {
	return &recordingStore{
		KVStore: db,
		changes: make(map[string][]byte),
	}
}

// KVPairs returns the content of changes as KVPairs
// Key is the merkle store key that changes.
// Value is "s" or "d" for set or delete.
//
// TODO: return new value???
func (r *recordingStore) KVPairs() []common.KVPair {
	l := len(r.changes)
	if l == 0 {
		return nil
	}
	res := make([]common.KVPair, 0, l)
	for k, v := range r.changes {
		pair := common.KVPair{
			Key:   []byte(k),
			Value: v,
		}
		res = append(res, pair)
	}
	return res
}

// Set records the changes while performing
func (r *recordingStore) Set(key, value []byte) {
	r.changes[string(key)] = recordSet
	r.KVStore.Set(key, value)
}

// Delete records the changes while performing
func (r *recordingStore) Delete(key []byte) {
	r.changes[string(key)] = recordDelete
	r.KVStore.Delete(key)
}
