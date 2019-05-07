package orm

import (
	"fmt"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

type VersioningBucket struct {
	IDGenBucket
}

func WithVersioning(b IDGenBucket) VersioningBucket {
	return VersioningBucket{b}
}

func (b VersioningBucket) GetLatestVersion(db weave.ReadOnlyKVStore, id []byte) (Object, error) {
	idWithoutVersion := &VersionedIDRef{ID: id}
	prefix, err := idWithoutVersion.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal versioned ID ref")
	}
	dbKeyLength := len(b.DBKey(prefix)) - len(prefix)
	fmt.Printf("searching with: #%v\n", prefix)
	matches, err := b.Query(db, weave.PrefixQueryMod, prefix)
	if err != nil {
		return nil, errors.Wrap(err, "prefix query")
	}
	// find highest version for that ID
	var highestVersion VersionedIDRef
	var found weave.Model
	for _, m := range matches {
		var vID VersionedIDRef
		idData := m.Key[dbKeyLength:]
		fmt.Printf("found : #%v\n", idData)

		if err := vID.Unmarshal(idData); err != nil {
			return nil, errors.Wrap(err, "wrong key type")
		}
		if vID.Version > highestVersion.Version {
			highestVersion = vID
			found = m
		}
	}
	if len(found.Value) == 0 {
		return nil, errors.Wrap(errors.ErrNotFound, "unknown id")
	}
	return b.Parse(found.Key, found.Value)
}

func (b VersioningBucket) GetVersion(db weave.ReadOnlyKVStore, ref VersionedIDRef) (Object, error) {
	key, err := ref.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal version id key")
	}
	return b.Get(db, key)
}

type versioned interface {
	GetVersion() uint32
}
type versionedData interface {
	CloneableData
	versioned
	SetVersion(uint32)
}

// Create assigns an ID and initial version number to given object instance and returns it as an persisted orm
// Object.
func (b VersioningBucket) Create(db weave.KVStore, data versionedData) (Object, error) {
	if data.GetVersion() != 0 {
		return nil, errors.Wrap(errors.ErrInvalidInput, "version is set on create")
	}
	data.SetVersion(1)
	newID, err := b.idGen.NextVal(db, data)
	if err != nil {
		return nil, err
	}
	idRef := &VersionedIDRef{ID: newID, Version: data.GetVersion()}
	key, err := idRef.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshall versioned id ref")
	}
	obj := NewSimpleObj(key, data)
	return obj, b.IDGenBucket.Save(db, obj)
}

// Deprecated: Save will always return an error. Use Create or Update instead.
func (b VersioningBucket) Save(db weave.KVStore, model Object) error {
	return errors.Wrap(errors.ErrHuman, "raw save not supported")
}

// Update persists the given data object with a new derived version key in the storage.
func (b VersioningBucket) Update(db weave.KVStore, oldKey VersionedIDRef, data versionedData) (VersionedIDRef, error) {
	if data.GetVersion() != oldKey.Version {
		return oldKey, errors.Wrap(errors.ErrInvalidState, "versions not matching")
	}
	newKey, err := oldKey.NextVersion()
	if err != nil {
		return oldKey, err
	}
	data.SetVersion(newKey.Version)
	key, err := newKey.Marshal()
	if err != nil {
		return oldKey, errors.Wrap(err, "failed to marshal key")
	}
	return newKey, b.Bucket.Save(db, NewSimpleObj(key, data))
}
