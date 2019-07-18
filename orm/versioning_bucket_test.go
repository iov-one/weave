package orm

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
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
	vID, err := versionedBucket.Create(db, anyValue)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	// then
	// test some iterations
	versionUpdates := 1<<8 + 1
	for i := 1; i < versionUpdates; i++ {
		anyUniquePayload := make([]byte, 32)
		rand.Read(anyUniquePayload)
		persistentValue := VersionedIDRef{ID: anyUniquePayload, Version: uint32(i)}
		vID, err = versionedBucket.Update(db, vID.ID, &persistentValue)
		if err != nil {
			t.Fatalf("unexpected error: %+v", err)
		}
		_, obj, err := versionedBucket.GetLatestVersion(db, vID.ID)
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
			expErr: errors.ErrInput,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			// when & then
			_, err := versionedBucket.Create(db, spec.src)
			if !spec.expErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}
		})
	}
}

func TestCreateWithIDWithVersioning(t *testing.T) {
	bucketImpl := NewBucket("any", NewSimpleObj(nil, &VersionedIDRef{}))
	idGenBucket := WithSeqIDGenerator(bucketImpl, "id")
	versionedBucket := WithVersioning(idGenBucket)

	specs := map[string]struct {
		src    *VersionedIDRef
		id     []byte
		expErr *errors.Error
	}{
		"Happy path": {
			src: &VersionedIDRef{ID: []byte("anyValue")},
			id:  weavetest.SequenceID(1),
		},
		"Fails with version set": {
			src:    &VersionedIDRef{ID: []byte("anyValue"), Version: 1},
			id:     weavetest.SequenceID(1),
			expErr: errors.ErrInput,
		},
		"Fails with nil id": {
			src:    &VersionedIDRef{ID: []byte("anyValue"), Version: 1},
			id:     nil,
			expErr: errors.ErrEmpty,
		},
		"Fails with empty id": {
			src:    &VersionedIDRef{ID: []byte("anyValue"), Version: 1},
			id:     make([]byte, 0),
			expErr: errors.ErrEmpty,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			// when & then
			_, err := versionedBucket.CreateWithID(db, spec.id, spec.src)
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
		init    func(*testing.T, weave.KVStore)
		srcID   []byte
		srcData versionedData
		expErr  *errors.Error
	}{
		"Happy path": {
			srcID:   weavetest.SequenceID(1),
			srcData: &VersionedIDRef{ID: []byte("otherValue"), Version: 1},
		},
		"Fails when current key ID not exists": {
			srcID:   []byte("nonExisting"),
			srcData: &VersionedIDRef{ID: []byte("anyValue"), Version: 1},
			expErr:  errors.ErrNotFound,
		},
		"Fails when version not set": {
			srcID:   weavetest.SequenceID(1),
			srcData: &VersionedIDRef{ID: []byte("anyValue"), Version: 0},
			expErr:  errors.ErrEmpty,
		},
		"Fails when current key version not exists": {
			srcID:   weavetest.SequenceID(1),
			srcData: &VersionedIDRef{ID: []byte("anyValue"), Version: 100},
			expErr:  errors.ErrNotFound,
		},
		"Fails when already deleted": {
			init: func(t *testing.T, db weave.KVStore) {
				if _, err := versionedBucket.Delete(db, weavetest.SequenceID(1)); err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
			},
			srcID:   weavetest.SequenceID(1),
			srcData: &VersionedIDRef{ID: []byte("otherValue"), Version: 1},
			expErr:  errors.ErrDeleted,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			// given
			if _, err := versionedBucket.Create(db, &VersionedIDRef{ID: []byte("anyValue")}); err != nil {
				t.Fatal(err)
			}
			if spec.init != nil {
				spec.init(t, db)
			}
			// when
			newKey, err := versionedBucket.Update(db, spec.srcID, spec.srcData)
			if !spec.expErr.Is(err) {
				t.Fatalf("expected %v but got %v", spec.expErr, err)
			}
			if spec.expErr != nil {
				return
			}
			// then
			if exp, got := spec.srcID, newKey.ID; !bytes.Equal(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
			if exp, got := uint32(2), newKey.Version; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
			// and check new one persisted
			obj, err := versionedBucket.GetVersion(db, *newKey)
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if exp, got := spec.srcData, obj.Value().(*VersionedIDRef); !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}

			// and validate old version still exists
			obj, err = versionedBucket.GetVersion(db, VersionedIDRef{ID: spec.srcID, Version: 1})
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if exp, got := []byte("anyValue"), obj.Value().(*VersionedIDRef).ID; !bytes.Equal(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}

		})
	}
}

func TestDeleteWithVersioning(t *testing.T) {
	bucketImpl := NewBucket("any", NewSimpleObj(nil, &VersionedIDRef{}))
	idGenBucket := WithSeqIDGenerator(bucketImpl, "id")
	versionedBucket := WithVersioning(idGenBucket)

	specs := map[string]struct {
		init   func(*testing.T, weave.KVStore)
		srcID  []byte
		expErr *errors.Error
	}{
		"Happy path": {
			srcID: weavetest.SequenceID(1),
		},
		"Fails with non existing id": {
			srcID:  []byte("nonExisting"),
			expErr: errors.ErrNotFound,
		},
		"Fails when deleted before": {
			init: func(t *testing.T, db weave.KVStore) {
				if _, err := versionedBucket.Delete(db, weavetest.SequenceID(1)); err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
			},
			srcID:  weavetest.SequenceID(1),
			expErr: errors.ErrDeleted,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			// given
			_, err := versionedBucket.Create(db, &VersionedIDRef{ID: []byte("anyValue")})
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if spec.init != nil {
				spec.init(t, db)
			}
			// when
			newKey, err := versionedBucket.Delete(db, spec.srcID)
			if !spec.expErr.Is(err) {
				t.Fatalf("expected %v but got %v", spec.expErr, err)
			}
			if spec.expErr != nil {
				return
			}
			// then
			if exp, got := spec.srcID, newKey.ID; !bytes.Equal(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}

			if exp, got := uint32(2), newKey.Version; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
			// and check new one persisted
			obj, err := versionedBucket.GetVersion(db, *newKey)

			if !errors.ErrDeleted.Is(err) {
				t.Fatalf("unexpected error: %+v, %#v", err, obj)
			}
			if got := obj; got != nil {
				t.Errorf("expected nil but got %v", got)
			}

		})
	}
}

func TestVersioningExists(t *testing.T) {
	bucketImpl := NewBucket("any", NewSimpleObj(nil, &VersionedIDRef{}))
	idGenBucket := WithSeqIDGenerator(bucketImpl, "id")
	versionedBucket := WithVersioning(idGenBucket)

	specs := map[string]struct {
		init      func(*testing.T, weave.KVStore)
		srcVID    VersionedIDRef
		expErr    *errors.Error
		expResult bool
	}{
		"True when exists": {
			srcVID:    VersionedIDRef{ID: weavetest.SequenceID(1), Version: 1},
			expResult: true,
		},
		"False with non existing id": {
			srcVID:    VersionedIDRef{ID: []byte("nonExisting")},
			expResult: false,
		},
		"False with non existing Version": {
			srcVID:    VersionedIDRef{ID: []byte("nonExisting"), Version: 111},
			expResult: false,
		},
		"Error when deleted": {
			init: func(t *testing.T, db weave.KVStore) {
				if _, err := versionedBucket.Delete(db, weavetest.SequenceID(1)); err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
			},
			srcVID: VersionedIDRef{ID: weavetest.SequenceID(1), Version: 2},
			expErr: errors.ErrDeleted,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			// given
			_, err := versionedBucket.Create(db, &VersionedIDRef{ID: []byte("anyValue")})
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if spec.init != nil {
				spec.init(t, db)
			}
			// when & then
			result, err := versionedBucket.Exists(db, spec.srcVID)
			if !spec.expErr.Is(err) {
				t.Fatalf("expected %v but got %v", spec.expErr, err)
			}
			if spec.expErr != nil {
				return
			}
			if exp, got := spec.expResult, result; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}
}
