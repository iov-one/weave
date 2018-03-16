package utils

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/confio/weave"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
)

func TestSavepoint(t *testing.T) {
	var help x.TestHelpers

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

		written  [][]byte // keys to find
		missing [][]byte // keys not to find
	}{
		// savepoint disactivated, returns error, both written
		0: {
			NewSavepoint(),
			help.WriteHandler(nk, nv, derr),
			true,
			true,
			[][]byte{ok, nk},
			nil,
		},
		// savepoint activated, returns error, one written
		1: {
			NewSavepoint().OnCheck(),
			help.WriteHandler(nk, nv, derr),
			true,
			true,
			[][]byte{ok},
			[][]byte{nk},
		},
		// savepoint activated for deliver, returns error, one written
		2: {
			NewSavepoint().OnDeliver(),
			help.WriteHandler(nk, nv, derr),
			false,
			true,
			[][]byte{ok},
			[][]byte{nk},
		},
		// double-activation maintains both behaviors
		3: {
			NewSavepoint().OnDeliver().OnCheck(),
			help.WriteHandler(nk, nv, derr),
			false,
			true,
			[][]byte{ok},
			[][]byte{nk},
		},
		// savepoint check doesn't affect deliver
		4: {
			NewSavepoint().OnCheck(),
			help.WriteHandler(nk, nv, derr),
			false,
			true,
			[][]byte{ok, nk},
			nil,
		},
		// don't rollback when success returned
		5: {
			NewSavepoint().OnCheck().OnDeliver(),
			help.WriteHandler(nk, nv, nil),
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, k := range tc.written {
				assert.True(t, kv.Has(k), "%x", k)
			}
			for _, k := range tc.missing {
				assert.False(t, kv.Has(k), "%x", k)
			}
		})
	}
}
