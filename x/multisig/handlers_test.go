package multisig

import (
	"context"
	"github.com/iov-one/weave/errors"
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

func TestCreateContractMsgHandler(t *testing.T) {
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
				Sigs:                newSigs(a, b, c),
				ActivationThreshold: 2,
				AdminThreshold:      3,
			},
			err: nil,
		},
		{
			name: "missing sigs",
			msg:  &CreateContractMsg{},
			err:  errors.ErrInvalidMsg.New("missing sigs"),
		},
		{
			name: "bad activation threshold",
			msg: &CreateContractMsg{
				Sigs:                newSigs(a, b, c),
				ActivationThreshold: 4,
				AdminThreshold:      3,
			},
			err: errors.ErrInvalidMsg.New(invalidThreshold),
		},
		{
			name: "bad admin threshold",
			msg: &CreateContractMsg{
				Sigs:                newSigs(a, b, c),
				ActivationThreshold: 1,
				AdminThreshold:      -1,
			},
			err: errors.ErrInvalidMsg.New(invalidThreshold),
		},
		{
			name: "0 activation threshold",
			msg: &CreateContractMsg{
				Sigs:                newSigs(a, b, c),
				ActivationThreshold: 0,
				AdminThreshold:      1,
			},
			err: errors.ErrInvalidMsg.New(invalidThreshold),
		},
	}

	db := store.MemStore()
	for _, test := range testcases {
		msg := test.msg
		ctx, auth := newContextWithAuth(a)
		handler := CreateContractMsgHandler{auth, NewContractBucket()}

		_, err := handler.Check(ctx, db, newTx(msg))
		if test.err == nil {
			require.NoError(t, err, test.name)
		} else {
			require.EqualError(t, err, test.err.Error(), test.name)
		}

		res, err := handler.Deliver(ctx, db, newTx(msg))
		if test.err == nil {
			require.NoError(t, err, test.name)
			contract := queryContract(t, db, handler.bucket, res.Data)
			require.EqualValues(t,
				Contract{msg.Sigs, msg.ActivationThreshold, msg.AdminThreshold},
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
			err:     errors.ErrUnauthorized.Newf("contract=%X", mutableID),
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
			err:     errors.ErrUnauthorized.Newf("contract=%X", immutableID),
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
			err:     errors.ErrInvalidMsg.New(invalidThreshold),
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
			require.True(t, errors.Is(err, test.err), test.name)
		}

		_, err = handler.Deliver(ctx, db, newTx(msg))
		if test.err == nil {
			require.NoError(t, err, test.name)
			contract := queryContract(t, db, handler.bucket, msg.Id)
			require.EqualValues(t,
				Contract{msg.Sigs, msg.ActivationThreshold, msg.AdminThreshold},
				contract,
				test.name)
		} else {
			require.EqualError(t, err, test.err.Error(), test.name)
		}
	}
}
