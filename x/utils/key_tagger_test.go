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
	// ok, ov := []byte("demo"), []byte("data")
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
		// return error with no tags
		0: {
			help.WriteHandler(nk, nv, derr),
			true,
			nil,
			nil,
			nil,
		},
		// with success records tags
		1: {
			help.WriteHandler(nk, nv, nil),
			false,
			common.KVPairs{{Key: nk, Value: recordSet}},
			nk,
			nv,
		},
		// TODO:
		//  - write multiple values (sorted order)
		//  - overwrite value (set/delete)

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
				return
			}

			assert.NoError(t, err)
			// tags are set properly
			assert.EqualValues(t, tc.tags, res.Tags)
			// data is also writen to underlying db
			if tc.k != nil {
				v := db.Get(tc.k)
				assert.EqualValues(t, tc.v, v)
			}
		})
	}
}
