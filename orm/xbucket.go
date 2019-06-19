package orm

import (
	"fmt"

	"github.com/iov-one/weave"
)

type BucketBuilderOpt interface {
	Apply(*BucketBuilder) *BucketBuilder
}

type BucketBuilderOptFunc func(*BucketBuilder) *BucketBuilder

func (f BucketBuilderOptFunc) Apply(bb *BucketBuilder) *BucketBuilder {
	return f(bb)
}

type BucketBuilder struct {
	b   bucket
	pkg string
}

func NewBucketBuilder(name string, model Cloneable, opts ...BucketBuilderOpt) *BucketBuilder {
	if !isBucketName(name) {
		panic(fmt.Sprintf("Illegal bucket: %s", name))
	}

	b := bucket{
		name:   name,
		prefix: append([]byte(name), ':'),
		proto:  model,
	}
	bb := &BucketBuilder{
		b: b,
	}
	for _, v := range opts {
		bb = v.Apply(bb)
	}
	return bb
}

func (b *BucketBuilder) WithIndex(name string, indexer Indexer, unique bool) *BucketBuilder {
	return b.WithMultiKeyIndex(name, asMultiKeyIndexer(indexer), unique)
}

func (b *BucketBuilder) WithMultiKeyIndex(name string, indexer MultiKeyIndexer, unique bool) *BucketBuilder {
	b.b = b.b.withMultiKeyIndex(name, indexer, unique)
	return b
}

func (b BucketBuilder) Build() BaseBucket {
	return b.b
}

type QueryableBucket interface {
	Register(name string, r weave.QueryRouter)
	weave.QueryHandler
}

type BaseBucket interface {
	// core functionality
	Get(db weave.ReadOnlyKVStore, key []byte) (Object, error)
	Save(db weave.KVStore, model Object) error
	Delete(db weave.KVStore, key []byte) error
	GetIndexed(db weave.ReadOnlyKVStore, name string, key []byte) ([]Object, error)

	// extension points
	Name() string
	dbKey(key []byte) []byte
	parse(key, value []byte) (Object, error)
	EmbeddedBucket
	VisitableBucket

	// migration phase ??
	QueryableBucket
}

type XMigrationBucket interface {
	BaseBucket
}

type XLastModifiedBucket interface {
	BaseBucket
}

type XIDGenBucket interface {
	Get(db weave.ReadOnlyKVStore, key []byte) (Object, error)
	Delete(db weave.KVStore, key []byte) error
	GetIndexed(db weave.ReadOnlyKVStore, name string, key []byte) ([]Object, error)
	Create(db weave.KVStore, data CloneableData) (Object, error)
	Update(db weave.KVStore, id []byte, data CloneableData) (Object, error)

	// extension points
	nextVal(db weave.KVStore, obj CloneableData) ([]byte, error)
	VisitableBucket
	EmbeddedBucket
}

type XVersioningBucket interface {
	Create(db weave.KVStore, data versionedData) (*VersionedIDRef, error)
	Update(db weave.KVStore, id []byte, data versionedData) (*VersionedIDRef, error)
	Delete(db weave.KVStore, id []byte) (*VersionedIDRef, error)
	Get(db weave.ReadOnlyKVStore, key []byte) (Object, error)
	GetVersion(db weave.ReadOnlyKVStore, ref VersionedIDRef) (Object, error)
	GetLatestVersion(db weave.ReadOnlyKVStore, id []byte) (*VersionedIDRef, Object, error)
	// Can be renamed to Has ?
	Exists(db weave.KVStore, idRef VersionedIDRef) (bool, error)
	// extension points
	VisitableBucket
	EmbeddedBucket
}

var (
	_ XIDGenBucket      = &IDGenBucket{}
	_ XVersioningBucket = VersioningBucket{}
	_ XModelBucket      = &modelBucket{}
	_ BaseBucket        = &bucket{}
)

type XModelBucket interface {
	One(db weave.ReadOnlyKVStore, key []byte, dest Model) error
	ByIndex(db weave.ReadOnlyKVStore, indexName string, key []byte, dest ModelSlicePtr) (keys [][]byte, err error)
	Put(db weave.KVStore, key []byte, m Model) ([]byte, error)
	Delete(db weave.KVStore, key []byte) error
	Has(db weave.KVStore, key []byte) error
	// extension points
	VisitableBucket
	EmbeddedBucket
}
