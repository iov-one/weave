package app

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	abci "github.com/tendermint/tendermint/abci/types"
)

// Queryable is implemented by an abci.Application, but also weave/client.Client (over tendermint)
type Queryable interface {
	Query(query abci.RequestQuery) abci.ResponseQuery
}

// ABCIStore exposes the weave abci.Query interface as a ReadonlyKVStore
type ABCIStore struct {
	app Queryable
}

var _ weave.ReadOnlyKVStore = (*ABCIStore)(nil)

func NewABCIStore(app Queryable) *ABCIStore {
	return &ABCIStore{app: app}
}

// Get will query for exactly one value over the abci store.
// This can be wrapped with a bucket to reuse key/index/parse logic
func (a *ABCIStore) Get(key []byte) ([]byte, error) {
	query := a.app.Query(abci.RequestQuery{
		Path: "/",
		Data: key,
	})
	// if only the interface supported returning errors....
	if query.Code != 0 {
		return nil, errors.Wrap(errors.ErrDatabase, query.Log)
	}
	var value ResultSet
	if err := value.Unmarshal(query.Value); err != nil {
		return nil, errors.Wrapf(errors.ErrState, "unmarshal result set: %v", err.Error())
	}
	if len(value.Results) == 0 {
		return nil, nil
	}
	// TODO: assert error if len > 1 ???
	return value.Results[0], nil
}

// Has returns true if the given key in in the abci app store
func (a *ABCIStore) Has(key []byte) (bool, error) {
	got, err := a.Get(key)
	if err != nil {
		return false, err
	}
	return len(got) > 0, nil
}

// Iterator attempts to do a range iteration over the store,
// We only support prefix queries in the abci server for now.
// This client only supports listing everything...
func (a *ABCIStore) Iterator(start, end []byte) (weave.Iterator, error) {
	// TODO: support all prefix searches (later even more ranges)
	// look at orm/query.go:prefixRange for an idea how we turn prefix->iterator,
	// we should detect this case and reverse it so we can serialize over abci query
	if start != nil || end != nil {
		return nil, errors.Wrap(errors.ErrDatabase, "iterator only implemented for entire range")
	}

	query := a.app.Query(abci.RequestQuery{
		Path: "/?prefix",
		Data: nil,
	})
	if query.Code != 0 {
		return nil, errors.Wrap(errors.ErrDatabase, query.Log)
	}
	models, err := toModels(query.Key, query.Value)
	if err != nil {
		return nil, err
	}

	return store.NewSliceIterator(models), nil
}

func (a *ABCIStore) ReverseIterator(start, end []byte) (weave.Iterator, error) {
	// TODO: load normal iterator but then play it backwards?
	return nil, errors.Wrap(errors.ErrDatabase, "not implemented")
}

func toModels(keys, values []byte) ([]weave.Model, error) {
	var k, v ResultSet
	if err := k.Unmarshal(keys); err != nil {
		return nil, errors.Wrapf(errors.ErrState, "cannot unmarshal keys: %v", err.Error())
	}
	if err := v.Unmarshal(values); err != nil {
		return nil, errors.Wrapf(errors.ErrState, "cannot unmarshal values: %v", err.Error())
	}
	return JoinResults(&k, &v)
}
