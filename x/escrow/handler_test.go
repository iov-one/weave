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
	return Authenticator().SetPermissions(ctx, a.perms)
}

// Authenticator returns a default for all tests...
// clean this up?
func Authenticator() ctxAuther {
	return ctxAuther{authKey}
}

// how to do a query... TODO: abstract this??

type query struct {
	path     string
	mod      string
	data     []byte
	isError  bool
	expected []orm.Object
}

func (q query) check(t *testing.T, db weave.ReadOnlyKVStore,
	qr weave.QueryRouter) {

	h := qr.Handler(q.path)
	require.NotNil(t, h)
	mods, err := h.Query(db, q.mod, q.data)
	if q.isError {
		require.Error(t, err)
		return
	}
	require.NoError(t, err)
	if assert.Equal(t, len(q.expected), len(mods)) {
		for i, ex := range q.expected {
			should, err := objToModel(ex)
			require.NoError(t, err)
			assert.Equal(t, should, mods[i])
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

//--- todo: move this to x/helpers.go

type ctxAuther struct {
	key interface{}
}

var _ x.Authenticator = ctxAuther{}

func (a ctxAuther) SetPermissions(ctx weave.Context, perms []weave.Permission) weave.Context {
	return context.WithValue(ctx, a.key, perms)
}

func (a ctxAuther) GetPermissions(ctx weave.Context) []weave.Permission {
	val, _ := ctx.Value(a.key).([]weave.Permission)
	return val
}

func (a ctxAuther) HasAddress(ctx weave.Context, addr weave.Address) bool {
	for _, s := range a.GetPermissions(ctx) {
		if addr.Equals(s.Address()) {
			return true
		}
	}
	return false
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
	plus := mustCombineCoins(x.NewCoin(100, 0, "FOO"))

	id := func(i int64) []byte {
		bz := make([]byte, 8)
		binary.BigEndian.PutUint64(bz, uint64(i))
		return bz
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
		// simplest test, sending money we have
		0: {
			a.Address(),
			plus,
			nil, // no prep, just one action
			action{
				perms: []weave.Permission{a},
				msg: &CreateEscrowMsg{
					Sender:    a,
					Arbiter:   b,
					Recipient: c,
					Amount:    plus,
					Timeout:   12345,
				},
				height: 1000,
			},
			false,
			// verify escrow is stored
			[]query{{
				"/escrow", "", id(1), false,
				[]orm.Object{
					NewEscrow(id(1), a, b, c, plus, 12345, ""),
				},
			}},
		},
	}

	bank := cash.NewBucket()
	ctrl := cash.NewController(bank)
	auth := Authenticator()
	// TODO: create handler objects
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
			for _, q := range tc.queries {
				q.check(t, db, qr)
			}
		})
	}
}
