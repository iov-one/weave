package escrow

import (
	"context"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/cash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	blockNow = time.Now().UTC()
	Timeout  = weave.AsUnixTime(blockNow.Add(2 * time.Hour))

	zeroBucket = orm.NewBucket("zero", nil)
)

// TestHandler runs a number of scenario of tx to make
// sure they work as expected.
//
// I really should get quick-check working....
func TestHandler(t *testing.T) {

	a := weavetest.NewCondition()
	b := weavetest.NewCondition()
	c := weavetest.NewCondition()
	// d is just an observer, no role in escrow
	d := weavetest.NewCondition()

	// good
	all := mustCombineCoins(coin.NewCoin(100, 0, "FOO"),
		coin.NewCoin(10, 0, "BAR"))
	some := mustCombineCoins(coin.NewCoin(32, 0, "FOO"),
		coin.NewCoin(1, 0, "BAR"))
	remain := mustCombineCoins(coin.NewCoin(68, 0, "FOO"),
		coin.NewCoin(9, 0, "BAR"))

	escrowAddr := func(i uint64) weave.Address {
		return Condition(weavetest.SequenceID(i)).Address()
	}

	cases := map[string]struct {
		// initial balance to set
		account weave.Address
		balance []*coin.Coin
		// preparation transactions, must all succeed
		prep []action
		// tx to test
		do action
		// check if do should return an error
		isError bool
		// otherwise, a series of queries...
		queries []query
	}{
		"simplest test, sending money we have creates an escrow": {
			a.Address(),
			all,
			nil, // no prep, just one action
			createAction(a, b, c, all, ""),
			false,
			[]query{
				// verify escrow is stored
				{
					"/escrows", "", weavetest.SequenceID(1), false,
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), all, Timeout, ""),
					},
					NewBucket().Bucket,
				},
				// bank deducted from sender
				{"/wallets", "", a.Address(), false,
					[]orm.Object{
						cash.NewWallet(a.Address()),
					},
					cash.NewBucket().Bucket,
				},
				// and added to escrow
				{"/wallets", "", escrowAddr(1), false,
					[]orm.Object{
						mo(cash.WalletWith(escrowAddr(1), all...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		"partial send, default sender taken from permissions": {
			a.Address(),
			all,
			nil, // no prep, just one action
			createAction(a, b, c, some, ""),
			false,
			[]query{
				// verify escrow is stored
				{
					"/escrows", "", weavetest.SequenceID(1), false,
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), some, Timeout, ""),
					},
					NewBucket().Bucket,
				},
				// make sure sender index works
				{
					"/escrows/sender", "", a.Address(), false,
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), some, Timeout, ""),
					},
					NewBucket().Bucket,
				},
				// make sure recipient index works
				{
					"/escrows/recipient", "", b.Address(), false,
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), some, Timeout, ""),
					},
					NewBucket().Bucket,
				},
				// make sure arbiter index works
				{
					"/escrows/arbiter", "", c.Address(), false,
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), some, Timeout, ""),
					},
					NewBucket().Bucket,
				},
				// make sure wrong query misses
				{
					"/escrows/arbiter", "", b, false, nil, NewBucket().Bucket,
				},
				// others id are empty
				{
					"/escrows", "", weavetest.SequenceID(2), false, nil, zeroBucket,
				},
				// bank deducted from sender
				{"/wallets", "", a.Address(), false,
					[]orm.Object{
						mo(cash.WalletWith(a.Address(), remain...)),
					},
					cash.NewBucket().Bucket,
				},
				// and added to escrow
				{"/wallets", "", escrowAddr(1), false,
					[]orm.Object{
						mo(cash.WalletWith(escrowAddr(1), some...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		"cannot send money we don't have": {
			a.Address(),
			some,
			nil, // no prep, just one action
			createAction(a, b, c, all, ""),
			true,
			nil,
		},
		"cannot send money from other account": {
			a.Address(),
			all,
			nil, // no prep, just one action
			action{
				// note permission is not the sender!
				perms: []weave.Condition{b},
				msg:   NewCreateMsg(a.Address(), b.Address(), c.Address(), some, Timeout, ""),
			},
			true,
			nil,
		},
		"cannot set timeout in the past": {
			a.Address(),
			all,
			nil, // no prep, just one action
			action{
				perms: []weave.Condition{a},
				// defaults to sender!
				msg:       NewCreateMsg(nil, b.Address(), c.Address(), all, weave.AsUnixTime(blockNow.Add(-2*time.Hour)), ""),
				blockTime: Timeout.Time().Add(-time.Hour),
			},
			true,
			nil,
		},
		"arbiter can successfully release all": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, all, "")},
			action{
				perms: []weave.Condition{c},
				msg: &ReleaseEscrowMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			false,
			[]query{
				// verify escrow is deleted
				{
					"/escrows", "", weavetest.SequenceID(1), false, nil, zeroBucket,
				},
				// escrow is empty
				{"/wallets", "", escrowAddr(1), false,
					[]orm.Object{
						cash.NewWallet(escrowAddr(1)),
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
				// recipient has bank
				{"/wallets", "", b.Address(), false,
					[]orm.Object{
						mo(cash.WalletWith(b.Address(), all...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		"sender can successfully release part": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, all, "hello")},
			action{
				perms: []weave.Condition{a},
				msg: &ReleaseEscrowMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
					Amount:   some,
				},
			},
			false,
			[]query{
				// verify escrow balance is updated
				{
					"/escrows", "", weavetest.SequenceID(1), false,
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), remain, Timeout, "hello"),
					},
					NewBucket().Bucket,
				},
				// escrow is reduced
				{"/wallets", "", escrowAddr(1), false,
					[]orm.Object{
						mo(cash.WalletWith(escrowAddr(1), remain...)),
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
				{"/wallets", "", b.Address(), false,
					[]orm.Object{
						mo(cash.WalletWith(b.Address(), some...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		"recipient cannot release": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, all, "")},
			action{
				perms: []weave.Condition{b},
				msg: &ReleaseEscrowMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			true,
			nil,
		},
		"cannot release after timeout": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, all, "")},
			action{
				perms: []weave.Condition{c},
				msg: &ReleaseEscrowMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
				blockTime: Timeout.Time().Add(time.Hour),
			},
			true,
			nil,
		},
		//"successful return after expired (can be done by anyone)": {
		//	a.Address(),
		//	all,
		//	[]action{createAction(a, b, c, all, "")},
		//	action{
		//		perms: []weave.Condition{a},
		//		msg: &ReturnEscrowMsg{
		//			EscrowId: weavetest.SequenceID(1),
		//		},
		//		height: Timeout + 1,
		//	},
		//	false,
		//	[]query{
		//		// verify escrow is deleted
		//		{
		//			"/escrows", "", weavetest.SequenceID(1), false, nil, zeroBucket,
		//		},
		//		// escrow is empty
		//		{"/wallets", "", escrowAddr(1), false,
		//			[]orm.Object{
		//				cash.NewWallet(escrowAddr(1)),
		//			},
		//			cash.NewBucket().Bucket,
		//		},
		//		// sender recover all his money
		//		{"/wallets", "", a.Address(), false,
		//			[]orm.Object{
		//				mo(cash.WalletWith(a.Address(), all...)),
		//			},
		//			cash.NewBucket().Bucket,
		//		},
		//		// recipient doesn't get paid
		//		{"/wallets", "", b.Address(), false, nil,
		//			cash.NewBucket().Bucket,
		//		},
		//	},
		//},
		//"cannot return before timeout": {
		//	a.Address(),
		//	all,
		//	[]action{createAction(a, b, c, all, "")},
		//	action{
		//		perms: []weave.Condition{a},
		//		msg: &ReturnEscrowMsg{
		//			EscrowId: weavetest.SequenceID(1),
		//		},
		//		height: Timeout - 1,
		//	},
		//	true,
		//	nil,
		//},
		"we update the arbiter and then make sure the new actors are used": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, some, ""),
				{
					perms: []weave.Condition{c},
					// c hands off to d
					msg: &UpdateEscrowPartiesMsg{
						Metadata: &weave.Metadata{Schema: 1},
						EscrowId: weavetest.SequenceID(1),
						Arbiter:  d.Address(),
					},
				}},
			action{
				// new arbiter can resolve
				perms: []weave.Condition{d},
				msg: &ReleaseEscrowMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			false,
			[]query{
				// verify escrow is deleted (resolved)
				{
					"/escrows", "", weavetest.SequenceID(1), false, nil, zeroBucket,
				},
				// bank deducted from sender
				{"/wallets", "", a.Address(), false,
					[]orm.Object{
						mo(cash.WalletWith(a.Address(), remain...)),
					},
					cash.NewBucket().Bucket,
				},
				// and added to recipient
				{"/wallets", "", b.Address(), false,
					[]orm.Object{
						mo(cash.WalletWith(b.Address(), some...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		"after update, original arbiter cannot resolve": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, some, ""),
				{
					perms: []weave.Condition{c},
					// c hands off to d
					msg: &UpdateEscrowPartiesMsg{
						Metadata: &weave.Metadata{Schema: 1},
						EscrowId: weavetest.SequenceID(1),
						Arbiter:  d.Address(),
					},
				}},
			action{
				// original arbiter can no longer resolve
				perms: []weave.Condition{c},
				msg: &ReleaseEscrowMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			true,
			nil,
		},
		"cannot update without proper permissions": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, some, "")},
			action{
				perms: []weave.Condition{a},
				msg: &UpdateEscrowPartiesMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
					Arbiter:  a.Address(),
				},
			},
			true,
			nil,
		},
		//"cannot update parties after timeout": {
		//	a.Address(),
		//	all,
		//	[]action{createAction(a, b, c, some, "")},
		//	action{
		//		perms: []weave.Condition{a},
		//		msg: &UpdateEscrowPartiesMsg{
		//			EscrowId: weavetest.SequenceID(1),
		//			Sender:   d,
		//		},
		//		height: Timeout + 100,
		//	},
		//	true,
		//	nil,
		//},
		"cannot claim escrow twice": {
			a.Address(),
			all,
			[]action{
				createAction(a, b, c, all, ""),
				{
					perms: []weave.Condition{c},
					msg: &ReleaseEscrowMsg{
						Metadata: &weave.Metadata{Schema: 1},
						EscrowId: weavetest.SequenceID(1),
					},
				},
			},
			action{
				perms: []weave.Condition{c},
				msg: &ReleaseEscrowMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			true,
			[]query{
				// verify escrow is deleted
				{
					"/escrows", "", weavetest.SequenceID(1), false, nil, zeroBucket,
				},
				// escrow is empty
				{"/wallets", "", escrowAddr(1), false,
					[]orm.Object{
						cash.NewWallet(escrowAddr(1)),
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
				{"/wallets", "", b.Address(), false,
					[]orm.Object{
						mo(cash.WalletWith(b.Address(), all...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		//"return overpaid amount and delete escrow": {
		//	a.Address(),
		//	mustCombineCoins(coin.NewCoin(2, 0, "FOO")),
		//	[]action{
		//		createAction(a, b, c, mustCombineCoins(coin.NewCoin(1, 0, "FOO")), ""),
		//		{
		//			perms: []weave.Condition{a},
		//			msg: &cash.SendMsg{
		//				Src:    a.Address(),
		//				Dest:   escrowAddr(1),
		//				Amount: &coin.Coin{Whole: 1, Ticker: "FOO"},
		//			},
		//		},
		//	},
		//	action{
		//		perms: []weave.Condition{a},
		//		msg: &ReturnEscrowMsg{
		//			EscrowId: weavetest.SequenceID(1),
		//		},
		//		height: Timeout + 1,
		//	},
		//	false,
		//	[]query{
		//		// verify escrow is deleted
		//		{
		//			"/escrows", "", weavetest.SequenceID(1), false, nil, zeroBucket,
		//		},
		//		// escrow is empty
		//		{"/wallets", "", escrowAddr(1), false,
		//			[]orm.Object{
		//				cash.NewWallet(escrowAddr(1)),
		//			},
		//			cash.NewBucket().Bucket,
		//		},
		//		// sender recover all his money
		//		{"/wallets", "", a.Address(), false,
		//			[]orm.Object{
		//				mo(cash.WalletWith(a.Address(), mustCombineCoins(coin.NewCoin(2, 0, "FOO"))...)),
		//			},
		//			cash.NewBucket().Bucket,
		//		},
		//		// recipient doesn't get paid
		//		{"/wallets", "", b.Address(), false, nil,
		//			cash.NewBucket().Bucket,
		//		},
		//	},
		//},
		"release overpaid amount and delete escrow": {
			a.Address(),
			mustCombineCoins(coin.NewCoin(2, 0, "FOO")),
			[]action{
				createAction(a, b, c, mustCombineCoins(coin.NewCoin(1, 0, "FOO")), ""),
				{
					perms: []weave.Condition{a},
					msg: &cash.SendMsg{
						Metadata: &weave.Metadata{Schema: 1},
						Src:      a.Address(),
						Dest:     escrowAddr(1),
						Amount:   &coin.Coin{Whole: 1, Ticker: "FOO"},
					},
				},
			},
			action{
				perms: []weave.Condition{c},
				msg: &ReleaseEscrowMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			false,
			[]query{
				// verify escrow is deleted
				{
					"/escrows", "", weavetest.SequenceID(1), false, nil, zeroBucket,
				},
				// escrow is empty
				{"/wallets", "", escrowAddr(1), false,
					[]orm.Object{
						cash.NewWallet(escrowAddr(1)),
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
				// recipient has bank
				{"/wallets", "", b.Address(), false,
					[]orm.Object{
						mo(cash.WalletWith(b.Address(), mustCombineCoins(coin.NewCoin(2, 0, "FOO"))...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
	}

	bank := cash.NewBucket()
	ctrl := cash.NewController(bank)
	auth := authenticator()
	// create handler objects and query objects
	router := app.NewRouter()
	RegisterRoutes(router, auth, ctrl)
	cash.RegisterRoutes(router, auth, ctrl)
	qr := weave.NewQueryRouter()
	cash.RegisterQuery(qr)
	RegisterQuery(qr)

	for descr, tc := range cases {
		t.Run(descr, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "escrow", "cash")

			// set initial data
			acct, err := cash.WalletWith(tc.account, tc.balance...)
			require.NoError(t, err)
			err = bank.Save(db, acct)
			require.NoError(t, err)

			// do delivertx
			for j, p := range tc.prep {
				// try check
				cache := db.CacheWrap()
				_, err = router.Check(p.ctx(), cache, p.tx())
				require.NoError(t, err, "%d", j)
				cache.Discard()

				// then perform
				_, err = router.Deliver(p.ctx(), db, p.tx())
				require.NoError(t, err, "%d", j)
			}

			_, err = router.Deliver(tc.do.ctx(), db, tc.do.tx())
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// run through all queries
			for k, q := range tc.queries {
				q.check(t, db, qr, "%d", k)
			}
		})
	}
}

func createAction(sender, rcpt, arbiter weave.Condition, amount coin.Coins, memo string) action {
	return action{
		perms: []weave.Condition{sender},
		msg:   NewCreateMsg(sender.Address(), rcpt.Address(), arbiter.Address(), amount, Timeout, memo),
	}
}

// MinusCoins returns a-b
func MinusCoins(a, b coin.Coins) (coin.Coins, error) {
	// TODO: add coins.Negative...
	minus := b.Clone()
	for _, m := range minus {
		m.Whole *= -1
		m.Fractional *= -1
	}
	return a.Combine(minus)
}

func MustMinusCoins(t *testing.T, a, b coin.Coins) coin.Coins {
	remain, err := MinusCoins(a, b)
	require.NoError(t, err)
	return remain
}

func MustAddCoins(t *testing.T, a, b coin.Coins) coin.Coins {
	res, err := a.Combine(b)
	require.NoError(t, err)
	return res
}

//-------------------------------------------------
// specific helpers for these tests

type action struct {
	perms []weave.Condition
	msg   weave.Msg

	// if not zero, overwrites blockTime function for timeout
	blockTime time.Time
}

func (a action) tx() weave.Tx {
	return &weavetest.Tx{Msg: a.msg}
}

func (a action) ctx() weave.Context {
	ctx := context.Background()
	if !a.blockTime.IsZero() {
		ctx = weave.WithBlockTime(ctx, a.blockTime)
	} else {
		ctx = weave.WithBlockTime(ctx, blockNow)
	}
	return authenticator().SetConditions(ctx, a.perms...)
}

// authenticator returns a default for all tests...
// clean this up?
func authenticator() *weavetest.CtxAuth {
	return &weavetest.CtxAuth{Key: "auth"}
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

func (q query) check(t testing.TB, db weave.ReadOnlyKVStore, qr weave.QueryRouter, msg ...interface{}) {
	t.Helper()

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
			key := orm.DBKey(q.bucket, ex.Key())
			assert.Equal(t, key, mods[i].Key)

			// parse out value
			got, err := orm.Parse(q.bucket, nil, mods[i].Value)
			require.NoError(t, err)
			assert.EqualValues(t, ex.Value(), got.Value(), msg...)
		}
	}
}

// mo = must object... takes (Object, error) result and
// convert to Object or panic
func mo(obj orm.Object, err error) orm.Object {
	if err != nil {
		panic(err)
	}
	return obj
}
