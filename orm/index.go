package orm

import (
	"bytes"
	"errors"

	"github.com/iov-one/weave"
)

var indPrefix = []byte("_i.")

// Indexer calculates the secondary index key for a given object
type Indexer func(Object) ([]byte, error)

// MultiKeyIndexer calculates the secondary index keys for a given object
type MultiKeyIndexer func(Object) ([][]byte, error)

// Index represents a secondary index on some data.
// It is indexed by an arbitrary key returned by Indexer.
// The value is one primary key (unique),
// Or an array of primary keys (!unique).
type Index struct {
	name   string
	id     []byte
	unique bool
	index  MultiKeyIndexer
	refKey func([]byte) []byte
}

var _ Indexed = Index{}

// NewIndex constructs an index
// Indexer calculcates the index for an object
// unique enforces a unique constraint on the index
// refKey calculates the absolute dbkey for a ref
func NewIndex(name string, indexer Indexer, unique bool,
	refKey func([]byte) []byte) Index {
	// TODO: index name must be [a-z_]
	return Index{
		name: name,
		id:   append(indPrefix, []byte(name+":")...),
		index: func(obj Object) ([][]byte, error) {
			key, err := indexer(obj)
			if err != nil {
				return nil, err
			}
			return [][]byte{key}, nil
		},
		unique: unique,
		refKey: refKey,
	}
}
func NewMulitiKeyIndex(name string, indexer MultiKeyIndexer, unique bool,
	refKey func([]byte) []byte) Index {
	// TODO: index name must be [a-z_]
	return Index{
		name:   name,
		id:     append(indPrefix, []byte(name+":")...),
		index:  indexer,
		unique: unique,
		refKey: refKey,
	}
}

// IndexKey is the full key we store in the db, including prefix
// We copy into a new array rather than use append, as we don't
// want consequetive calls to overwrite the same byte array.
func (i Index) IndexKey(key []byte) []byte {
	l := len(i.id)
	out := make([]byte, l+len(key))
	copy(out, i.id)
	copy(out[l:], key)
	return out
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
		keys, err := i.index(save)
		if err != nil {
			return err
		}
		for _, key := range keys {
			if err := i.insert(db, key, save.Key()); err != nil {
				return err
			}
		}
		return nil
	case s{false, true}:
		keys, err := i.index(prev)
		if err != nil {
			return err
		}
		for _, key := range keys {
			if err := i.remove(db, key, prev.Key()); err != nil {
				return err
			}
		}
		return nil
	case s{false, false}:
		return i.move(db, prev, save)
	}
	return ErrBoolean()
}

// GetLike calculates the index for the given pattern, and
// returns a list of all pk that match (may be empty), or an error
func (i Index) GetLike(db weave.ReadOnlyKVStore, pattern Object) ([][]byte, error) {
	indexes, err := i.index(pattern)
	if err != nil {
		return nil, err
	}
	for _, index := range indexes {
		return i.GetAt(db, index)
	}
	return [][]byte{}, nil
}

// GetAt returns a list of all pk at that index (may be empty), or an error
func (i Index) GetAt(db weave.ReadOnlyKVStore, index []byte) ([][]byte, error) {
	key := i.IndexKey(index)
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

// GetPrefix returns all references that have an index that
// begins with a given prefix
func (i Index) GetPrefix(db weave.ReadOnlyKVStore, prefix []byte) ([][]byte, error) {
	dbPrefix := i.IndexKey(prefix)
	itr := db.Iterator(prefixRange(dbPrefix))
	var data [][]byte

	for ; itr.Valid(); itr.Next() {
		if i.unique {
			data = append(data, itr.Value())
		} else {
			tmp := new(MultiRef)
			err := tmp.Unmarshal(itr.Value())
			if err != nil {
				return nil, err
			}
			data = append(data, tmp.Refs...)
		}
	}

	return data, nil
}

// Query handles queries from the QueryRouter
func (i Index) Query(db weave.ReadOnlyKVStore, mod string,
	data []byte) ([]weave.Model, error) {

	switch mod {
	case weave.KeyQueryMod:
		refs, err := i.GetAt(db, data)
		if err != nil {
			return nil, err
		}
		return i.loadRefs(db, refs), nil
	case weave.PrefixQueryMod:
		refs, err := i.GetPrefix(db, data)
		if err != nil {
			return nil, err
		}
		return i.loadRefs(db, refs), nil
	default:
		return nil, errors.New("no implemented: " + mod)
	}
}

func (i Index) loadRefs(db weave.ReadOnlyKVStore,
	refs [][]byte) []weave.Model {

	if len(refs) == 0 {
		return nil
	}
	res := make([]weave.Model, len(refs))
	for j, ref := range refs {
		key := i.refKey(ref)
		res[j] = weave.Model{
			Key:   key,
			Value: db.Get(key),
		}
	}
	return res
}

func (i Index) move(db weave.KVStore, prev Object, save Object) error {
	// if the primary key is not equal, we have a problem
	if !bytes.Equal(prev.Key(), save.Key()) {
		return ErrModifiedPK()
	}

	oldKeys, err := i.index(prev)
	if err != nil {
		return err
	}
	newKeys, err := i.index(save)
	if err != nil {
		return err
	}

	oldKeysSet := make(map[string]struct{})
	for _, v := range oldKeys {
		oldKeysSet[string(v)] = struct{}{}
	}

	newKeysSet := make(map[string]struct{})
	for _, v := range newKeys {
		newKeysSet[string(v)] = struct{}{}
	}

	// check unique constraints first
	for _, newKey := range newKeys {
		if _, exists := oldKeysSet[string(newKey)]; exists { // skip existing keys
			continue
		}
		// check unique constraint before removing
		if i.unique && len(oldKeysSet) != 0 {
			k := i.IndexKey(newKey)
			val := db.Get(k)
			if val != nil {
				return ErrUniqueConstraint(i.name)
			}
		}
	}

	// remove unused keys
	for _, oldKey := range oldKeys {
		if _, exists := newKeysSet[string(oldKey)]; !exists {
			if err = i.remove(db, oldKey, prev.Key()); err != nil {
				return err
			}
			continue
		}
	}

	// add new keys
	for _, newKey := range newKeys {
		// check unique constraint before removing
		if _, exists := oldKeysSet[string(newKey)]; !exists {
			if err = i.insert(db, newKey, prev.Key()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (i Index) remove(db weave.KVStore, index []byte, pk []byte) error {
	// don't deal with empty keys
	if len(index) == 0 {
		return nil
	}

	key := i.IndexKey(index)
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
	// don't deal with empty keys
	if len(index) == 0 {
		return nil
	}

	key := i.IndexKey(index)
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
