package feedist

import (
	"context"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
)

func TestHandlers(t *testing.T) {
	_, src := helper.MakeKey()

	addr1 := newAddress()
	addr2 := newAddress()

	rt := app.NewRouter()
	auth := helper.CtxAuth("auth")
	cashBucket := cash.NewBucket()
	ctrl := cash.NewController(cashBucket)
	RegisterRoutes(rt, auth, ctrl)

	// In below cases, asSeqID(1) is used - this is the address of the
	// first revenue instance created. Sequence is reset for each test
	// case.

	cases := map[string]struct {
		// prepareAccounts is used to set the funds for each declared
		// account, before executing actions.
		prepareAccounts []account
		// actions is a set of messages that will be handled by the
		// router. Successfully handled messages are altering the
		// state.
		actions []action
		// wantAccounts is used to declare desired state of each
		// account after all actions are applied.
		wantAccounts []account
	}{
		"at least one recipient is required": {
			prepareAccounts: nil,
			wantAccounts:    nil,
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &NewRevenueMsg{
						Admin:      []byte("f427d624ed29c1fae0e2"),
						Recipients: []*Recipient{},
					},
					blocksize:    100,
					wantCheckErr: errors.InvalidMsgErr,
				},
				{
					conditions: []weave.Condition{src},
					msg: &NewRevenueMsg{
						Admin: []byte("f427d624ed29c1fae0e2"),
						Recipients: []*Recipient{
							{Weight: 1, Address: addr1},
						},
					},
					blocksize: 101,
				},
				{
					conditions: []weave.Condition{src},
					msg: &ResetRevenueMsg{
						RevenueID:  asSeqID(1),
						Recipients: []*Recipient{},
					},
					blocksize:    102,
					wantCheckErr: errors.InvalidMsgErr,
				},
				{
					conditions: []weave.Condition{src},
					msg: &ResetRevenueMsg{
						RevenueID: asSeqID(1),
						Recipients: []*Recipient{
							{Weight: 1, Address: addr1},
							{Weight: 2, Address: addr2},
						},
					},
					blocksize: 104,
				},
			},
		},
		"revenue not found": {
			prepareAccounts: nil,
			wantAccounts:    nil,
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &DistributeMsg{
						RevenueID: []byte("revenue-with-this-id-does-not-exist"),
					},
					blocksize:      100,
					wantCheckErr:   errors.NotFoundErr,
					wantDeliverErr: errors.NotFoundErr,
				},
			},
		},
		"weights are normalized during distribution": {
			prepareAccounts: []account{
				{address: RevenueAccount(asSeqID(1)), coins: x.Coins{coinp(0, 7, "BTC")}},
			},
			wantAccounts: []account{
				// All funds must be transferred to the only recipient.
				{address: addr1, coins: x.Coins{coinp(0, 7, "BTC")}},
			},
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &NewRevenueMsg{
						Admin: []byte("f427d624ed29c1fae0e2"),
						Recipients: []*Recipient{
							// There is only one recipient with a ridiculously high weight.
							// All funds should be send to this account.
							{Weight: 1234, Address: addr1},
						},
					},
					blocksize:      100,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
				{
					conditions:     []weave.Condition{src},
					msg:            &DistributeMsg{RevenueID: asSeqID(1)},
					blocksize:      101,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
			},
		},
		"revenue without an account distributing funds": {
			prepareAccounts: nil,
			wantAccounts:    nil,
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &NewRevenueMsg{
						Admin: []byte("f427d624ed29c1fae0e2"),
						Recipients: []*Recipient{
							{Weight: 1, Address: addr1},
							{Weight: 2, Address: addr2},
						},
					},
					blocksize:      100,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
				{
					conditions: []weave.Condition{src},
					msg: &DistributeMsg{
						RevenueID: asSeqID(1),
					},
					blocksize:      101,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
			},
		},
		"revenue with an account but without enough funds": {
			prepareAccounts: []account{
				{address: RevenueAccount(asSeqID(1)), coins: x.Coins{coinp(0, 1, "BTC")}},
			},
			wantAccounts: []account{
				{address: RevenueAccount(asSeqID(1)), coins: x.Coins{coinp(0, 1, "BTC")}},
			},
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &NewRevenueMsg{
						Admin: []byte("f427d624ed29c1fae0e2"),
						Recipients: []*Recipient{
							{Weight: 1, Address: addr1},
							{Weight: 2, Address: addr2},
						},
					},
					blocksize:      100,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
				{
					conditions: []weave.Condition{src},
					msg: &DistributeMsg{
						RevenueID: asSeqID(1),
					},
					blocksize:      101,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
			},
		},
		"distribute revenue with a leftover funds": {
			prepareAccounts: []account{
				{address: RevenueAccount(asSeqID(1)), coins: x.Coins{coinp(0, 7, "BTC")}},
			},
			wantAccounts: []account{
				{address: RevenueAccount(asSeqID(1)), coins: x.Coins{coinp(0, 1, "BTC")}},
				{address: addr1, coins: x.Coins{coinp(0, 2, "BTC")}},
				{address: addr2, coins: x.Coins{coinp(0, 4, "BTC")}},
			},
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &NewRevenueMsg{
						Admin: []byte("f427d624ed29c1fae0e2"),
						Recipients: []*Recipient{
							{Weight: 10000, Address: addr1},
							{Weight: 20000, Address: addr2},
						},
					},
					blocksize:      100,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
				{
					conditions: []weave.Condition{src},
					msg: &DistributeMsg{
						RevenueID: asSeqID(1),
					},
					blocksize:      101,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
			},
		},
		"distribute revenue with an account holding various tickers": {
			prepareAccounts: []account{
				{address: RevenueAccount(asSeqID(1)), coins: x.Coins{coinp(0, 3, "BTC"), coinp(0, 7, "ETH")}},
			},
			wantAccounts: []account{
				{address: RevenueAccount(asSeqID(1)), coins: x.Coins{coinp(0, 1, "ETH")}},
				{address: addr1, coins: x.Coins{coinp(0, 1, "BTC"), coinp(0, 2, "ETH")}},
				{address: addr2, coins: x.Coins{coinp(0, 2, "BTC"), coinp(0, 4, "ETH")}},
			},
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &NewRevenueMsg{
						Admin: []byte("f427d624ed29c1fae0e2"),
						Recipients: []*Recipient{
							{Weight: 1, Address: addr1},
							{Weight: 2, Address: addr2},
						},
					},
					blocksize:      100,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
				{
					conditions: []weave.Condition{src},
					msg: &DistributeMsg{
						RevenueID: asSeqID(1),
					},
					blocksize:      101,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
			},
		},
		"updating a revenue is distributing the collected funds first": {
			prepareAccounts: []account{
				{address: RevenueAccount(asSeqID(1)), coins: x.Coins{coinp(0, 3, "BTC")}},
			},
			wantAccounts: []account{
				{address: addr1, coins: x.Coins{coinp(0, 1, "BTC")}},
				// Below is the state of the second account after ALL the actions applied.
				{address: addr2, coins: x.Coins{coinp(0, 2, "BTC")}},
			},
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &NewRevenueMsg{
						Admin: []byte("f427d624ed29c1fae0e2"),
						Recipients: []*Recipient{
							{Weight: 20, Address: addr1},
							{Weight: 20, Address: addr2},
						},
					},
					blocksize:      100,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
				// Issuing an update must distribute first.
				// Distributing 3 BTC cents equally, means that 1 BTC cent will be left.
				{
					conditions: []weave.Condition{src},
					msg: &ResetRevenueMsg{
						RevenueID: asSeqID(1),
						Recipients: []*Recipient{
							{Weight: 1234, Address: addr2},
						},
					},
					blocksize:      102,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
				// After the update, all funds (1 cent) should be moved to addr2
				{
					conditions: []weave.Condition{src},
					msg: &DistributeMsg{
						RevenueID: asSeqID(1),
					},
					blocksize:      103,
					wantCheckErr:   nil,
					wantDeliverErr: nil,
				},
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			for _, a := range tc.prepareAccounts {
				for _, c := range a.coins {
					if err := ctrl.IssueCoins(db, a.address, *c); err != nil {
						t.Fatalf("cannot issue %q to %x: %s", c, a.address, err)
					}
				}
			}

			for i, a := range tc.actions {
				cache := db.CacheWrap()
				if _, err := rt.Check(a.ctx(), cache, a.tx()); !errors.Is(err, a.wantCheckErr) {
					t.Logf("want: %+v", a.wantCheckErr)
					t.Logf(" got: %+v", err)
					t.Fatalf("action %d check (%T)", i, a.msg)
				}
				cache.Discard()
				if a.wantCheckErr != nil {
					// Failed checks are causing the message to be ignored.
					continue
				}

				if _, err := rt.Deliver(a.ctx(), db, a.tx()); !errors.Is(err, a.wantDeliverErr) {
					t.Logf("want: %+v", a.wantDeliverErr)
					t.Logf(" got: %+v", err)
					t.Fatalf("action %d delivery (%T)", i, a.msg)
				}
			}

			for i, a := range tc.wantAccounts {
				coins, err := ctrl.Balance(db, a.address)
				if err != nil {
					t.Fatalf("cannot get %+v balance: %s", a, err)
				}
				if !coins.Equals(a.coins) {
					t.Logf("want: %+v", a.coins)
					t.Logf("got: %+v", coins)
					t.Errorf("unexpected coins for account #%d (%s)", i, a.address)
				}
			}
		})
	}
}

// account represents a single account state - the coins/funds it holds.
type account struct {
	address weave.Address
	coins   x.Coins
}

// action represents a single request call that is handled by a handler.
type action struct {
	conditions     []weave.Condition
	msg            weave.Msg
	blocksize      int64
	wantCheckErr   error
	wantDeliverErr error
}

func (a *action) tx() weave.Tx {
	return helper.MockTx(a.msg)
}

func (a *action) ctx() weave.Context {
	ctx := weave.WithHeight(context.Background(), a.blocksize)
	ctx = weave.WithChainID(ctx, "testchain-123")
	return helper.CtxAuth("auth").SetConditions(ctx, a.conditions...)
}

var helper x.TestHelpers

func newAddress() weave.Address {
	_, key := helper.MakeKey()
	return key.Address()
}

// asSeqID returns an ID encoded as if it was generated by the bucket sequence
// call.
func asSeqID(i int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

func coinp(w, f int64, ticker string) *x.Coin {
	c := x.NewCoin(w, f, ticker)
	return &c
}

func TestFindGdc(t *testing.T) {
	cases := map[string]struct {
		want   int32
		values []int32
	}{
		"empty": {
			want:   0,
			values: nil,
		},
		"one element": {
			want:   7,
			values: []int32{7},
		},
		"two elements": {
			want:   3,
			values: []int32{9, 6},
		},
		"three elements": {
			want:   3,
			values: []int32{9, 3, 6},
		},
		"four elements": {
			want:   6,
			values: []int32{12, 6, 18},
		},
		"less common divisors": {
			want:   2,
			values: []int32{24, 12, 64, 18},
		},
		"prime numbers": {
			want:   1,
			values: []int32{67, 71, 73, 79, 83, 89, 97},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			got := findGcd(tc.values...)
			if got != tc.want {
				t.Fatalf("want %d, got %d", tc.want, got)
			}
		})
	}
}

func TestDistribute(t *testing.T) {
	cases := map[string]struct {
		recipients []*Recipient
		ctrl       *testController
		// Each MoveCoins call on the testController result in creation
		// of a movecall. Those can be used later to validate that
		// certain MoveCoins calls were made.
		wantMoves []movecall
		wantErr   error
	}{
		"zero funds is not distributed": {
			recipients: []*Recipient{
				{Address: weave.Address("address-1"), Weight: 1},
				{Address: weave.Address("address-2"), Weight: 2},
			},
			ctrl: &testController{
				balance: nil,
			},
			wantErr: nil,
		},
		"tiny funds are not distributed if cannot be split": {
			recipients: []*Recipient{
				{Address: weave.Address("address-1"), Weight: 1},
				{Address: weave.Address("address-2"), Weight: 2},
			},
			ctrl: &testController{
				balance: x.Coins{coinp(0, 1, "ETH")},
			},
			wantErr: nil,
		},
		"simple distribute case": {
			recipients: []*Recipient{
				{Address: weave.Address("address-1"), Weight: 1},
				{Address: weave.Address("address-2"), Weight: 2},
			},
			ctrl: &testController{
				balance: x.Coins{coinp(3, 0, "BTC")},
			},
			wantErr: nil,
			wantMoves: []movecall{
				{dst: weave.Address("address-1"), amount: x.NewCoin(1, 0, "BTC")},
				{dst: weave.Address("address-2"), amount: x.NewCoin(2, 0, "BTC")},
			},
		},
		"distribution splits whole into fractional": {
			recipients: []*Recipient{
				{Address: weave.Address("address-1"), Weight: 1},
				{Address: weave.Address("address-2"), Weight: 2},
			},
			ctrl: &testController{
				balance: x.Coins{coinp(1, 0, "BTC")},
			},
			wantErr: nil,
			wantMoves: []movecall{
				// One cent is left on the revenue account,
				// because it is too small to divide.
				{dst: weave.Address("address-1"), amount: x.NewCoin(0, x.FracUnit/3, "BTC")},
				{dst: weave.Address("address-2"), amount: x.NewCoin(0, x.FracUnit/3*2, "BTC")},
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			src := weave.Address("address-source")
			err := distribute(nil, tc.ctrl, src, tc.recipients)
			if !errors.Is(tc.wantErr, err) {
				t.Errorf("want %q error, got %q", tc.wantErr, err)
			}
			if !reflect.DeepEqual(tc.wantMoves, tc.ctrl.moves) {
				t.Logf("got %d MoveCoins calls", len(tc.ctrl.moves))
				for i, m := range tc.ctrl.moves {
					t.Logf("%d: %v", i, m)
				}
				t.Fatalf("unexpected MoveCoins calls")
			}
		})
	}
}

type testController struct {
	balance x.Coins
	err     error
	moves   []movecall
}

type movecall struct {
	dst    weave.Address
	amount x.Coin
}

func (tc *testController) Balance(weave.KVStore, weave.Address) (x.Coins, error) {
	return tc.balance, tc.err
}

func (tc *testController) MoveCoins(db weave.KVStore, src, dst weave.Address, amount x.Coin) error {
	tc.moves = append(tc.moves, movecall{dst: dst, amount: amount})
	return tc.err
}
