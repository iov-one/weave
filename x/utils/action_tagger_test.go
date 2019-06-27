package utils_test

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/batch"
	"github.com/iov-one/weave/x/utils"
	"github.com/tendermint/tendermint/libs/common"
)

func stringTag(key, value string) common.KVPair {
	return common.KVPair{
		Key:   []byte(key),
		Value: []byte(value),
	}
}

func TestActionTagger(t *testing.T) {
	cases := map[string]struct {
		stack weave.Handler
		tx    weave.Tx
		err   *errors.Error
		tags  []common.KVPair
	}{
		"simple call": {
			stack: app.ChainDecorators(utils.NewActionTagger()).WithHandler(
				&weavetest.Handler{},
			),
			tx:   &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foobar/create"}},
			tags: []common.KVPair{stringTag(utils.ActionKey, "foobar/create")},
		},
		"passes through error": {
			stack: app.ChainDecorators(utils.NewActionTagger()).WithHandler(
				&weavetest.Handler{DeliverErr: errors.ErrHuman},
			),
			tx:  &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foobar/create"}},
			err: errors.ErrHuman,
		},
		"tags are additive": {
			stack: app.ChainDecorators(utils.NewActionTagger()).WithHandler(
				&weavetest.Handler{
					DeliverResult: weave.DeliverResult{Tags: []common.KVPair{stringTag(utils.ActionKey, "random")}},
				},
			),
			tx:   &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "foobar/create"}},
			tags: []common.KVPair{stringTag(utils.ActionKey, "random"), stringTag(utils.ActionKey, "foobar/create")},
		},
		"all in batch are tagged": {
			stack: app.ChainDecorators(
				batch.NewDecorator(),
				utils.NewActionTagger(),
			).WithHandler(
				&weavetest.Handler{},
			),
			tx: &weavetest.Tx{Msg: &batchMsg{
				msgs: []weave.Msg{
					&weavetest.Msg{RoutePath: "username/register"},
					&weavetest.Msg{RoutePath: "cash/send"},
					&weavetest.Msg{RoutePath: "gov/vote"},
				},
			}},
			tags: []common.KVPair{
				stringTag(utils.ActionKey, "username/register"),
				stringTag(utils.ActionKey, "cash/send"),
				stringTag(utils.ActionKey, "gov/vote"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := store.MemStore()

			// we get tagged on success
			res, err := tc.stack.Deliver(ctx, store, tc.tx)
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

var _ batch.Msg = (*batchMsg)(nil)

type batchMsg struct {
	msgs []weave.Msg
}

func (m *batchMsg) Marshal() ([]byte, error) {
	panic("implement me")
}

func (m *batchMsg) Unmarshal([]byte) error {
	panic("implement me")
}

func (m *batchMsg) Path() string {
	panic("implement me")
}

func (m *batchMsg) Validate() error {
	return nil
}

func (m *batchMsg) MsgList() ([]weave.Msg, error) {
	return m.msgs, nil
}
