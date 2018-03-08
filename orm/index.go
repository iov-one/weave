package orm

import (
	"bytes"

	"github.com/pkg/errors"

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
	name   string
	id     []byte
	unique bool
	index  Indexer
}

// NewIndex constructs an index
func NewIndex(name string, indexer Indexer, unique bool) Index {
	// TODO: index name must be [a-z_]
	return Index{
		name:   name,
		id:     append(indPrefix, []byte(name+":")...),
		index:  indexer,
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
		key, err := i.index(save)
		if err != nil {
			return err
		}
		return i.insert(db, key, save.Key())
	case s{false, true}:
		key, err := i.index(prev)
		if err != nil {
			return err
		}
		return i.remove(db, key, prev.Key())
	case s{false, false}:
		return i.move(db, prev, save)
	}
	return errors.New("You have violated the rules of boolean logic")
}

// GetLike calculates the index for the given pattern, and
// returns a list of all pk that match (may be empty), or an error
func (i Index) GetLike(db weave.KVStore, pattern Object) ([][]byte, error) {
	index, err := i.index(pattern)
	if err != nil {
		return nil, err
	}
	return i.GetAt(db, index)
}

// GetAt returns a list of all pk at that index (may be empty), or an error
func (i Index) GetAt(db weave.KVStore, index []byte) ([][]byte, error) {
	key := append(i.id, index...)
	val := db.Get(key)
	if val == nil {
		return nil, nil
	}
	if i.unique {
		return [][]byte{val}, nil
	}
	var data = new(MultiRef)
	err := data.Unmarshal(val)
	if err != nil {
		return nil, err
	}
	return data.GetRefs(), nil
}

func (i Index) move(db weave.KVStore, prev Object, save Object) error {
	// if the primary key is not equal, we have a problem
	if !bytes.Equal(prev.Key(), save.Key()) {
		// TODO: do we want to handle this????
		return errors.New("Can only update Index for objects with same primary key")
	}

	// if the keys don't change, then
	oldKey, err := i.index(prev)
	if err != nil {
		return err
	}
	newKey, err := i.index(save)
	if err != nil {
		return err
	}
	if bytes.Equal(oldKey, newKey) {
		return nil
	}

	// check unique constraint before removing
	if i.unique {
		k := append(i.id, newKey...)
		val := db.Get(k)
		if val != nil {
			return errors.Errorf("Duplicate violates unique constraint on index %s", i.name)
		}
	}

	err = i.remove(db, oldKey, prev.Key())
	if err != nil {
		return err
	}
	return i.insert(db, newKey, save.Key())
}

func (i Index) remove(db weave.KVStore, index []byte, pk []byte) error {
	key := append(i.id, index...)
	cur := db.Get(key)
	if cur == nil {
		return errors.New("Try to remove at empty index")
	}
	if i.unique {
		// if something else was here, don't delete
		if !bytes.Equal(cur, pk) {
			return errors.New("Can't remove reference to other object")
		}
		db.Delete(key)
		return nil
	}

	// otherwise, remove one from a list....
	var data = new(MultiRef)
	err := data.Unmarshal(cur)
	if err != nil {
		return err
	}
	err = data.Remove(pk)
	if err != nil {
		return err
	}
	// nothing left, delete this key
	if data.Size() == 0 {
		db.Delete(key)
		return nil
	}
	// other left, just update state
	save, err := data.Marshal()
	if err != nil {
		return err
	}
	db.Set(key, save)
	return nil
}

func (i Index) insert(db weave.KVStore, index []byte, pk []byte) error {
	key := append(i.id, index...)
	cur := db.Get(key)

	if i.unique {
		if cur != nil {
			return errors.Errorf("Duplicate violates unique constraint on index %s", i.name)
		}
		db.Set(key, pk)
		return nil
	}

	// otherwise, add one to a list....
	var data = new(MultiRef)
	if cur != nil {
		err := data.Unmarshal(cur)
		if err != nil {
			return err
		}
	}
	err := data.Add(pk)
	if err != nil {
		return err
	}

	// other left, just update state
	save, err := data.Marshal()
	if err != nil {
		return err
	}
	db.Set(key, save)
	return nil
}
