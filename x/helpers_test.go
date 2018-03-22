package x

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/confio/weave"
	"github.com/confio/weave/store"
)

type pair struct {
	key   []byte
	value []byte
}

func TestHelperFuncs(t *testing.T) {
	var helper TestHelpers

	a := []byte("a")
	b := []byte("b")
	body := []byte("body")
	tx := helper.MockTx(helper.MockMsg(body))
	err := fmt.Errorf("test error")

	cases := []struct {
		h           weave.Handler
		height      int64
		tx          weave.Tx
		expectPanic bool
		expectError bool
		expectCount int // after called both check and deliver
		queries     []pair
		tags        []pair
	}{
		0: {
			helper.WriteHandler(a, b, nil),
			1, tx,
			false, false, 0,
			[]pair{{a, b}},
			nil,
		},
		1: {
			helper.CountingHandler(),
			1, tx,
			false, false, 2,
			nil,
			nil,
		},
		2: {
			helper.PanicHandler(err),
			1, tx,
			true, false, 0,
			nil,
			nil,
		},
		3: {
			helper.Wrap(helper.ErrorDecorator(err),
				helper.WriteHandler(a, b, nil)),
			1, tx,
			false, true, 0,
			[]pair{{a, nil}},
			nil,
		},
		4: {
			helper.Wrap(helper.PanicAtHeightDecorator(5),
				helper.WriteHandler(a, b, nil)),
			3, tx,
			false, false, 0,
			[]pair{{a, b}},
			nil,
		},
		5: {
			helper.Wrap(helper.PanicAtHeightDecorator(5),
				helper.WriteHandler(a, b, nil)),
			8, tx,
			true, false, 0,
			[]pair{{a, nil}},
			nil,
		},
		6: {
			helper.Wrap(helper.CountingDecorator(),
				helper.WriteHandler(a, b, err)),
			1, tx,
			false, true, 0,
			[]pair{{a, b}},
			nil,
		},
		7: {
			helper.Wrap(helper.WriteDecorator(a, b, true),
				helper.ErrorHandler(err)),
			1, tx,
			false, true, 0,
			[]pair{{a, nil}},
			nil,
		},
		8: {
			helper.Wrap(helper.WriteDecorator(a, b, false),
				helper.ErrorHandler(err)),
			1, tx,
			false, true, 0,
			[]pair{{a, b}},
			nil,
		},
		9: {
			helper.Wrap(helper.WriteDecorator(a, b, true),
				helper.TagHandler(b, a, nil)),
			1, tx,
			false, false, 0,
			[]pair{{a, b}},
			[]pair{{b, a}},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			db := store.MemStore()
			ctx := weave.WithHeight(context.Background(), tc.height)
			if tc.expectPanic {
				assert.Panics(t, func() { tc.h.Check(ctx, db, tc.tx) })
				assert.Panics(t, func() { tc.h.Deliver(ctx, db, tc.tx) })
				return
			}
			_, err := tc.h.Check(ctx, db.CacheWrap(), tc.tx)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			dres, err := tc.h.Deliver(ctx, db, tc.tx)
			if counter, ok := tc.h.(CountingHandler); ok {
				assert.Equal(t, tc.expectCount, counter.GetCount())
			}

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, q := range tc.queries {
				v := db.Get(q.key)
				assert.EqualValues(t, q.value, v)
			}
			if assert.Equal(t, len(dres.Tags), len(tc.tags)) {
				for i, tag := range tc.tags {
					assert.Equal(t, tag.key, dres.Tags[i].Key)
					assert.Equal(t, tag.value, dres.Tags[i].Value)
				}
			}
		})
	}
}
