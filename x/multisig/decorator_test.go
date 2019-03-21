package multisig

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x"
	"github.com/stretchr/testify/require"
)

func TestDecorator(t *testing.T) {
	db := store.MemStore()

	// create some keys
	a := weavetest.NewCondition()
	b := weavetest.NewCondition()
	c := weavetest.NewCondition()
	d := weavetest.NewCondition()
	e := weavetest.NewCondition()
	f := weavetest.NewCondition()

	// the contract we'll be using in our tests
	contractID1 := createContract(t, db, CreateContractMsg{
		Participants: []*Participant{
			{Power: 1, Signature: a.Address()},
			{Power: 1, Signature: b.Address()},
			{Power: 1, Signature: c.Address()},
		},
		ActivationThreshold: 2,
		AdminThreshold:      3,
	})

	// contractID2 is used as a sig for contractID3
	contractID2 := createContract(t, db, CreateContractMsg{
		Participants: []*Participant{
			{Power: 1, Signature: d.Address()},
			{Power: 1, Signature: e.Address()},
			{Power: 1, Signature: f.Address()},
		},
		ActivationThreshold: 2,
		AdminThreshold:      3,
	})

	// contractID3 requires either sig for a or activation for contractID2
	contractID3 := createContract(t, db, CreateContractMsg{
		Participants: []*Participant{
			{Power: 1, Signature: a.Address()},
			{Power: 1, Signature: MultiSigCondition(contractID2).Address()},
		},
		ActivationThreshold: 1,
		AdminThreshold:      2,
	})
	// helper to create a ContractTx
	multisigTx := func(payload []byte, multisig ...[]byte) ContractTx {
		tx := &weavetest.Tx{Msg: &weavetest.Msg{Serialized: payload}}
		return ContractTx{Tx: tx, MultisigID: multisig}
	}

	cases := map[string]struct {
		tx      weave.Tx
		signers []weave.Condition
		perms   []weave.Condition
		wantGas int64
		wantErr *errors.Error
	}{
		"does not support multisig interface": {
			tx:      &weavetest.Tx{Msg: &weavetest.Msg{Serialized: []byte{1, 2, 3}}},
			signers: []weave.Condition{a},
		},
		"correct interface but no content": {
			tx:      multisigTx([]byte("john"), nil),
			signers: []weave.Condition{a},
		},
		"with multisig contract": {
			tx:      multisigTx([]byte("foo"), contractID1),
			signers: []weave.Condition{a, b},
			perms:   []weave.Condition{MultiSigCondition(contractID1)},
			wantGas: multisigParticipantGasCost * 2,
		},
		"with multisig contract but not enough signatures to activate": {
			tx:      multisigTx([]byte("foo"), contractID1),
			signers: []weave.Condition{a},
			wantErr: errors.ErrUnauthorized,
		},
		"with invalid multisig contract ID": {
			tx:      multisigTx([]byte("foo"), []byte("bad id")),
			signers: []weave.Condition{a, b},
			wantErr: errors.ErrNotFound,
		},
		"contractID3 is activated by contractID2": {
			tx:      multisigTx([]byte("foo"), contractID2, contractID3),
			signers: []weave.Condition{d, e},
			perms:   []weave.Condition{MultiSigCondition(contractID2), MultiSigCondition(contractID3)},
			wantGas: multisigParticipantGasCost * 3,
		},
		"contractID3 is activated by a": {
			tx:      multisigTx([]byte("foo"), contractID3),
			signers: []weave.Condition{a},
			perms:   []weave.Condition{MultiSigCondition(contractID3)},
			wantGas: multisigParticipantGasCost * 1,
		},
		"contractID3 is not activated": {
			tx: multisigTx([]byte("foo"), contractID3),
			// cconditions for ontractID2 are there but ontractID2 must be passed explicitly
			signers: []weave.Condition{d, e},
			wantErr: errors.ErrUnauthorized,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			ctx := context.Background()
			ctx = weave.WithHeight(ctx, 100)
			auth := &weavetest.CtxAuth{Key: "authKey"}
			ctx = auth.SetConditions(ctx, tc.signers...)
			d := NewDecorator(x.ChainAuth(auth, Authenticate{}))

			var hn MultisigCheckHandler
			stack := weavetest.Decorate(&hn, d)

			cres, err := stack.Check(ctx, db, tc.tx)
			if !tc.wantErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}
			if cres.GasPayment != tc.wantGas {
				t.Errorf("want %d gas payment, got %d", tc.wantGas, cres.GasPayment)
			}

			if _, err := stack.Deliver(ctx, db, tc.tx); !tc.wantErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
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

func createContract(t *testing.T, db weave.KVStore, msg CreateContractMsg) []byte {
	k := weavetest.NewCondition()
	ctx := context.Background()
	ctx = weave.WithHeight(ctx, 100)
	auth := &weavetest.CtxAuth{Key: "authKey"}
	ctx = auth.SetConditions(ctx, k)
	handler := CreateContractMsgHandler{
		auth:   auth,
		bucket: NewContractBucket(),
	}
	res, err := handler.Deliver(ctx, db, &weavetest.Tx{Msg: &msg})
	require.NoError(t, err, "cannot deliver")
	return res.Data
}
