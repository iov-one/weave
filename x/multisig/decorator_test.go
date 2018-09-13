package multisig

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

	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()
	_, c := helpers.MakeKey()

	h := new(MultisigCheckHandler)
	d := NewDecorator()
	stack := helpers.Wrap(d, h)

	db := store.MemStore()
	auth := helpers.CtxAuth("multisig")
	bg := auth.SetConditions(context.Background(), a, b, c)
	hc := CreateContractMsgHandler{auth, NewContractBucket()}
	res, err := hc.Deliver(bg, db,
		helpers.MockTx(
			&CreateContractMsg{
				Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
				ActivationThreshold: 2,
				ChangeThreshold:     3,
			}))
	require.NoError(t, err)
	contractID := res.Data

	multisigTx := func(payload, multisig []byte) ContractTx {
		tx := helpers.MockTx(helpers.MockMsg(payload))
		return ContractTx{Tx: tx, Multisig: multisig}
	}

	cases := []struct {
		tx    weave.Tx
		perms []weave.Address
	}{
		// doesn't support multisig interface
		{helpers.MockTx(helpers.MockMsg([]byte{1, 2, 3})), nil},
		// Correct interface but no content
		{multisigTx([]byte("john"), nil), nil},
		// Hash a preimage
		{multisigTx([]byte("foo"), []byte("bar")),
			[]weave.Address{MultiSigCondition(contractID).Address()}},
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
type MultisigCheckHandler struct {
	Perms []weave.Condition
}

var _ weave.Handler = (*MultisigCheckHandler)(nil)

func (s *MultisigCheckHandler) Check(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.CheckResult, err error) {
	s.Perms = Authenticate{}.GetConditions(ctx)
	return
}

func (s *MultisigCheckHandler) Deliver(ctx weave.Context, store weave.KVStore,
	tx weave.Tx) (res weave.DeliverResult, err error) {
	s.Perms = Authenticate{}.GetConditions(ctx)
	return
}

// ContractTx fulfills the MultiSigTx interface to satisfy the decorator
type ContractTx struct {
	weave.Tx
	Multisig weave.Address
}

var _ MultiSigTx = ContractTx{}
var _ weave.Tx = ContractTx{}

func (p ContractTx) GetMultiSig() weave.Address {
	return p.Multisig
}
