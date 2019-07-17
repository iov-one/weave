package escrow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
)

var (
	blockNow = time.Now().UTC()
	Timeout  = weave.AsUnixTime(blockNow.Add(2 * time.Hour))

	zeroBucket = orm.NewBucket("zero", nil)
)

// rawBucket returns a raw escrow bucket. This exist for the legacy setup of
// the tests that use bucket Parse and DBKey method not provided by
// ModelBucket. This bucket must not be used outside of tests as it does not
// provide indexes or migrations. It can be used only to access the data.
func rawBucket() orm.Bucket {
	return orm.NewBucket("esc", orm.NewSimpleObj(nil, &Escrow{}))
}

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
		err error

		// otherwise, a series of queries...
		queries []query
	}{
		"simplest test, sending money we have creates an escrow": {
			a.Address(),
			all,
			nil, // no prep, just one action
			createAction(a, b, c, all, ""),
			nil,
			[]query{
				// verify escrow is stored
				{
					"/escrows", "", weavetest.SequenceID(1),
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), all, Timeout, ""),
					},
					rawBucket(),
				},
				// bank deducted from source
				{"/wallets", "", a.Address(),
					[]orm.Object{
						cash.NewWallet(a.Address()),
					},
					cash.NewBucket().Bucket,
				},
				// and added to escrow
				{"/wallets", "", escrowAddr(1),
					[]orm.Object{
						mo(cash.WalletWith(escrowAddr(1), all...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		"partial send, default source taken from permissions": {
			a.Address(),
			all,
			nil, // no prep, just one action
			createAction(a, b, c, some, ""),
			nil,
			[]query{
				// verify escrow is stored
				{
					"/escrows", "", weavetest.SequenceID(1),
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), some, Timeout, ""),
					},
					rawBucket(),
				},
				// make sure source index works
				{
					"/escrows/source", "", a.Address(),
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), some, Timeout, ""),
					},
					rawBucket(),
				},
				// make sure destination index works
				{
					"/escrows/destination", "", b.Address(),
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), some, Timeout, ""),
					},
					rawBucket(),
				},
				// make sure arbiter index works
				{
					"/escrows/arbiter", "", c.Address(),
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), some, Timeout, ""),
					},
					rawBucket(),
				},
				// make sure wrong query misses
				{
					"/escrows/arbiter", "", b, nil, rawBucket(),
				},
				// others id are empty
				{
					"/escrows", "", weavetest.SequenceID(2), nil, zeroBucket,
				},
				// bank deducted from source
				{"/wallets", "", a.Address(),
					[]orm.Object{
						mo(cash.WalletWith(a.Address(), remain...)),
					},
					cash.NewBucket().Bucket,
				},
				// and added to escrow
				{"/wallets", "", escrowAddr(1),
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
			errors.ErrAmount,
			nil,
		},
		"cannot send money from other account": {
			a.Address(),
			all,
			nil, // no prep, just one action
			action{
				// note permission is not the source!
				perms: []weave.Condition{b},
				msg:   NewCreateMsg(a.Address(), b.Address(), c.Address(), some, Timeout, ""),
			},
			errors.ErrUnauthorized,
			nil,
		},
		"cannot set timeout in the past": {
			a.Address(),
			all,
			nil, // no prep, just one action
			action{
				perms: []weave.Condition{a},
				// defaults to source!
				msg:       NewCreateMsg(nil, b.Address(), c.Address(), all, weave.AsUnixTime(blockNow.Add(-2*time.Hour)), ""),
				blockTime: Timeout.Time().Add(-time.Hour),
			},
			errors.ErrInput,
			nil,
		},
		"arbiter can successfully release all": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, all, "")},
			action{
				perms: []weave.Condition{c},
				msg: &ReleaseMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			nil,
			[]query{
				// verify escrow is deleted
				{
					"/escrows", "", weavetest.SequenceID(1), nil, zeroBucket,
				},
				// escrow is empty
				{"/wallets", "", escrowAddr(1),
					[]orm.Object{
						cash.NewWallet(escrowAddr(1)),
					},
					cash.NewBucket().Bucket,
				},
				// source is broke
				{"/wallets", "", a.Address(),
					[]orm.Object{
						cash.NewWallet(a.Address()),
					},
					cash.NewBucket().Bucket,
				},
				// destination has bank
				{"/wallets", "", b.Address(),
					[]orm.Object{
						mo(cash.WalletWith(b.Address(), all...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		"source can successfully release part": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, all, "hello")},
			action{
				perms: []weave.Condition{a},
				msg: &ReleaseMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
					Amount:   some,
				},
			},
			nil,
			[]query{
				// verify escrow balance is updated
				{
					"/escrows", "", weavetest.SequenceID(1),
					[]orm.Object{
						NewEscrow(weavetest.SequenceID(1), a.Address(), b.Address(), c.Address(), remain, Timeout, "hello"),
					},
					rawBucket(),
				},
				// escrow is reduced
				{"/wallets", "", escrowAddr(1),
					[]orm.Object{
						mo(cash.WalletWith(escrowAddr(1), remain...)),
					},
					cash.NewBucket().Bucket,
				},
				// source is broke
				{"/wallets", "", a.Address(),
					[]orm.Object{
						cash.NewWallet(a.Address()),
					},
					cash.NewBucket().Bucket,
				},
				// destination has some money
				{"/wallets", "", b.Address(),
					[]orm.Object{
						mo(cash.WalletWith(b.Address(), some...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		"destination cannot release": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, all, "")},
			action{
				perms: []weave.Condition{b},
				msg: &ReleaseMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			errors.ErrUnauthorized,
			nil,
		},
		"cannot release after timeout": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, all, "")},
			action{
				perms: []weave.Condition{c},
				msg: &ReleaseMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
				blockTime: Timeout.Time().Add(time.Hour),
			},
			errors.ErrExpired,
			nil,
		},
		"we update the arbiter and then make sure the new actors are used": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, some, ""),
				{
					perms: []weave.Condition{c},
					// c hands off to d
					msg: &UpdatePartiesMsg{
						Metadata: &weave.Metadata{Schema: 1},
						EscrowId: weavetest.SequenceID(1),
						Arbiter:  d.Address(),
					},
				}},
			action{
				// new arbiter can resolve
				perms: []weave.Condition{d},
				msg: &ReleaseMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			nil,
			[]query{
				// verify escrow is deleted (resolved)
				{
					"/escrows", "", weavetest.SequenceID(1), nil, zeroBucket,
				},
				// bank deducted from source
				{"/wallets", "", a.Address(),
					[]orm.Object{
						mo(cash.WalletWith(a.Address(), remain...)),
					},
					cash.NewBucket().Bucket,
				},
				// and added to destination
				{"/wallets", "", b.Address(),
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
					msg: &UpdatePartiesMsg{
						Metadata: &weave.Metadata{Schema: 1},
						EscrowId: weavetest.SequenceID(1),
						Arbiter:  d.Address(),
					},
				}},
			action{
				// original arbiter can no longer resolve
				perms: []weave.Condition{c},
				msg: &ReleaseMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			errors.ErrUnauthorized,
			nil,
		},
		"cannot update without proper permissions": {
			a.Address(),
			all,
			[]action{createAction(a, b, c, some, "")},
			action{
				perms: []weave.Condition{a},
				msg: &UpdatePartiesMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
					Arbiter:  a.Address(),
				},
			},
			errors.ErrUnauthorized,
			nil,
		},
		"cannot claim escrow twice": {
			a.Address(),
			all,
			[]action{
				createAction(a, b, c, all, ""),
				{
					perms: []weave.Condition{c},
					msg: &ReleaseMsg{
						Metadata: &weave.Metadata{Schema: 1},
						EscrowId: weavetest.SequenceID(1),
					},
				},
			},
			action{
				perms: []weave.Condition{c},
				msg: &ReleaseMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			errors.ErrNotFound,
			[]query{
				// verify escrow is deleted
				{
					"/escrows", "", weavetest.SequenceID(1), nil, zeroBucket,
				},
				// escrow is empty
				{"/wallets", "", escrowAddr(1),
					[]orm.Object{
						cash.NewWallet(escrowAddr(1)),
					},
					cash.NewBucket().Bucket,
				},
				// source is broke
				{"/wallets", "", a.Address(),
					[]orm.Object{
						cash.NewWallet(a.Address()),
					},
					cash.NewBucket().Bucket,
				},
				// destination has cash
				{"/wallets", "", b.Address(),
					[]orm.Object{
						mo(cash.WalletWith(b.Address(), all...)),
					},
					cash.NewBucket().Bucket,
				},
			},
		},
		"release overpaid amount and delete escrow": {
			a.Address(),
			mustCombineCoins(coin.NewCoin(2, 0, "FOO")),
			[]action{
				createAction(a, b, c, mustCombineCoins(coin.NewCoin(1, 0, "FOO")), ""),
				{
					perms: []weave.Condition{a},
					msg: &cash.SendMsg{
						Metadata:    &weave.Metadata{Schema: 1},
						Source:      a.Address(),
						Destination: escrowAddr(1),
						Amount:      &coin.Coin{Whole: 1, Ticker: "FOO"},
					},
				},
			},
			action{
				perms: []weave.Condition{c},
				msg: &ReleaseMsg{
					Metadata: &weave.Metadata{Schema: 1},
					EscrowId: weavetest.SequenceID(1),
				},
			},
			nil,
			[]query{
				// verify escrow is deleted
				{
					"/escrows", "", weavetest.SequenceID(1), nil, zeroBucket,
				},
				// escrow is empty
				{"/wallets", "", escrowAddr(1),
					[]orm.Object{
						cash.NewWallet(escrowAddr(1)),
					},
					cash.NewBucket().Bucket,
				},
				// source is broke
				{"/wallets", "", a.Address(),
					[]orm.Object{
						cash.NewWallet(a.Address()),
					},
					cash.NewBucket().Bucket,
				},
				// destination has bank
				{"/wallets", "", b.Address(),
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
			assert.Nil(t, err)
			err = bank.Save(db, acct)
			assert.Nil(t, err)

			// do delivertx
			for j, p := range tc.prep {
				t.Run(fmt.Sprintf("case %d", j), func(t *testing.T) {
					// try check
					cache := db.CacheWrap()
					_, err = router.Check(p.ctx(), cache, p.tx())
					assert.Nil(t, err)
					cache.Discard()

					// then perform
					_, err = router.Deliver(p.ctx(), db, p.tx())
					assert.Nil(t, err)
				})

			}

			_, err = router.Deliver(tc.do.ctx(), db, tc.do.tx())
			assert.IsErr(t, tc.err, err)

			// run through all queries
			for k, q := range tc.queries {
				t.Run(fmt.Sprintf("query-%d", k), func(t *testing.T) {
					q.check(t, db, qr)
				})
			}
		})
	}
}

func createAction(source, rcpt, arbiter weave.Condition, amount coin.Coins, memo string) action {
	return action{
		perms: []weave.Condition{source},
		msg:   NewCreateMsg(source.Address(), rcpt.Address(), arbiter.Address(), amount, Timeout, memo),
	}
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
	expected []orm.Object
	bucket   orm.Bucket
}

func (q query) check(t testing.TB, db weave.ReadOnlyKVStore, qr weave.QueryRouter, msg ...interface{}) {
	t.Helper()

	h := qr.Handler(q.path)
	if h == nil {
		t.Fatalf("Handler is nil for path %s", q.path)
	}
	mods, err := h.Query(db, q.mod, q.data)

	assert.Nil(t, err)
	assert.Equal(t, len(q.expected), len(mods))

	for i, ex := range q.expected {
		// make sure keys match
		key := q.bucket.DBKey(ex.Key())
		assert.Equal(t, key, mods[i].Key)

		// parse out value
		got, err := q.bucket.Parse(nil, mods[i].Value)
		assert.Nil(t, err)
		assert.Equal(t, ex.Value(), got.Value())
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
