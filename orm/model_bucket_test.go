package orm

import (
	"strconv"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
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

func TestModelBucketMany(t *testing.T) {
	cases := map[string]struct {
		IndexName string
		QueryKey  string
		Dest      []Model
		WantErr   *errors.Error
		WantRes   []Model
	}{
		"find none": {
			IndexName: "value",
			QueryKey:  "124089710947120",
			Dest:      nil,
			WantErr:   nil,
			WantRes:   nil,
		},
		"find one": {
			IndexName: "value",
			QueryKey:  "1111",
			Dest:      nil,
			WantErr:   nil,
			WantRes: []Model{
				&Counter{Count: 1111},
			},
		},
		"find two with nil destination": {
			IndexName: "value",
			QueryKey:  "4444",
			Dest:      []Model{},
			WantErr:   nil,
			WantRes: []Model{
				&Counter{Count: 4444},
				&Counter{Count: 4444},
			},
		},
		"find two with allocated destination": {
			IndexName: "value",
			QueryKey:  "4444",
			Dest:      make([]Model, 0, 10),
			WantErr:   nil,
			WantRes: []Model{
				&Counter{Count: 4444},
				&Counter{Count: 4444},
			},
		},
		"find two with non empty destination": {
			IndexName: "value",
			QueryKey:  "4444",
			Dest: []Model{
				&Counter{Count: 007},
			},
			WantErr: nil,
			WantRes: []Model{
				// Destination is always appended to.
				&Counter{Count: 007},

				&Counter{Count: 4444},
				&Counter{Count: 4444},
			},
		},
		"non existing index name": {
			IndexName: "xyz",
			WantErr:   ErrInvalidIndex,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			indexByValue := func(obj Object) ([]byte, error) {
				c, ok := obj.Value().(*Counter)
				if !ok {
					return nil, errors.Wrapf(errors.ErrType, "%T", obj.Value())
				}
				raw := strconv.FormatInt(c.Count, 10)
				return []byte(raw), nil
			}
			objBucket := NewBucket("cnts", NewSimpleObj(nil, &Counter{})).
				WithIndex("value", indexByValue, false)
			b := NewModelBucket(objBucket)

			if err := b.Put(db, []byte("c1"), &Counter{Count: 4444}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}
			if err := b.Put(db, []byte("c2"), &Counter{Count: 4444}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}
			if err := b.Put(db, []byte("c3"), &Counter{Count: 1111}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}
			if err := b.Put(db, []byte("c4"), &Counter{Count: 99999}); err != nil {
				t.Fatalf("cannot save counter instance: %s", err)
			}

			err := b.Many(db, tc.IndexName, []byte(tc.QueryKey), &tc.Dest)
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %s", err)
			}
			assert.Equal(t, tc.WantRes, tc.Dest)
		})
	}
}
