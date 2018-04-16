package escrow

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/app"
	"github.com/confio/weave/orm"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// specific helpers for this test

const authKey = "auth"

type action struct {
	perms  []weave.Permission
	msg    weave.Msg
	height int64 // block height, for timeout
}

func (a action) tx() weave.Tx {
	var helpers x.TestHelpers
	return helpers.MockTx(a.msg)
}

func (a action) ctx() weave.Context {
	ctx := context.Background()
	ctx = weave.WithHeight(ctx, a.height)
	return authenticator().SetPermissions(ctx, a.perms...)
}

// authenticator returns a default for all tests...
// clean this up?
func authenticator() x.CtxAuther {
	return x.TestHelpers{}.CtxAuth(authKey)
}

// how to do a query... TODO: abstract this??

type query struct {
	path     string
	mod      string
	data     []byte
	isError  bool
	expected []orm.Object
	bucket   orm.Bucket
}

func (q query) check(t *testing.T, db weave.ReadOnlyKVStore,
	qr weave.QueryRouter, msg ...interface{}) {

	h := qr.Handler(q.path)
	require.NotNil(t, h)
	mods, err := h.Query(db, q.mod, q.data)
	if q.isError {
		require.Error(t, err)
		return
	}
	require.NoError(t, err)
	if assert.Equal(t, len(q.expected), len(mods), msg...) {
		for i, ex := range q.expected {
			// make sure keys match
			key := q.bucket.DBKey(ex.Key())
			assert.Equal(t, key, mods[i].Key)

			// parse out value
			got, err := q.bucket.Parse(nil, mods[i].Value)
			require.NoError(t, err)
			assert.EqualValues(t, ex.Value(), got.Value(), msg...)
		}
	}
}

// for test, panics if cannot convert to model....
func objToModel(obj orm.Object) (weave.Model, error) {
	// ugh, we need the full on length...
	key := obj.Key()
	val := obj.Value()
	// this is soo ugly....
	if _, ok := val.(*Escrow); ok {
		key = NewBucket().DBKey(key)
	} else if _, ok := val.(*cash.Set); ok {
		key = cash.NewBucket().DBKey(key)
	}
	bz, err := val.Marshal()
	return weave.Model{key, bz}, err
}

func mo(obj orm.Object, err error) orm.Object {
	if err != nil {
		panic(err)
	}
	return obj
}

// TestHandler runs a number of scenario of tx to make
// sure they work as expected.
//
// I really should get quickcheck working....
func TestHandler(t *testing.T) {
	var helpers x.TestHelpers

	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()
	_, c := helpers.MakeKey()

	// good
	all := mustCombineCoins(x.NewCoin(100, 0, "FOO"))
	some := mustCombineCoins(x.NewCoin(32, 0, "FOO"))
	// TODO: add coins.Negative...
	minus := some.Clone()
	for _, m := range minus {
		m.Whole *= -1
		m.Fractional *= -1
	}
	remain, err := all.Combine(minus)
	require.NoError(t, err)

	id := func(i int64) []byte {
		bz := make([]byte, 8)
		binary.BigEndian.PutUint64(bz, uint64(i))
		return bz
	}
	eaddr := func(i int64) weave.Address {
		return Permission(id(i)).Address()
	}

	cases := []struct {
		// initial balance to set
		account weave.Address
		balance []*x.Coin
		// preparation transactions, must all succeed
		prep []action
		// tx to test
		do action
		// check if do should return an error
		isError bool
		// otherwise, a series of queries...
		queries []query
	}{
		// simplest test, sending money we have creates an escrow
		0: {
			a.Address(),
			all,
			nil, // no prep, just one action
			action{
				perms: []weave.Permission{a},
				msg: &CreateEscrowMsg{
					Sender:    a,
					Arbiter:   b,
					Recipient: c,
					Amount:    all,
					Timeout:   12345,
				},
				height: 1000,
			},
			false,
			[]query{
				// verify escrow is stored
				{
					"/escrows", "", id(1), false,
					[]orm.Object{
						NewEscrow(id(1), a, b, c, all, 12345, ""),
					},
					NewBucket().Bucket,
				},
				// cash deducted from sender
				{"/wallets", "", a.Address(), false,
					[]orm.Object{
						cash.NewWallet(a.Address()),
					},
					cash.NewBucket().Bucket,
				},
				// and added to escrow
				{"/wallets", "", eaddr(1), false,
					[]orm.Object{
						mo(cash.WalletWith(eaddr(1), all...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		// partial send, default sender taken from permissions
		1: {
			a.Address(),
			all,
			nil, // no prep, just one action
			action{
				perms: []weave.Permission{a},
				msg: &CreateEscrowMsg{
					// defaults to sender!
					Arbiter:   b,
					Recipient: c,
					Amount:    some,
					Timeout:   777,
				},
				height: 123,
			},
			false,
			[]query{
				// verify escrow is stored
				{
					"/escrows", "", id(1), false,
					[]orm.Object{
						NewEscrow(id(1), a, b, c, some, 777, ""),
					},
					NewBucket().Bucket,
				},
				// others id are empty
				{
					"/escrows", "", id(2), false, nil, orm.Bucket{},
				},
				// cash deducted from sender
				{"/wallets", "", a.Address(), false,
					[]orm.Object{
						mo(cash.WalletWith(a.Address(), remain...)),
					},
					cash.NewBucket().Bucket,
				},
				// and added to escrow
				{"/wallets", "", eaddr(1), false,
					[]orm.Object{
						mo(cash.WalletWith(eaddr(1), some...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		// cannot send money we don't have
		2: {
			a.Address(),
			some,
			nil, // no prep, just one action
			action{
				perms: []weave.Permission{a},
				msg: &CreateEscrowMsg{
					// defaults to sender!
					Arbiter:   b,
					Recipient: c,
					Amount:    all,
					Timeout:   12345,
				},
				height: 123,
			},
			true,
			nil,
		},
		// cannot send money from other account
		3: {
			a.Address(),
			all,
			nil, // no prep, just one action
			action{
				perms: []weave.Permission{b},
				msg: &CreateEscrowMsg{
					Sender:    a,
					Arbiter:   b,
					Recipient: c,
					Amount:    some,
					Timeout:   12345,
				},
				height: 123,
			},
			true,
			nil,
		},
		// cannot set timeout in the past
		4: {
			a.Address(),
			all,
			nil, // no prep, just one action
			action{
				perms: []weave.Permission{a},
				msg: &CreateEscrowMsg{
					// defaults to sender!
					Arbiter:   b,
					Recipient: c,
					Amount:    all,
					Timeout:   123,
				},
				height: 888,
			},
			true,
			nil,
		},
		// recipient cannot release
		5: {
			a.Address(),
			all,
			[]action{{
				perms: []weave.Permission{a},
				msg: &CreateEscrowMsg{
					Sender:    a,
					Arbiter:   b,
					Recipient: c,
					Amount:    all,
					Timeout:   12345,
				},
				height: 1000,
			}},
			action{
				perms: []weave.Permission{c},
				msg: &ReleaseEscrowMsg{
					EscrowId: id(1),
				},
				height: 2000,
			},
			true,
			nil,
		},
		// arbiter can successfully release all
		6: {
			a.Address(),
			all,
			[]action{{
				perms: []weave.Permission{a},
				msg: &CreateEscrowMsg{
					Sender:    a,
					Arbiter:   b,
					Recipient: c,
					Amount:    all,
					Timeout:   12345,
				},
				height: 1000,
			}},
			action{
				perms: []weave.Permission{b},
				msg: &ReleaseEscrowMsg{
					EscrowId: id(1),
				},
				height: 2000,
			},
			false,
			[]query{
				// verify escrow is deleted
				{
					"/escrows", "", id(1), false, nil, orm.Bucket{},
				},
				// escrow is empty
				{"/wallets", "", eaddr(1), false,
					[]orm.Object{
						cash.NewWallet(eaddr(1)),
					},
					cash.NewBucket().Bucket,
				},
				// sender is broke
				{"/wallets", "", a.Address(), false,
					[]orm.Object{
						cash.NewWallet(a.Address()),
					},
					cash.NewBucket().Bucket,
				},
				// recipient has cash
				{"/wallets", "", c.Address(), false,
					[]orm.Object{
						mo(cash.WalletWith(c.Address(), all...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		// arbiter can successfully release part
		7: {
			a.Address(),
			all,
			[]action{{
				perms: []weave.Permission{a},
				msg: &CreateEscrowMsg{
					Sender:    a,
					Arbiter:   b,
					Recipient: c,
					Amount:    all,
					Timeout:   12345,
					Memo:      "hello",
				},
				height: 1000,
			}},
			action{
				perms: []weave.Permission{b},
				msg: &ReleaseEscrowMsg{
					EscrowId: id(1),
					Amount:   some,
				},
				height: 2000,
			},
			false,
			[]query{
				// verify escrow balance is updated
				{
					"/escrows", "", id(1), false,
					[]orm.Object{
						NewEscrow(id(1), a, b, c, remain, 12345, "hello"),
					},
					NewBucket().Bucket,
				},
				// escrow is reduced
				{"/wallets", "", eaddr(1), false,
					[]orm.Object{
						mo(cash.WalletWith(eaddr(1), remain...)),
					},
					cash.NewBucket().Bucket,
				},
				// sender is broke
				{"/wallets", "", a.Address(), false,
					[]orm.Object{
						cash.NewWallet(a.Address()),
					},
					cash.NewBucket().Bucket,
				},
				// recipient has some money
				{"/wallets", "", c.Address(), false,
					[]orm.Object{
						mo(cash.WalletWith(c.Address(), some...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		// cannot release after timeout
		8: {
			a.Address(),
			all,
			[]action{{
				perms: []weave.Permission{a},
				msg: &CreateEscrowMsg{
					Sender:    a,
					Arbiter:   b,
					Recipient: c,
					Amount:    all,
					Timeout:   1234,
				},
				height: 1000,
			}},
			action{
				perms: []weave.Permission{b},
				msg: &ReleaseEscrowMsg{
					EscrowId: id(1),
				},
				height: 2000,
			},
			true,
			nil,
		},
	}

	bank := cash.NewBucket()
	ctrl := cash.NewController(bank)
	auth := authenticator()
	// create handler objects and query objects
	h := app.NewRouter()
	RegisterRoutes(h, auth, ctrl)
	qr := weave.NewQueryRouter()
	cash.RegisterQuery(qr)
	RegisterQuery(qr)

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			db := store.MemStore()

			// set initial data
			acct, err := cash.WalletWith(tc.account, tc.balance...)
			require.NoError(t, err)
			err = bank.Save(db, acct)
			require.NoError(t, err)

			// try checktx...
			cache := db.CacheWrap()
			for j, p := range tc.prep {
				_, err = h.Check(p.ctx(), cache, p.tx())
				require.NoError(t, err, "%d", j)
			}
			cache.Discard()

			// do delivertx
			for j, p := range tc.prep {
				_, err = h.Deliver(p.ctx(), db, p.tx())
				require.NoError(t, err, "%d", j)
			}
			_, err = h.Deliver(tc.do.ctx(), db, tc.do.tx())
			if tc.isError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// run through all queries
			for k, q := range tc.queries {
				q.check(t, db, qr, "%d", k)
			}
		})
	}
}
