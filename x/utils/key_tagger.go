package utils

import (
	"encoding/hex"
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/tendermint/tendermint/libs/common"
)

// KeyTagger is a decorate that records all Set/Delete
// operations performed by it's children and adds all those keys
// as DeliverTx tags.
//
// Tags is the hex encoded key, value is "s" (for set) or
// "d" (for delete)
//
// Desired behavior, impossible as tendermint will collapse
// multiple tags with same key:
//   Tags are added as Key=<bucket name>, Value=<hex of remainder>,
//   like Key=cash, Value=00CAFE00
type KeyTagger struct{}

var _ weave.Decorator = KeyTagger{}

// NewKeyTagger creates a KeyTagger decorator
func NewKeyTagger() KeyTagger {
	return KeyTagger{}
}

// Check does nothing
func (KeyTagger) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	return next.Check(ctx, db, tx)
}

// Deliver passes in a recording KVStore into the child and
// uses that to calculate tags to add to DeliverResult
func (KeyTagger) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	record := store.NewRecordingStore(db)
	res, err := next.Deliver(ctx, record, tx)
	if err != nil {
		return nil, err
	}
	res.Tags = append(res.Tags, kvPairs(record)...)
	return res, nil
}

// kvPairs will get the kvpairs from an underlying store if possible
// use this, so we can use interface for recordingStore
func kvPairs(db weave.KVStore) common.KVPairs {
	r, ok := db.(store.Recorder)
	if !ok {
		return nil
	}
	return changesToTags(r.KVPairs())
}

func changesToTags(changes map[string][]byte) common.KVPairs {
	l := len(changes)
	if l == 0 {
		return nil
	}
	res := make(common.KVPairs, 0, l)
	for k, v := range changes {
		key := strings.ToUpper(hex.EncodeToString([]byte(k)))
		value := []byte{'s'}
		if v == nil {
			value = []byte{'d'}
		}
		pair := common.KVPair{
			Key:   []byte(key),
			Value: value,
		}
		res = append(res, pair)
	}
	res.Sort()
	return res
}
