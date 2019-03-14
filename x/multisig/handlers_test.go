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

// newContextWithAuth creates a context with perms as signers and sets the height
func newContextWithAuth(perms ...weave.Condition) (weave.Context, x.Authenticator) {
	ctx := context.Background()
	// Set current block height to 100
	ctx = weave.WithHeight(ctx, 100)
	auth := &weavetest.CtxAuth{Key: "authKey"}
	// Create a new context and add addr to the list of signers
	return auth.SetConditions(ctx, perms...), auth
}

// newSigs creates an array with addresses from each condition
func newSigs(perms ...weave.Condition) [][]byte {
	// initial addresses controlling contract
	var sigs [][]byte
	for _, p := range perms {
		sigs = append(sigs, p.Address())
	}
	return sigs
}

// queryContract queries a contract from the bucket and handles errors
// so you get a strongly typed object or a test failure
func queryContract(t *testing.T, db weave.KVStore, bucket ContractBucket, id []byte) Contract {
	// run query
	contracts, err := bucket.Query(db, "", id)
	require.NoError(t, err)
	require.Len(t, contracts, 1)

	actual, err := bucket.Parse(nil, contracts[0].Value)
	require.NoError(t, err)

	return *actual.Value().(*Contract)
}

func withContract(t *testing.T, db weave.KVStore, msg CreateContractMsg) []byte {
	k := weavetest.NewCondition()
	ctx, auth := newContextWithAuth(k)
	handler := CreateContractMsgHandler{auth, NewContractBucket()}
	res, err := handler.Deliver(
		ctx,
		db,
		&weavetest.Tx{Msg: &msg})
	require.NoError(t, err)
	return res.Data
}

func TestCreateContractMsgHandler(t *testing.T) {
	a := weavetest.NewCondition()
	b := weavetest.NewCondition()
	c := weavetest.NewCondition()

	testcases := []struct {
		name string
		msg  *CreateContractMsg
		err  error
	}{
		{
			name: "valid use case",
			msg: &CreateContractMsg{
				Sigs:                newSigs(a, b, c),
				ActivationThreshold: 2,
				AdminThreshold:      3,
			},
			err: nil,
		},
		{
			name: "missing sigs",
			msg:  &CreateContractMsg{},
			err:  errors.Wrap(errors.ErrInvalidMsg, "missing sigs"),
		},
		{
			name: "bad activation threshold",
			msg: &CreateContractMsg{
				Sigs:                newSigs(a, b, c),
				ActivationThreshold: 4,
				AdminThreshold:      3,
			},
			err: errors.Wrap(errors.ErrInvalidMsg, invalidThreshold),
		},
		{
			name: "bad admin threshold",
			msg: &CreateContractMsg{
				Sigs:                newSigs(a, b, c),
				ActivationThreshold: 1,
				AdminThreshold:      -1,
			},
			err: errors.Wrap(errors.ErrInvalidMsg, invalidThreshold),
		},
		{
			name: "0 activation threshold",
			msg: &CreateContractMsg{
				Sigs:                newSigs(a, b, c),
				ActivationThreshold: 0,
				AdminThreshold:      1,
			},
			err: errors.Wrap(errors.ErrInvalidMsg, invalidThreshold),
		},
	}

	db := store.MemStore()
	for _, test := range testcases {
		msg := test.msg
		ctx, auth := newContextWithAuth(a)
		handler := CreateContractMsgHandler{auth, NewContractBucket()}

		_, err := handler.Check(ctx, db, &weavetest.Tx{Msg: msg})
		if test.err == nil {
			require.NoError(t, err, test.name)
		} else {
			require.EqualError(t, err, test.err.Error(), test.name)
		}

		res, err := handler.Deliver(ctx, db, &weavetest.Tx{Msg: msg})
		if test.err == nil {
			require.NoError(t, err, test.name)
			contract := queryContract(t, db, handler.bucket, res.Data)
			require.EqualValues(t,
				Contract{Sigs: msg.Sigs, ActivationThreshold: msg.ActivationThreshold, AdminThreshold: msg.AdminThreshold},
				contract,
				test.name)
		} else {
			require.EqualError(t, err, test.err.Error(), test.name)
		}

	}
}

func TestUpdateContractMsgHandler(t *testing.T) {
	db := store.MemStore()

	// addresses controlling contract
	a := weavetest.NewCondition()
	b := weavetest.NewCondition()
	c := weavetest.NewCondition()
	d := weavetest.NewCondition()
	e := weavetest.NewCondition()

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
		err     *errors.Error
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
			err:     errors.ErrUnauthorized,
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
			err:     errors.ErrUnauthorized,
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
			err:     errors.ErrInvalidMsg,
		},
	}

	for _, test := range testcases {
		msg := test.msg
		ctx, auth := newContextWithAuth(test.signers...)
		handler := UpdateContractMsgHandler{auth, NewContractBucket()}

		_, err := handler.Check(ctx, db, &weavetest.Tx{Msg: msg})
		if test.err == nil {
			require.NoError(t, err, test.name)
		} else {
			require.True(t, test.err.Is(err), test.name)
		}

		_, err = handler.Deliver(ctx, db, &weavetest.Tx{Msg: msg})
		if test.err == nil {
			require.NoError(t, err, test.name)
			contract := queryContract(t, db, handler.bucket, msg.Id)
			require.EqualValues(t,
				Contract{Sigs: msg.Sigs, ActivationThreshold: msg.ActivationThreshold, AdminThreshold: msg.AdminThreshold},
				contract,
				test.name)
		} else {
			require.True(t, test.err.Is(err), test.name)
		}
	}
}
