package hashlock

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestDecorator(t *testing.T) {
	h := new(HashCheckHandler)
	d := NewDecorator()
	stack := weavetest.Decorate(h, d)

	db := store.MemStore()
	bg := context.Background()

	hashTx := func(payload, preimage []byte) PreimageTx {
		return PreimageTx{
			Tx: &weavetest.Tx{
				Msg: &weavetest.Msg{Serialized: payload},
			},
			Preimage: preimage,
		}
	}

	cases := map[string]struct {
		tx    weave.Tx
		perms []weave.Condition
	}{
		"doesn't support hashlock interface": {
			tx: &weavetest.Tx{
				Msg: &weavetest.Msg{
					Serialized: []byte{1, 2, 3},
				},
			},
		},
		"Correct interface but no content": {
			tx: hashTx([]byte("john"), nil),
		},
		"Hash a preimage": {
			tx:    hashTx([]byte("foo"), []byte("bar")),
			perms: []weave.Condition{PreimageCondition([]byte("bar"))},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			_, err := stack.Check(bg, db, tc.tx)
			assert.Nil(t, err)
			assert.Equal(t, tc.perms, h.Perms)

			_, err = stack.Deliver(bg, db, tc.tx)
			assert.Nil(t, err)
			assert.Equal(t, tc.perms, h.Perms)
		})
	}
}

// HashCheckHandler stores the seen permissions on each call
type HashCheckHandler struct {
	Perms []weave.Condition
}

var _ weave.Handler = (*HashCheckHandler)(nil)

func (s *HashCheckHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	s.Perms = Authenticate{}.GetConditions(ctx)
	return &weave.CheckResult{}, nil
}

func (s *HashCheckHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	s.Perms = Authenticate{}.GetConditions(ctx)
	return &weave.DeliverResult{}, nil
}

// PreimageTx fulfills the HashKeyTx interface to satisfy the decorator
type PreimageTx struct {
	weave.Tx
	Preimage []byte
}

var _ HashKeyTx = PreimageTx{}
var _ weave.Tx = PreimageTx{}

func (p PreimageTx) GetPreimage() []byte {
	return p.Preimage
}
