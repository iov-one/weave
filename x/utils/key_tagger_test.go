package utils

import (
	"context"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/tendermint/tendermint/libs/common"
)

func TestKeyTagger(t *testing.T) {
	// always write ok, ov before calling functions
	ok, ov := []byte("foo:demo"), []byte("data")
	// some key, value to try to write
	nk, nv := []byte{1, 0xab, 3}, []byte{4, 5, 6}
	// a default error if desired
	derr := fmt.Errorf("something went wrong")

	otag, oval := []byte("666F6F3A64656D6F"), []byte("s") // "foo:demo" as upper-case hex
	ntag, nval := []byte("01AB03"), []byte("s")

	cases := map[string]struct {
		handler weave.Handler
		isError bool // true iff we expect errors
		tags    []common.KVPair
		k, v    []byte
	}{
		"return error doesn't add tags": {
			&writeHandler{key: nk, value: nv, err: derr},
			true,
			nil,
			// note that these were writen as we had no SavePoint
			nk,
			nv,
		},
		"with success records tags": {
			&writeHandler{key: nk, value: nv, err: nil},
			false,
			[]common.KVPair{{Key: ntag, Value: nval}},
			nk,
			nv,
		},
		"write multiple values (sorted order)": {
			weavetest.Decorate(
				&writeHandler{key: nk, value: nv, err: nil},
				&writeDecorator{key: ok, value: ov, after: true}),
			false,
			[]common.KVPair{{Key: ntag, Value: nval}, {Key: otag, Value: oval}},
			nk,
			nv,
		},
		"savepoint must revert any writes": {
			weavetest.Decorate(
				&writeHandler{key: nk, value: nv, err: derr},
				NewSavepoint().OnDeliver()),
			true,
			nil,
			nk,
			nil,
		},
		"savepoint keeps writes on success": {
			weavetest.Decorate(
				&writeHandler{key: nk, value: nv, err: nil},
				NewSavepoint().OnDeliver()),
			false,
			[]common.KVPair{{Key: ntag, Value: nval}},
			nk,
			nv,
		},
		"combine with other tags from the Handler": {
			weavetest.Decorate(
				newTagHandler(nk, nv, nil),
				&writeDecorator{key: ok, value: ov, after: false}),
			false,
			// note that the nk, nv set explicitly are not modified
			[]common.KVPair{{Key: nk, Value: nv}, {Key: otag, Value: oval}},
			nk,
			nil,
		},
		"on error don't add tags, but leave original ones": {
			weavetest.Decorate(
				newTagHandler(nk, nv, derr),
				&writeDecorator{key: ok, value: ov, after: false}),
			true,
			[]common.KVPair{{Key: nk, Value: nv}},
			nk,
			nil,
		},
		// TODO: also check delete
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			db := store.MemStore()
			tagger := NewKeyTagger()

			// try check - no op
			check := db.CacheWrap()
			_, err := tagger.Check(ctx, check, nil, tc.handler)
			if tc.isError {
				if err == nil {
					t.Fatalf("Expected error")
				}
			} else {
				assert.Nil(t, err)
			}

			// try deliver - records tags and sets values on success
			res, err := tagger.Deliver(ctx, db, nil, tc.handler)
			if tc.isError {
				if err == nil {
					t.Fatalf("Expected error")
				}
			} else {
				assert.Nil(t, err)
				// tags are set properly
				assert.Equal(t, tc.tags, res.Tags)
			}

			// optionally check if data was writen to underlying db
			if tc.k != nil {
				v, err := db.Get(tc.k)
				assert.Nil(t, err)
				assert.Equal(t, tc.v, v)
			}
		})
	}
}

// writeDecorator writes the key, value pair.
// either before or after calling the handlers
type writeDecorator struct {
	key   []byte
	value []byte
	after bool
}

var _ weave.Decorator = writeDecorator{}

func (d writeDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	if !d.after {
		store.Set(d.key, d.value)
	}
	res, err := next.Check(ctx, store, tx)
	if d.after && err == nil {
		store.Set(d.key, d.value)
	}
	return res, err
}

func (d writeDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	if !d.after {
		store.Set(d.key, d.value)
	}
	res, err := next.Deliver(ctx, store, tx)
	if d.after && err == nil {
		store.Set(d.key, d.value)
	}
	return res, err
}

func newTagHandler(key, value []byte, err error) weave.Handler {
	return &weavetest.Handler{
		CheckErr:   err,
		DeliverErr: err,
		DeliverResult: weave.DeliverResult{
			Tags: []common.KVPair{
				{Key: key, Value: value},
			},
		},
	}
}
