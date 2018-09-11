package multisig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iov-one/weave/store"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

var (
	newTx   = x.TestHelpers{}.MockTx
	helpers = x.TestHelpers{}
)

// newContextWithAuth creates a context with perms as signers and sets the height
func newContextWithAuth(perms ...weave.Condition) (weave.Context, x.Authenticator) {
	ctx := context.Background()
	// Set current block height to 100
	ctx = weave.WithHeight(ctx, 100)
	auth := helpers.CtxAuth("authKey")
	// Create a new context and add addr to the list of signers
	return auth.SetConditions(ctx, perms...), auth
}

func TestCreateContractMsgHandlerValidate(t *testing.T) {
	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()
	_, c := helpers.MakeKey()
	_, d := helpers.MakeKey()

	testcases := []struct {
		name string
		msg  *CreateContractMsg
		err  error
	}{
		{
			name: "valid use case",
			msg: &CreateContractMsg{
				Address:             d.Address(),
				Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
				ActivationThreshold: 2,
				ChangeThreshold:     3,
			},
			err: nil,
		},
		{
			name: "missing sigs",
			msg:  &CreateContractMsg{},
			err:  errors.ErrUnrecognizedAddress(nil),
		},
		{
			name: "missing sigs",
			msg:  &CreateContractMsg{Address: d.Address()},
			err:  ErrMissingSigs(),
		},
		{
			name: "bad activation threshold",
			msg: &CreateContractMsg{
				Address:             d.Address(),
				Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
				ActivationThreshold: 4,
				ChangeThreshold:     3,
			},
			err: ErrInvalidActivationThreshold(),
		},
		{
			name: "bad activation threshold",
			msg: &CreateContractMsg{
				Address:             d.Address(),
				Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
				ActivationThreshold: 1,
				ChangeThreshold:     -1,
			},
			err: ErrInvalidChangeThreshold(),
		},
	}

	db := store.MemStore()
	ctx, auth := newContextWithAuth(a)
	handler := CreateContractMsgHandler{auth, NewContractBucket()}
	for _, test := range testcases {
		_, err := handler.validate(ctx, db, newTx(test.msg))
		if test.err == nil {
			require.NoError(t, err, test.name)
		} else {
			require.EqualError(t, err, test.err.Error(), test.name)
		}
	}
}

func TestCreateContractMsgHandlerCheck(t *testing.T) {
	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()
	_, c := helpers.MakeKey()
	_, d := helpers.MakeKey()

	db := store.MemStore()
	ctx, auth := newContextWithAuth(a)
	handler := CreateContractMsgHandler{auth, NewContractBucket()}
	res, err := handler.Check(
		ctx,
		db,
		newTx(&CreateContractMsg{
			Address:             d.Address(),
			Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
			ActivationThreshold: 2,
			ChangeThreshold:     3,
		}))

	require.NoError(t, err)
	require.Equal(t, creationCost, res.GasAllocated)
}

func TestCreateContractMsgHandlerDeliver(t *testing.T) {
	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()
	_, c := helpers.MakeKey()
	_, d := helpers.MakeKey()

	db := store.MemStore()
	ctx, auth := newContextWithAuth(a)
	handler := CreateContractMsgHandler{auth, NewContractBucket()}
	_, err := handler.Deliver(
		ctx,
		db,
		newTx(&CreateContractMsg{
			Address:             d.Address(),
			Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
			ActivationThreshold: 2,
			ChangeThreshold:     3,
		}))

	require.NoError(t, err)
	obj, err := handler.bucket.Get(db, d.Address())
	require.NoError(t, err)
	require.NotNil(t, obj)
	require.EqualValues(t,
		Contract{[][]byte{a.Address(), b.Address(), c.Address()}, 2, 3},
		*obj.Value().(*Contract))
}
