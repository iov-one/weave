package utils

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/common"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
)

func TestKeyTagger(t *testing.T) {
	var help x.TestHelpers

	// always write ok, ov before calling functions
	ok, ov := []byte("foo:demo"), []byte("data")
	// some key, value to try to write
	nk, nv := []byte{1, 0xab, 3}, []byte{4, 5, 6}
	// a default error if desired
	derr := fmt.Errorf("something went wrong")

	otag, oval := []byte("666F6F3A64656D6F"), []byte("s") // "foo:demo" as upper-case hex
	ntag, nval := []byte("01AB03"), []byte("s")

	cases := [...]struct {
		handler weave.Handler
		isError bool // true iff we expect errors
		tags    common.KVPairs
		k, v    []byte
	}{
		// return error doesn't add tags
		0: {
			help.WriteHandler(nk, nv, derr),
			true,
			nil,
			// note that these were writen as we had no SavePoint
			nk,
			nv,
		},
		// with success records tags
		1: {
			help.WriteHandler(nk, nv, nil),
			false,
			common.KVPairs{{Key: ntag, Value: nval}},
			nk,
			nv,
		},
		// write multiple values (sorted order)
		2: {
			help.Wrap(help.WriteDecorator(ok, ov, true),
				help.WriteHandler(nk, nv, nil)),
			false,
			common.KVPairs{{Key: ntag, Value: nval}, {Key: otag, Value: oval}},
			nk,
			nv,
		},
		// savepoint must revert any writes
		3: {
			help.Wrap(NewSavepoint().OnDeliver(),
				help.WriteHandler(nk, nv, derr)),
			true,
			nil,
			nk,
			nil,
		},
		// savepoint keeps writes on success
		4: {
			help.Wrap(NewSavepoint().OnDeliver(),
				help.WriteHandler(nk, nv, nil)),
			false,
			common.KVPairs{{Key: ntag, Value: nval}},
			nk,
			nv,
		},
		// combine with other tags from the Handler
		5: {
			help.Wrap(help.WriteDecorator(ok, ov, false),
				help.TagHandler(nk, nv, nil)),
			false,
			// note that the nk, nv set explicitly are not modified
			common.KVPairs{{Key: nk, Value: nv}, {Key: otag, Value: oval}},
			nk,
			nil,
		},
		// on error don't add tags, but leave original ones
		6: {
			help.Wrap(help.WriteDecorator(ok, ov, false),
				help.TagHandler(nk, nv, derr)),
			true,
			common.KVPairs{{Key: nk, Value: nv}},
			nk,
			nil,
		},
		// TODO: also check delete
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			ctx := context.Background()
			db := store.MemStore()
			tagger := NewKeyTagger()

			// try check - no op
			check := db.CacheWrap()
			_, err := tagger.Check(ctx, check, nil, tc.handler)
			if tc.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// try deliver - records tags and sets values on success
			res, err := tagger.Deliver(ctx, db, nil, tc.handler)
			if tc.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// tags are set properly
				assert.EqualValues(t, tc.tags, res.Tags)
			}

			// optionally check if data was writen to underlying db
			if tc.k != nil {
				v := db.Get(tc.k)
				assert.EqualValues(t, tc.v, v)
			}
		})
	}
}
