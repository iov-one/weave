package utils

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/tendermint/tendermint/libs/common"
)

func TestActionTagger(t *testing.T) {
	cases := map[string]struct {
		wrap weave.Decorator
		h    weave.Handler
		tx   weave.Tx
		err  *errors.Error
		tags []common.KVPair
	}{
		"simple call": {
			wrap: NewActionTagger(),
			h:    &weavetest.Handler{},
			tx:   &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foobar/create"}},
			tags: []common.KVPair{{Key: []byte("action"), Value: []byte("foobar/create")}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := store.MemStore()

			// we get tagged on success
			res, err := tc.wrap.Deliver(ctx, store, tc.tx, tc.h)
			if tc.err != nil {
				if !tc.err.Is(err) {
					t.Fatalf("Unexpected error type returned: %v", err)
				}
				return
			}
			assert.Nil(t, err)
			assert.Equal(t, len(tc.tags), len(res.Tags))
			for i := range tc.tags {
				assert.Equal(t, string(tc.tags[i].Key), string(res.Tags[i].Key))
				assert.Equal(t, string(tc.tags[i].Value), string(res.Tags[i].Value))
			}
		})
	}
}
