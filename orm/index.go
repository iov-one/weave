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
	name   string
	id     []byte
	unique bool
	index  Indexer
}

var _ weave.QueryHandler = Index{}

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

// Query handles queries from the QueryRouter
func (i Index) Query(db weave.ReadOnlyKVStore, mod string,
	data []byte) ([]weave.Model, error) {

	switch mod {
	case weave.KeyQueryMod:
		keys, err := i.GetAt(db, data)
		if err != nil {
			return nil, err
		}
		res := make([]weave.Model, len(keys))
		for i, key := range keys {
			res[i] = weave.Model{
				Key:   key,
				Value: db.Get(key),
			}
		}
		return res, nil
	case weave.PrefixQueryMod:
		return nil, errors.New("prefix not yet implemented")
	default:
		return nil, errors.New("no implemented: " + mod)
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
		return ErrUpdateNil()
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
	return ErrBoolean()
}

// GetLike calculates the index for the given pattern, and
// returns a list of all pk that match (may be empty), or an error
func (i Index) GetLike(db weave.ReadOnlyKVStore, pattern Object) ([][]byte, error) {
	index, err := i.index(pattern)
	if err != nil {
		return nil, err
	}
	return i.GetAt(db, index)
}

// GetAt returns a list of all pk at that index (may be empty), or an error
func (i Index) GetAt(db weave.ReadOnlyKVStore, index []byte) ([][]byte, error) {
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
		return ErrModifiedPK()
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
			return ErrUniqueConstraint(i.name)
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
		return ErrRemoveUnregistered()
	}
	if i.unique {
		// if something else was here, don't delete
		if !bytes.Equal(cur, pk) {
			return ErrRemoveUnregistered()
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
			return ErrUniqueConstraint(i.name)
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
