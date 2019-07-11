package utils

import (
	"context"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestSavepoint(t *testing.T) {
	// always write ok, ov before calling functions
	ok, ov := []byte("demo"), []byte("data")
	// some key, value to try to write
	nk, nv := []byte{1, 2, 3}, []byte{4, 5, 6}
	// a default error if desired
	derr := fmt.Errorf("something went wrong")

	cases := map[string]struct {
		save    weave.Decorator // decorator at savepoint
		handler weave.Handler
		check   bool // whether to call Check or Deliver
		isError bool // true iff we expect errors

		written [][]byte // keys to find
		missing [][]byte // keys not to find
	}{
		"savepoint dis-activated, returns error, both written": {
			NewSavepoint(),
			&writeHandler{key: nk, value: nv, err: derr},
			true,
			true,
			[][]byte{ok, nk},
			nil,
		},
		"savepoint activated, returns error, one written": {
			NewSavepoint().OnCheck(),
			&writeHandler{key: nk, value: nv, err: derr},
			true,
			true,
			[][]byte{ok},
			[][]byte{nk},
		},
		"savepoint activated for deliver, returns error, one written": {
			NewSavepoint().OnDeliver(),
			&writeHandler{key: nk, value: nv, err: derr},
			false,
			true,
			[][]byte{ok},
			[][]byte{nk},
		},
		"double-activation maintains both behaviors": {
			NewSavepoint().OnDeliver().OnCheck(),
			&writeHandler{key: nk, value: nv, err: derr},
			false,
			true,
			[][]byte{ok},
			[][]byte{nk},
		},
		"savepoint check doesn't affect deliver": {
			NewSavepoint().OnCheck(),
			&writeHandler{key: nk, value: nv, err: derr},
			false,
			true,
			[][]byte{ok, nk},
			nil,
		},
		"don't rollback when success returned": {
			NewSavepoint().OnCheck().OnDeliver(),
			&writeHandler{key: nk, value: nv, err: nil},
			false,
			false,
			[][]byte{ok, nk},
			nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
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

func TestCacheWriteFail(t *testing.T) {
	handler := &weavetest.Handler{
		CheckResult:   weave.CheckResult{Log: "all good"},
		DeliverResult: weave.DeliverResult{Log: "all good"},
	}
	tx := &weavetest.Tx{}

	// Register an error that is guaranteed to be unique.
	myerr := errors.Register(921928, "my error")

	db := &cacheableStoreMock{
		CacheableKVStore: store.MemStore(),
		err:              myerr,
	}

	decorator := NewSavepoint().OnCheck().OnDeliver()

	if _, err := decorator.Check(context.TODO(), db, tx, handler); !myerr.Is(err) {
		t.Fatalf("unexpected check result error: %+v", err)
	}

	if _, err := decorator.Deliver(context.TODO(), db, tx, handler); !myerr.Is(err) {
		t.Fatalf("unexpected deliver result error: %+v", err)
	}
}

// cacheableStoreMock is a mock of a store and a cache wrap. Use it to pass
// through all operation to wrapped CacheableKVStore. Write call returns
// defined error.
type cacheableStoreMock struct {
	weave.CacheableKVStore
	err error
}

// CachceWrap overwrites wrapped store method in order to return
// self-reference. cacheableStoreMock implements KVCacheWrap interface as well.
func (s *cacheableStoreMock) CacheWrap() weave.KVCacheWrap {
	return s
}

// Write implements KVCacheWrap interface.
func (c *cacheableStoreMock) Write() error {
	return c.err
}

// Discard implements KVCacheWrap interface.
func (cacheableStoreMock) Discard() {}
