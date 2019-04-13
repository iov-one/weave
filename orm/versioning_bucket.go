package orm

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

const latestVersionIndexName = "latest"

type VersioningBucket struct {
	// todo: do not shadow bucket
	Bucket
}

func WithVersioning(b Bucket) VersioningBucket {
	return VersioningBucket{b.WithRawIndex(NewVersionIndex(b.MustBuildInternalIndexName(latestVersionIndexName)), latestVersionIndexName)}
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
