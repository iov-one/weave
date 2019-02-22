package multisig

import (
	"fmt"
	"github.com/iov-one/weave/errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
)

func TestDecorator(t *testing.T) {
	var helpers x.TestHelpers
	db := store.MemStore()

	// create some keys
	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()
	_, c := helpers.MakeKey()
	_, d := helpers.MakeKey()
	_, e := helpers.MakeKey()
	_, f := helpers.MakeKey()

	// the contract we'll be using in our tests
	contractID1 := withContract(t, db, CreateContractMsg{
		Sigs:                newSigs(a, b, c),
		ActivationThreshold: 2,
		AdminThreshold:      3,
	})

	// contractID2 is used as a sig for contractID3
	contractID2 := withContract(t, db, CreateContractMsg{
		Sigs:                newSigs(d, e, f),
		ActivationThreshold: 2,
		AdminThreshold:      3,
	})

	// contractID3 requires either sig for a or activation for contractID2
	contractID3 := withContract(t, db, CreateContractMsg{
		Sigs:                newSigs(a, MultiSigCondition(contractID2)),
		ActivationThreshold: 1,
		AdminThreshold:      2,
	})

	// helper to create a ContractTx
	multisigTx := func(payload []byte, multisig ...[]byte) ContractTx {
		tx := helpers.MockTx(helpers.MockMsg(payload))
		return ContractTx{Tx: tx, MultisigID: multisig}
	}

	cases := []struct {
		tx      weave.Tx
		signers []weave.Condition
		perms   []weave.Condition
		err     error
	}{
		// doesn't support multisig interface
		{
			helpers.MockTx(helpers.MockMsg([]byte{1, 2, 3})),
			[]weave.Condition{a},
			nil,
			nil,
		},
		// Correct interface but no content
		{
			multisigTx([]byte("john"), nil),
			[]weave.Condition{a},
			nil,
			nil,
		},
		// with multisig contract
		{
			multisigTx([]byte("foo"), contractID1),
			[]weave.Condition{a, b},
			[]weave.Condition{MultiSigCondition(contractID1)},
			nil,
		},
		// with multisig contract but not enough signatures to activate
		{
			multisigTx([]byte("foo"), contractID1),
			[]weave.Condition{a},
			nil,
			errors.ErrUnauthorized.Newf("contract=%X", contractID1),
		},
		// with invalid multisig contract ID
		{
			multisigTx([]byte("foo"), []byte("bad id")),
			[]weave.Condition{a, b},
			nil,
			errors.ErrNotFound.Newf(contractNotFoundFmt, []byte("bad id")),
		},
		// contractID3 is activated by contractID2
		{
			multisigTx([]byte("foo"), contractID2, contractID3),
			[]weave.Condition{d, e},
			[]weave.Condition{MultiSigCondition(contractID2), MultiSigCondition(contractID3)},
			nil,
		},
		// contractID3 is activated by a
		{
			multisigTx([]byte("foo"), contractID3),
			[]weave.Condition{a},
			[]weave.Condition{MultiSigCondition(contractID3)},
			nil,
		},
		// contractID3 is not activated
		{
			multisigTx([]byte("foo"), contractID3),
			[]weave.Condition{d, e}, // cconditions for ontractID2 are there but ontractID2 must be passed explicitly
			nil,
			errors.ErrUnauthorized.Newf("contract=%X", contractID3),
		},
	}

	// the handler we're chaining with the decorator
	h := new(MultisigCheckHandler)
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			ctx, auth := newContextWithAuth(tc.signers...)
			d := NewDecorator(x.ChainAuth(auth, Authenticate{}))

			stack := helpers.Wrap(d, h)
			_, err := stack.Check(ctx, db, tc.tx)
			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.perms, h.Perms)
			}

			_, err = stack.Deliver(ctx, db, tc.tx)
			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.perms, h.Perms)
			}
		})
	}
}

//---------------- helpers --------

// MultisigCheckHandler stores the seen permissions on each call
// for this extension's authenticator (ie. multisig.Authenticate)
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
	MultisigID [][]byte
}

var _ MultiSigTx = ContractTx{}
var _ weave.Tx = ContractTx{}

func (p ContractTx) GetMultisig() [][]byte {
	return p.MultisigID
}
