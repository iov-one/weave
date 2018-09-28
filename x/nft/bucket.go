package nft

import (
	"fmt"
	"sync"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
)

// OwnerIndexName is the index to query nft by owner
const OwnerIndexName = "owner"

type Owned interface {
	OwnerAddress() weave.Address
}

//TODO: Better name?
type BucketAccess interface {
	Get(db weave.ReadOnlyKVStore, key []byte) (orm.Object, error)
	Save(db weave.KVStore, model orm.Object) error
}

type BucketDispatcher interface {
	Register(t string, bucket BucketAccess) error
	AssertRegistered(types ...fmt.Stringer)
	Get(t string) (BucketAccess, error)
}

type bucketDispatcher struct {
	mutex     sync.RWMutex
	bucketMap map[string]BucketAccess
}

var bs BucketDispatcher

//TODO: if we ever want to support concurrency here
// then we might be better of with a separate Init() method or similar
func GetBucketDispatcher() BucketDispatcher {
	if bs == nil {
		bs = &bucketDispatcher{}
	}

	return bs
}

func (b *bucketDispatcher) Register(t string, bucket BucketAccess) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if _, ok := b.bucketMap[t]; ok {
		return ErrDuplicateEntry()
	}
	b.bucketMap[t] = bucket

	return nil
}

func (b *bucketDispatcher) AssertRegistered(types ...fmt.Stringer) {
	if len(types) != len(b.bucketMap) {
		panic("Not enough types registered")
	}

	for _, t := range types {
		if _, ok := b.bucketMap[t.String()]; !ok {
			panic(fmt.Sprintf("Missing registered type: %s", t))
		}
	}
}

func (b *bucketDispatcher) Get(t string) (BucketAccess, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if v, ok := b.bucketMap[t]; ok {
		return v, nil
	}
	return nil, ErrUnsupportedTokenType()
}

func WithOwnerIndex(bucket orm.Bucket) orm.Bucket {
	return bucket.WithIndex(OwnerIndexName, ownerIndex, false)
}

func ownerIndex(obj orm.Object) ([]byte, error) {
	if obj == nil {
		return nil, orm.ErrInvalidIndex("nil")
	}
	o, ok := obj.Value().(Owned)
	if !ok {
		return nil, orm.ErrInvalidIndex("unsupported type")
	}
	return []byte(o.OwnerAddress()), nil
}
