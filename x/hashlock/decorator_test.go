package hashlock

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
)

func TestDecorator(t *testing.T) {
	var helpers x.TestHelpers

	h := new(HashCheckHandler)
	d := NewDecorator()
	stack := helpers.Wrap(d, h)

	db := store.MemStore()
	bg := context.Background()

	hashTx := func(payload, preimage []byte) PreimageTx {
		tx := helpers.MockTx(helpers.MockMsg(payload))
		return PreimageTx{Tx: tx, Preimage: preimage}
	}

	cases := []struct {
		tx    weave.Tx
		perms []weave.Condition
	}{
		// doesn't support hashlock interface
		{helpers.MockTx(helpers.MockMsg([]byte{1, 2, 3})), nil},
		// Correct interface but no content
		{hashTx([]byte("john"), nil), nil},
		// Hash a preimage
		{hashTx([]byte("foo"), []byte("bar")),
			[]weave.Condition{PreimageCondition([]byte("bar"))}},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			_, err := stack.Check(bg, db, tc.tx)
			require.NoError(t, err)
			assert.Equal(t, tc.perms, h.Perms)

			_, err = stack.Deliver(bg, db, tc.tx)
			require.NoError(t, err)
			assert.Equal(t, tc.perms, h.Perms)
		})
	}
}

//---------------- helpers --------

// HashCheckHandler stores the seen permissions on each call
type HashCheckHandler struct {
	Perms []weave.Condition
}

var _ weave.Handler = (*HashCheckHandler)(nil)

func (s *HashCheckHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.CheckResult, err error) {
	s.Perms = Authenticate{}.GetConditions(ctx)
	return
}

func (s *HashCheckHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.DeliverResult, err error) {
	s.Perms = Authenticate{}.GetConditions(ctx)
	return
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
