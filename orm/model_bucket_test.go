package orm

import (
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
)

func TestModelBucket(t *testing.T) {
	db := store.MemStore()

	obj := NewSimpleObj(nil, &Counter{})
	objBucket := NewBucket("cnts", obj)

	b := NewModelBucket(objBucket)

	if err := b.Put(db, []byte("c1"), &Counter{Count: 1}); err != nil {
		t.Fatalf("cannot save counter instance: %s", err)
	}

	var c1 Counter
	if err := b.One(db, []byte("c1"), &c1); err != nil {
		t.Fatalf("cannot get c1 counter: %s", err)
	}
	if c1.Count != 1 {
		t.Fatalf("unexpected counter state: %d", c1)
	}

	if err := b.Delete(db, []byte("c1")); err != nil {
		t.Fatalf("cannot delete c1 counter: %s", err)
	}
	if err := b.Delete(db, []byte("unknown")); !errors.ErrNotFound.Is(err) {
		t.Fatalf("unexpected error when deleting unexisting instance: %s", err)
	}
	if err := b.One(db, []byte("c1"), &c1); !errors.ErrNotFound.Is(err) {
		t.Fatalf("unexpected error for an unknown model get: %s", err)
	}
}
