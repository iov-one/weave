package orm

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

type VersionIndex struct {
	idx Index
}

func NewVersionIndex(name string) *VersionIndex {
	return &VersionIndex{
		idx: NewIndex(name, nil, true),
	}
}
func (i VersionIndex) GetAt(db weave.ReadOnlyKVStore, index []byte) ([][]byte, error) {
	return i.idx.GetAt(db, index)
}

func (i VersionIndex) GetPrefix(db weave.ReadOnlyKVStore, prefix []byte) ([][]byte, error) {
	return i.idx.GetPrefix(db, prefix)
}

func (i VersionIndex) GetLike(db weave.ReadOnlyKVStore, pattern Object) ([][]byte, error) {
	return i.idx.GetLike(db, pattern)
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
func (i VersionIndex) Update(db weave.KVStore, prev Object, save Object) error {
	fmt.Printf("UPDATING: prev: %#v to %#v\n", prev, save)
	type s struct{ a, b bool }
	sw := s{prev == nil, save == nil}
	switch sw {
	case s{true, true}:
		return errors.Wrap(errors.ErrHuman, "update requires at least one non-nil object")
	case s{true, false}:
		id, err := indexID(save)
		if err != nil {
			return err
		}
		version, err := indexVersion(save)
		if err != nil {
			return err
		}
		fmt.Printf("+++ INSERT: %q: %d", string(id), version)
		return i.idx.insert(db, id, save.Key())
	case s{false, true}:
		fmt.Println("+++ DELETE")

		id, err := indexID(prev)
		if err != nil {
			return err
		}
		// TODO: revisit this: store empty ref key to mark deleted or drop value but that is not the same as never existed before
		return i.move(db, id, prev.Key(), []byte{})
	case s{false, false}:
		fmt.Println("+++ UPDATE")
		id1, err := indexID(prev)
		if err != nil {
			return err
		}
		switch id2, err := indexID(prev); {
		case err != nil:
			return err
		case !bytes.Equal(id1, id2):
			return errors.Wrap(errors.ErrHuman, "not equal ID")
		}
		v1, err := indexVersion(prev)
		if err != nil {
			return err
		}
		switch v2, err := indexVersion(prev); {
		case err != nil:
		case v1 <= v2:
			return errors.Wrap(errors.ErrHuman, "version not greater than previous")
		}
		return i.move(db, id1, prev.Key(), save.Key())
	}
	return errors.Wrap(errors.ErrHuman, "you have violated the rules of boolean logic")
}

func (i VersionIndex) move(db weave.KVStore, id []byte, prevKey []byte, newKey []byte) error {
	// TODO: think about unique constraint!
	if err := i.idx.remove(db, id, prevKey); err != nil {
		return err
	}
	return i.idx.insert(db, id, newKey)
}

func indexVersion(obj Object) (uint32, error) {
	if obj == nil {
		return 0, errors.Wrap(errors.ErrHuman, "cannot take index of nil")
	}
	type versioner interface {
		GetVersion() uint32
	}
	v, ok := obj.Value().(versioner)
	if !ok {
		return 0, errors.Wrap(errors.ErrHuman, "Can only take index of versioned objects")
	}
	//result := make([]byte, 4)
	//binary.BigEndian.PutUint32(result, v.GetVersion())
	return v.GetVersion(), nil
}

func indexID(obj Object) ([]byte, error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "cannot take index of nil")
	}
	type ider interface {
		GetID() []byte
	}
	v, ok := obj.Value().(ider)
	if !ok {
		return nil, errors.Wrap(errors.ErrHuman, "Can only take index of obj with support GetID()")
	}
	return v.GetID(), nil
}

// VersionedKey builds a combined key containing version and id.
func VersionedKey(id []byte, version uint32) []byte {
	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, version)
	return append(result, id...)
}
