package orm

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestGetLatestVersion(t *testing.T) {
	bucketImpl := NewBucket("any", NewSimpleObj(nil, &VersionedIDRef{}))
	idGenBucket := WithSeqIDGenerator(bucketImpl, "id")
	versionedBucket := WithVersioning(idGenBucket)
	db := store.MemStore()
	// when
	obj, err := versionedBucket.Create(db, &VersionedIDRef{ID: []byte("anyValue")})
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	var vID VersionedIDRef
	if err := vID.Unmarshal(obj.Key()); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	// test some iterations
	for i := 0; i < 100; i++ {
		vID, err = versionedBucket.Update(db, vID, &vID)
		if err != nil {
			t.Fatalf("unexpected error: %+v", err)
		}
		obj, err := versionedBucket.GetLatestVersion(db, vID.ID)
		if err != nil {
			t.Fatalf("unexpected error: %+v", err)
		}
		if exp, got := &vID, obj.Value().(*VersionedIDRef); !reflect.DeepEqual(exp, got) {
			t.Errorf("expected %v but got %v", exp, got)
		}
	}
	// and old versions still exist
	for i := 1; i < 102; i++ { // 1 (initial) + 100 (updates)
		vID := VersionedIDRef{ID: weavetest.SequenceID(1), Version: uint32(i)}
		data, _ := vID.Marshal()
		obj, err = versionedBucket.Get(db, data)
		if err != nil {
			t.Fatalf("unexpected error: %+v", err)
		}
		if obj == nil || obj.Value() == nil {
			t.Fatalf("expected version exists: %d ", i)
		}
	}
}
