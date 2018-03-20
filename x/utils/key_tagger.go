package utils

import (
	"github.com/tendermint/tmlibs/common"

	"github.com/confio/weave"
	"github.com/confio/weave/store"
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
func (KeyTagger) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx,
	next weave.Checker) (weave.CheckResult, error) {
	return next.Check(ctx, db, tx)
}

// Deliver passes in a recording KVStore into the child and
// uses that to calculate tags to add to DeliverResult
func (KeyTagger) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx,
	next weave.Deliverer) (weave.DeliverResult, error) {

	record := store.NewRecordingStore(db)
	res, err := next.Deliver(ctx, record, tx)
	if err != nil {
		return res, err
	}

	res.Tags = append(res.Tags, kvPairs(record)...)
	return res, nil
}

var (
	recordSet    = []byte("s")
	recordDelete = []byte("d")
)

// kvPairs will get the kvpairs from an underlying store if possible
// use this, so we can use interface for recordingStore
func kvPairs(db weave.KVStore) common.KVPairs {
	r, ok := db.(store.Recorder)
	if !ok {
		return nil
	}
	return changesToTags(r.KVPairs())
}

//----- helpers ---

func changesToTags(changes map[string][]byte) common.KVPairs {
	l := len(changes)
	if l == 0 {
		return nil
	}
	res := make(common.KVPairs, 0, l)
	for k, v := range changes {
		tag := recordSet
		if v == nil {
			tag = recordDelete
		}
		pair := common.KVPair{
			Key:   []byte(k),
			Value: tag,
		}
		res = append(res, pair)
	}
	res.Sort()
	return res
}
