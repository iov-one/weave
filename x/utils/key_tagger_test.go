package utils

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tmlibs/common"

	"github.com/confio/weave"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
)

func TestKeyTagger(t *testing.T) {
	var help x.TestHelpers

	// always write ok, ov before calling functions
	ok, ov := []byte("demo"), []byte("data")
	// some key, value to try to write
	nk, nv := []byte{1, 2, 3}, []byte{4, 5, 6}
	// a default error if desired
	derr := fmt.Errorf("something went wrong")

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
			common.KVPairs{{Key: nk, Value: recordSet}},
			nk,
			nv,
		},
		// write multiple values (sorted order)
		2: {
			help.Wrap(help.WriteDecorator(ok, ov, false),
				help.WriteHandler(nk, nv, nil)),
			false,
			common.KVPairs{{Key: nk, Value: recordSet}, {Key: ok, Value: recordSet}},
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
			common.KVPairs{{Key: nk, Value: recordSet}},
			nk,
			nv,
		},
		// combine with other tags from the Handler
		5: {
			help.Wrap(help.WriteDecorator(ok, ov, false),
				help.TagHandler(nk, nv, nil)),
			false,
			common.KVPairs{{Key: nk, Value: nv}, {Key: ok, Value: recordSet}},
			nk,
			nil,
		},
		// on error don't add tags, but leave original ones
		6: {
			help.Wrap(help.WriteDecorator(ok, ov, false),
				help.TagHandler(nk, nv, derr)),
			true,
			common.KVPairs{{Key: nk, Value: recordSet}},
			nk,
			nil,
		},
		// TODO: also check delete
	}

	for i, tc := range cases {
		if i != 4 {
			continue
		}
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
