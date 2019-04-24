package utils

import (
	"context"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestSavepoint(t *testing.T) {
	// always write ok, ov before calling functions
	ok, ov := []byte("demo"), []byte("data")
	// some key, value to try to write
	nk, nv := []byte{1, 2, 3}, []byte{4, 5, 6}
	// a default error if desired
	derr := fmt.Errorf("something went wrong")

	cases := [...]struct {
		save    weave.Decorator // decorator at savepoint
		handler weave.Handler
		check   bool // whether to call Check or Deliver
		isError bool // true iff we expect errors

		written [][]byte // keys to find
		missing [][]byte // keys not to find
	}{
		// savepoint disactivated, returns error, both written
		0: {
			NewSavepoint(),
			&writeHandler{key: nk, value: nv, err: derr},
			true,
			true,
			[][]byte{ok, nk},
			nil,
		},
		// savepoint activated, returns error, one written
		1: {
			NewSavepoint().OnCheck(),
			&writeHandler{key: nk, value: nv, err: derr},
			true,
			true,
			[][]byte{ok},
			[][]byte{nk},
		},
		// savepoint activated for deliver, returns error, one written
		2: {
			NewSavepoint().OnDeliver(),
			&writeHandler{key: nk, value: nv, err: derr},
			false,
			true,
			[][]byte{ok},
			[][]byte{nk},
		},
		// double-activation maintains both behaviors
		3: {
			NewSavepoint().OnDeliver().OnCheck(),
			&writeHandler{key: nk, value: nv, err: derr},
			false,
			true,
			[][]byte{ok},
			[][]byte{nk},
		},
		// savepoint check doesn't affect deliver
		4: {
			NewSavepoint().OnCheck(),
			&writeHandler{key: nk, value: nv, err: derr},
			false,
			true,
			[][]byte{ok, nk},
			nil,
		},
		// don't rollback when success returned
		5: {
			NewSavepoint().OnCheck().OnDeliver(),
			&writeHandler{key: nk, value: nv, err: nil},
			false,
			false,
			[][]byte{ok, nk},
			nil,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			ctx := context.Background()
			kv := store.MemStore()
			kv.Set(ok, ov)

			var err error
			if tc.check {
				_, err = tc.save.Check(ctx, kv, nil, tc.handler)
			} else {
				_, err = tc.save.Deliver(ctx, kv, nil, tc.handler)
			}

			if tc.isError {
				if err == nil {
					t.Fatalf("Expected error")
				}
			} else {
				assert.Nil(t, err)
			}

			for _, k := range tc.written {
				has, err := kv.Has(k)
				assert.Nil(t, err)
				if !has {
					t.Errorf("Didn't write key: %X", k)
				}
			}
			for _, k := range tc.missing {
				has, err := kv.Has(k)
				assert.Nil(t, err)
				if has {
					t.Errorf("Wrote missing value: %X", k)
				}
			}
		})
	}
}

// writeHandler writes the key, value pair and returns the error (may be nil)
type writeHandler struct {
	key   []byte
	value []byte
	err   error
}

func (h writeHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	store.Set(h.key, h.value)
	return &weave.CheckResult{}, h.err
}

func (h writeHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	store.Set(h.key, h.value)
	return &weave.DeliverResult{}, h.err
}
