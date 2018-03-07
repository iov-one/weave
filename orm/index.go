package orm

import (
	"bytes"
	"errors"

	"github.com/confio/weave"
)

var indPrefix = []byte("_i:")

// Indexer calculates the secondary index key for a given object
type Indexer func(Object) ([]byte, error)

// Index represents a secondary index on some data.
// It is indexed by an arbitrary key returned by Indexer.
// The value is one primary key (unique),
// Or an array of primary keys (!unique).
type Index struct {
	id     []byte
	unique bool
	key    Indexer
}

// NewIndex constructs an index
func NewIndex(name string, indexer Indexer, unique bool) Index {
	// TODO: index name must be [a-z_]
	return Index{
		id:     append(indPrefix, []byte(name+":")...),
		key:    indexer,
		unique: unique,
	}
}

// Update handles updating the reference to the object in
// the secondary index.
//
// prev == nil means insert
// save == nil means delete
// both == nil is error
// if both != nil and prev.Key() != save.Key() this is an error
//
// Otherwise, it will check indexer(prev) and indexer(save)
// and make sure the key is now stored in the right location
func (i Index) Update(db weave.KVStore, prev Object, save Object) error {
	type s struct{ a, b bool }
	sw := s{prev == nil, save == nil}
	switch sw {
	case s{true, true}:
		return errors.New("update requires at least one non-nil object")
	case s{true, false}:
		key, err := i.key(save)
		if err != nil {
			return err
		}
		return i.insert(db, key, save.Key())
	case s{false, true}:
		key, err := i.key(prev)
		if err != nil {
			return err
		}
		return i.remove(db, key, prev.Key())
	case s{false, false}:
		return i.move(db, prev, save)
	}
	return errors.New("You have violated the rules of boolean logic")
}

func (i Index) move(db weave.KVStore, prev Object, save Object) error {
	// if the primary key is not equal, we have a problem
	if !bytes.Equal(prev.Key(), save.Key()) {
		// TODO: do we want to handle this????
		return errors.New("Can only update Index for objects with same primary key")
	}

	// if the keys don't change, then
	oldKey, err := i.key(prev)
	if err != nil {
		return err
	}
	newKey, err := i.key(save)
	if err != nil {
		return err
	}
	if bytes.Equal(oldKey, newKey) {
		return nil
	}

	err = i.remove(db, oldKey, prev.Key())
	if err != nil {
		return err
	}
	return i.insert(db, newKey, save.Key())
}

func (i Index) remove(db weave.KVStore, index []byte, pk []byte) error {
	// TODO
	return nil
}

func (i Index) insert(db weave.KVStore, index []byte, pk []byte) error {
	// TODO
	return nil
}
