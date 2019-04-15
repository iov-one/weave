package orm

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const latestVersionIndexName = "latest"

type VersioningBucket struct {
	IDGenBucket
}

func WithVersioning(b IDGenBucket) VersioningBucket {
	indexedBucket := b.withRawIndex(
		NewVersionIndex(b.MustBuildInternalIndexName(latestVersionIndexName), b),
		latestVersionIndexName,
	)
	return VersioningBucket{WithIDGenerator(indexedBucket, b.idGen)}
}

func (b VersioningBucket) GetLatestVersion(db weave.ReadOnlyKVStore, id []byte) (Object, error) {
	objs, err := b.Bucket.GetIndexed(db, latestVersionIndexName, id)
	switch {
	case err != nil:
		return nil, errors.Wrapf(err, "failed to load object with index: %q", latestVersionIndexName)
	case len(objs) == 0:
		return nil, errors.Wrap(errors.ErrNotFound, "unknown id")
	case len(objs) == 1:
		return objs[0], nil
	}
	return nil, errors.Wrap(errors.ErrHuman, "multiple values indexed")
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
		return nil, errors.Wrap(errors.ErrInvalidInput, "version is set by create")
	}
	data.SetVersion(1)
	idRef := &VersionedIDRef{ID: b.idGen.NextVal(db, data), Version: data.GetVersion()}
	key, err := idRef.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshall versioned id ref")
	}
	obj := NewSimpleObj(key, data)
	return obj, b.IDGenBucket.Save(db, obj)
}

func (b VersioningBucket) Save(db weave.KVStore, model Object) error {
	return errors.Wrap(errors.ErrHuman, "raw save not supported")
}

// Update persists the given data object with a new derived version key in the storage.
func (b VersioningBucket) Update(db weave.KVStore, oldKey VersionedIDRef, data versionedData) (VersionedIDRef, error) {
	if data.GetVersion() != oldKey.Version {
		return oldKey, errors.Wrap(errors.ErrInvalidState, "versions not matching")
	}
	newKey := oldKey.NextVersion()
	data.SetVersion(newKey.Version)
	key, err := newKey.Marshal()
	if err != nil {
		return oldKey, errors.Wrap(err, "failed to marshal key")
	}
	return newKey, b.Bucket.Save(db, NewSimpleObj(key, data))
}
