package orm

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestIDGenBucket(t *testing.T) {
	bucketImpl := NewBucket("any", NewSimpleObj(nil, &Counter{}))

	specs := map[string]struct {
		bucket IDGenBucket
		expID  []byte
		expErr *errors.Error
	}{
		"Calls NextVal on Create": {
			bucket: WithIDGenerator(bucketImpl,
				IDGeneratorFunc(func(db weave.KVStore, obj CloneableData) ([]byte, error) {
					return []byte("myKey"), nil
				})),
			expID: []byte("myKey"),
		},
		"Passes error from NextVal on Create": {
			bucket: WithIDGenerator(bucketImpl,
				IDGeneratorFunc(func(db weave.KVStore, obj CloneableData) ([]byte, error) {
					return nil, errors.ErrHuman
				})),
			expErr: errors.ErrHuman,
		},
		"Returns number with seqIDGenerator": {
			bucket: WithSeqIDGenerator(bucketImpl, "id"),
			expID:  weavetest.SequenceID(1),
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			// when
			obj, err := spec.bucket.Create(db, &Counter{})
			// then
			if !spec.expErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}
			if spec.expErr != nil {
				return
			}
			if exp, got := spec.expID, obj.Key(); !bytes.Equal(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
			loadedObj, err := spec.bucket.Get(db, obj.Key())
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if exp, got := obj, loadedObj; !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}

}
