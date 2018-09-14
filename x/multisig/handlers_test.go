package multisig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iov-one/weave/store"

	"github.com/iov-one/weave"
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

	testcases := []struct {
		name string
		msg  *CreateContractMsg
		err  error
	}{
		{
			name: "valid use case",
			msg: &CreateContractMsg{
				Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
				ActivationThreshold: 2,
				AdminThreshold:      3,
			},
			err: nil,
		},
		{
			name: "missing sigs",
			msg:  &CreateContractMsg{},
			err:  ErrMissingSigs(),
		},
		{
			name: "bad activation threshold",
			msg: &CreateContractMsg{
				Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
				ActivationThreshold: 4,
				AdminThreshold:      3,
			},
			err: ErrInvalidActivationThreshold(),
		},
		{
			name: "bad admin threshold",
			msg: &CreateContractMsg{
				Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
				ActivationThreshold: 1,
				AdminThreshold:      -1,
			},
			err: ErrInvalidChangeThreshold(),
		},
		{
			name: "0 activation threshold",
			msg: &CreateContractMsg{
				Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
				ActivationThreshold: 0,
				AdminThreshold:      1,
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

	db := store.MemStore()
	ctx, auth := newContextWithAuth(a)
	handler := CreateContractMsgHandler{auth, NewContractBucket()}
	res, err := handler.Check(
		ctx,
		db,
		newTx(&CreateContractMsg{
			Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
			ActivationThreshold: 2,
			AdminThreshold:      3,
		}))

	require.NoError(t, err)
	require.Equal(t, creationCost, res.GasAllocated)
}

func TestCreateContractMsgHandlerDeliver(t *testing.T) {
	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()
	_, c := helpers.MakeKey()

	db := store.MemStore()
	ctx, auth := newContextWithAuth(a)
	handler := CreateContractMsgHandler{auth, NewContractBucket()}
	res, err := handler.Deliver(
		ctx,
		db,
		newTx(&CreateContractMsg{
			Sigs:                [][]byte{a.Address(), b.Address(), c.Address()},
			ActivationThreshold: 2,
			AdminThreshold:      3,
		}))
	require.NoError(t, err)

	objKey := res.Data
	obj, err := handler.bucket.Get(db, objKey)
	require.NoError(t, err)
	require.NotNil(t, obj)
	require.EqualValues(t,
		Contract{[][]byte{a.Address(), b.Address(), c.Address()}, 2, 3},
		*obj.Value().(*Contract))
}

func newSigs(perms ...weave.Condition) [][]byte {
	// initial addresses controlling contract
	var sigs [][]byte
	for _, p := range perms {
		sigs = append(sigs, p.Address())
	}
	return sigs
}

func withContract(t *testing.T, db weave.KVStore, msg CreateContractMsg) []byte {
	_, k := helpers.MakeKey()
	ctx, auth := newContextWithAuth(k)
	handler := CreateContractMsgHandler{auth, NewContractBucket()}
	res, err := handler.Deliver(
		ctx,
		db,
		newTx(&msg))
	require.NoError(t, err)
	return res.Data
}

func queryContract(t *testing.T, db weave.KVStore, handler UpdateContractMsgHandler, id []byte) Contract {
	// run query
	contracts, err := handler.bucket.Query(db, "", id)
	require.NoError(t, err)
	require.Len(t, contracts, 1)

	actual, err := handler.bucket.Parse(nil, contracts[0].Value)
	return *actual.Value().(*Contract)
}

func TestUpdateContractMsgHandler(t *testing.T) {
	db := store.MemStore()

	// addresses controlling contract
	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()
	_, c := helpers.MakeKey()
	_, d := helpers.MakeKey()
	_, e := helpers.MakeKey()

	mutableID := withContract(t, db,
		CreateContractMsg{
			Sigs:                newSigs(a, b, c),
			ActivationThreshold: 1,
			AdminThreshold:      2,
		})

	immutableID := withContract(t, db,
		CreateContractMsg{
			Sigs:                newSigs(a, b, c),
			ActivationThreshold: 1,
			AdminThreshold:      4,
		})

	testcases := []struct {
		name    string
		msg     *UpdateContractMsg
		signers []weave.Condition
		err     error
	}{
		{
			name: "authorized",
			msg: &UpdateContractMsg{
				Id:                  mutableID,
				Sigs:                newSigs(a, b, c, d, e),
				ActivationThreshold: 4,
				AdminThreshold:      5,
			},
			signers: []weave.Condition{a, b},
			err:     nil,
		},
		{
			name: "unauthorised",
			msg: &UpdateContractMsg{
				Id:                  mutableID,
				Sigs:                newSigs(a, b, c, d, e),
				ActivationThreshold: 4,
				AdminThreshold:      5,
			},
			signers: []weave.Condition{a},
			err:     ErrUnauthorizedMultiSig(mutableID),
		},
		{
			name: "immutable",
			msg: &UpdateContractMsg{
				Id:                  immutableID,
				Sigs:                newSigs(a, b, c, d, e),
				ActivationThreshold: 4,
				AdminThreshold:      5,
			},
			signers: []weave.Condition{a, b, c, d, e},
			err:     ErrUnauthorizedMultiSig(immutableID),
		},
		{
			name: "bad change threshold",
			msg: &UpdateContractMsg{
				Id:                  mutableID,
				Sigs:                newSigs(a, b, c, d, e),
				ActivationThreshold: 1,
				AdminThreshold:      0,
			},
			signers: []weave.Condition{a, b, c, d, e},
			err:     ErrInvalidChangeThreshold(),
		},
	}

	for _, test := range testcases {
		msg := test.msg
		ctx, auth := newContextWithAuth(test.signers...)
		handler := UpdateContractMsgHandler{auth, NewContractBucket()}

		_, err := handler.Check(ctx, db, newTx(msg))
		if test.err == nil {
			require.NoError(t, err, test.name)
		} else {
			require.EqualError(t, err, test.err.Error(), test.name)
		}

		_, err = handler.Deliver(ctx, db, newTx(msg))
		if test.err == nil {
			require.NoError(t, err, test.name)
			contract := queryContract(t, db, handler, msg.Id)
			require.EqualValues(t,
				Contract{msg.Sigs, msg.ActivationThreshold, msg.AdminThreshold},
				contract,
				test.name)
		} else {
			require.EqualError(t, err, test.err.Error(), test.name)
		}
	}
}
