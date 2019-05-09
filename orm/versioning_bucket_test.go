package orm

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestGetLatestVersion(t *testing.T) {
	bucketImpl := NewBucket("any", NewSimpleObj(nil, &VersionedIDRef{}))
	idGenBucket := WithSeqIDGenerator(bucketImpl, "id")
	versionedBucket := WithVersioning(idGenBucket)
	db := store.MemStore()
	// when
	anyValue := &VersionedIDRef{ID: []byte("anyValue")}
	obj, err := versionedBucket.Create(db, anyValue)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	// then
	var vID VersionedIDRef
	if err := vID.Unmarshal(obj.Key()); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	// test some iterations
	versionUpdates := 1<<8 + 1
	for i := 1; i < versionUpdates; i++ {
		anyUniquePayload := make([]byte, 32)
		rand.Read(anyUniquePayload)
		persistentValue := VersionedIDRef{ID: anyUniquePayload, Version: uint32(i)}
		vID, err = versionedBucket.Update(db, vID, &persistentValue)
		if err != nil {
			t.Fatalf("unexpected error: %+v", err)
		}
		obj, err := versionedBucket.GetLatestVersion(db, vID.ID)
		if err != nil {
			t.Fatalf("unexpected error: %+v", err)
		}
		if exp, got := &persistentValue, obj.Value().(*VersionedIDRef); !reflect.DeepEqual(exp, got) {
			t.Errorf("expected %v but got %v", exp, got)
		}
	}
}

func TestCreateWithVersioning(t *testing.T) {
	bucketImpl := NewBucket("any", NewSimpleObj(nil, &VersionedIDRef{}))
	idGenBucket := WithSeqIDGenerator(bucketImpl, "id")
	versionedBucket := WithVersioning(idGenBucket)

	specs := map[string]struct {
		src    *VersionedIDRef
		expErr *errors.Error
	}{
		"Happy path": {
			src: &VersionedIDRef{ID: []byte("anyValue")},
		},
		"Fails with version set": {
			src:    &VersionedIDRef{ID: []byte("anyValue"), Version: 1},
			expErr: errors.ErrInvalidInput,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			_, err := versionedBucket.Create(db, spec.src)
			if !spec.expErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}
		})
	}

}
func TestUpdateWithVersioning(t *testing.T) {
	bucketImpl := NewBucket("any", NewSimpleObj(nil, &VersionedIDRef{}))
	idGenBucket := WithSeqIDGenerator(bucketImpl, "id")
	versionedBucket := WithVersioning(idGenBucket)

	specs := map[string]struct {
		srcCurrentVersionKey VersionedIDRef
		srcData              versionedData
		expErr               *errors.Error
	}{
		"Happy path": {
			srcCurrentVersionKey: VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			srcData:              &VersionedIDRef{ID: []byte("otherValue"), Version: 1},
		},
		"Fails with version mismatch": {
			srcCurrentVersionKey: VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			srcData:              &VersionedIDRef{ID: []byte("anyValue"), Version: 10},
			expErr:               errors.ErrInvalidState,
		},
		"Fails when current key ID not exists": {
			srcCurrentVersionKey: VersionedIDRef{ID: []byte("nonExisting"), Version: 1},
			srcData:              &VersionedIDRef{ID: []byte("anyValue"), Version: 1},
			expErr:               errors.ErrNotFound,
		},
		"Fails when current key version not exists": {
			srcCurrentVersionKey: VersionedIDRef{ID: weavetest.SequenceID(1), Version: 100},
			srcData:              &VersionedIDRef{ID: []byte("anyValue"), Version: 100},
			expErr:               errors.ErrNotFound,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			if _, err := versionedBucket.Create(db, &VersionedIDRef{ID: []byte("anyValue")}); err != nil {
				t.Fatal(err)
			}
			// when
			newKey, err := versionedBucket.Update(db, spec.srcCurrentVersionKey, spec.srcData)
			if !spec.expErr.Is(err) {
				t.Errorf("expected %v but got %v", spec.expErr, err)
			}
			if spec.expErr != nil {
				return
			}
			// then
			if exp, got := spec.srcCurrentVersionKey.ID, newKey.ID; !bytes.Equal(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
			if exp, got := uint32(2), newKey.Version; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
			// and check new one persisted
			key, _ := newKey.Marshal()
			obj, err := versionedBucket.Get(db, key)
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if exp, got := spec.srcData, obj.Value().(*VersionedIDRef); !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}

			// and validate old version still exists
			key, _ = spec.srcCurrentVersionKey.Marshal()
			obj, err = versionedBucket.Get(db, key)
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if exp, got := []byte("anyValue"), obj.Value().(*VersionedIDRef).ID; !bytes.Equal(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}

		})
	}

}
