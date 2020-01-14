package termdeposit

import (
	"context"
	"testing"
	"time"

	weave "github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/cash"
)

func TestUseCases(t *testing.T) {
	type Request struct {
		Now         weave.UnixTime
		Conditions  []weave.Condition
		Tx          weave.Tx
		BlockHeight int64
		WantErr     *errors.Error
	}

	type AccountBalance struct {
		Wallet weave.Address
		Amount coin.Coin
	}

	var (
		adminCond   = weavetest.NewCondition()
		aliceCond   = weavetest.NewCondition()
		bobCond     = weavetest.NewCondition()
		charlieCond = weavetest.NewCondition()

		now = weave.UnixTime(1572247483)
	)

	cases := map[string]struct {
		Requests  []Request
		Funds     []AccountBalance
		AfterTest func(t *testing.T, db weave.KVStore)
		Bonuses   []DepositBonus
	}{
		"admin can create a contarct": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &CreateDepositContractMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							ValidSince: now.Add(time.Hour),
							ValidUntil: now.Add(2 * time.Hour),
						},
					},
					BlockHeight: 100,
					WantErr:     errors.ErrUnauthorized,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &CreateDepositContractMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							ValidSince: now.Add(time.Hour),
							ValidUntil: now.Add(2 * time.Hour),
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
			},
		},
		"depositor signature is required in order to create a deposit": {
			Funds: []AccountBalance{
				{Wallet: bobCond.Address(), Amount: coin.NewCoin(4, 0, "IOV")},
			},
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &CreateDepositContractMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							ValidSince: now,
							ValidUntil: now.Add(2 * time.Hour),
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{adminCond, aliceCond},
					Tx: &weavetest.Tx{
						Msg: &DepositMsg{
							Metadata:          &weave.Metadata{Schema: 1},
							DepositContractID: weavetest.SequenceID(1),
							Amount:            coin.NewCoin(1, 0, "IOV"),
							Depositor:         bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     errors.ErrUnauthorized,
				},
			},
		},
		"enough funds are required to create deposit": {
			Funds: []AccountBalance{
				{Wallet: bobCond.Address(), Amount: coin.NewCoin(4, 0, "IOV")},
			},
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &CreateDepositContractMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							ValidSince: now,
							ValidUntil: now.Add(2 * time.Hour),
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &DepositMsg{
							Metadata:          &weave.Metadata{Schema: 1},
							DepositContractID: weavetest.SequenceID(1),
							Amount:            coin.NewCoin(321, 0, "IOV"),
							Depositor:         bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     errors.ErrAmount,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertFunds(t, db, bobCond.Address(), coin.NewCoin(4, 0, "IOV"))
			},
		},
		"anyone with enough funds can create a deposit": {
			Funds: []AccountBalance{
				{Wallet: bobCond.Address(), Amount: coin.NewCoin(100, 0, "IOV")},
			},
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &CreateDepositContractMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							ValidSince: now,
							ValidUntil: now.Add(2 * time.Hour),
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &DepositMsg{
							Metadata:          &weave.Metadata{Schema: 1},
							DepositContractID: weavetest.SequenceID(1),
							Amount:            coin.NewCoin(1, 0, "IOV"),
							Depositor:         bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertFunds(t, db, bobCond.Address(), coin.NewCoin(99, 0, "IOV"))

				var d Deposit
				if err := NewDepositBucket().One(db, weavetest.SequenceID(2), &d); err != nil {
					t.Fatalf("cannot get deposit: %s", err)
				}
				if d.CreatedAt != now+1 {
					t.Fatalf("invalid created at time: %d != %d", d.CreatedAt, now+1)
				}
			},
		},
		"deposit cannot be created for a contract that is not yet active": {
			Funds: []AccountBalance{
				{Wallet: bobCond.Address(), Amount: coin.NewCoin(100, 0, "IOV")},
			},
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &CreateDepositContractMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							ValidSince: now.Add(time.Hour),
							ValidUntil: now.Add(2 * time.Hour),
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &DepositMsg{
							Metadata:          &weave.Metadata{Schema: 1},
							DepositContractID: weavetest.SequenceID(1),
							Amount:            coin.NewCoin(1, 0, "IOV"),
							Depositor:         bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     errors.ErrState,
				},
			},
		},
		"deposit cannot be created for an expired contract": {
			Funds: []AccountBalance{
				{Wallet: bobCond.Address(), Amount: coin.NewCoin(100, 0, "IOV")},
			},
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &CreateDepositContractMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							ValidSince: now,
							ValidUntil: now.Add(time.Minute),
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 10000,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &DepositMsg{
							Metadata:          &weave.Metadata{Schema: 1},
							DepositContractID: weavetest.SequenceID(1),
							Amount:            coin.NewCoin(1, 0, "IOV"),
							Depositor:         bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     errors.ErrExpired,
				},
			},
		},
		"non expired deposit cannot be released": {
			Funds: []AccountBalance{
				{Wallet: bobCond.Address(), Amount: coin.NewCoin(100, 0, "IOV")},
			},
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &CreateDepositContractMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							ValidSince: now,
							ValidUntil: now.Add(2 * time.Hour),
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &DepositMsg{
							Metadata:          &weave.Metadata{Schema: 1},
							DepositContractID: weavetest.SequenceID(1),
							Amount:            coin.NewCoin(1, 0, "IOV"),
							Depositor:         bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now: now + 2,
					Tx: &weavetest.Tx{
						Msg: &ReleaseDepositMsg{
							Metadata:  &weave.Metadata{Schema: 1},
							DepositID: weavetest.SequenceID(2),
						},
					},
					BlockHeight: 102,
					WantErr:     errors.ErrState,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertFunds(t, db, bobCond.Address(), coin.NewCoin(99, 0, "IOV"))
			},
		},
		"deposit can be released only once": {
			Funds: []AccountBalance{
				{Wallet: bobCond.Address(), Amount: coin.NewCoin(100, 0, "IOV")},
			},
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &CreateDepositContractMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							ValidSince: now,
							ValidUntil: now.Add(2 * time.Hour),
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &DepositMsg{
							Metadata:          &weave.Metadata{Schema: 1},
							DepositContractID: weavetest.SequenceID(1),
							Amount:            coin.NewCoin(1, 0, "IOV"),
							Depositor:         bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now: now + 1000000,
					Tx: &weavetest.Tx{
						Msg: &ReleaseDepositMsg{
							Metadata:  &weave.Metadata{Schema: 1},
							DepositID: weavetest.SequenceID(2),
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
				{
					Now: now + 1000001,
					Tx: &weavetest.Tx{
						Msg: &ReleaseDepositMsg{
							Metadata:  &weave.Metadata{Schema: 1},
							DepositID: weavetest.SequenceID(2),
						},
					},
					BlockHeight: 103,
					WantErr:     errors.ErrState,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertFunds(t, db, bobCond.Address(), coin.NewCoin(100, 0, "IOV"))
			},
		},
		"anyone can release a deposit of an expired contract": {
			Funds: []AccountBalance{
				{Wallet: bobCond.Address(), Amount: coin.NewCoin(100, 0, "IOV")},
			},
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &CreateDepositContractMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							ValidSince: now,
							ValidUntil: now.Add(2 * time.Hour),
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &DepositMsg{
							Metadata:          &weave.Metadata{Schema: 1},
							DepositContractID: weavetest.SequenceID(1),
							Amount:            coin.NewCoin(1, 0, "IOV"),
							Depositor:         bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now: now + 100000,
					Tx: &weavetest.Tx{
						Msg: &ReleaseDepositMsg{
							Metadata:  &weave.Metadata{Schema: 1},
							DepositID: weavetest.SequenceID(2),
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				assertFunds(t, db, bobCond.Address(), coin.NewCoin(100, 0, "IOV"))
			},
		},
		"configuration owner can update configuration": {
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &UpdateConfigurationMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Patch: &Configuration{
								Metadata: &weave.Metadata{Schema: 1},
								Owner:    aliceCond.Address(),
								Admin:    bobCond.Address(),
								Bonuses: []DepositBonus{
									{LockinPeriod: asDays(1), BonusPercentage: 10},
								},
							},
						},
					},
					BlockHeight: 100,
					WantErr:     errors.ErrUnauthorized,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &UpdateConfigurationMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Patch: &Configuration{
								Metadata: &weave.Metadata{Schema: 1},
								Owner:    aliceCond.Address(),
								Admin:    bobCond.Address(),
								Bonuses: []DepositBonus{
									{LockinPeriod: asDays(1), BonusPercentage: 10},
								},
							},
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				{
					Now:        now + 2,
					Conditions: []weave.Condition{aliceCond},
					Tx: &weavetest.Tx{
						Msg: &UpdateConfigurationMsg{
							Metadata: &weave.Metadata{Schema: 1},
							Patch: &Configuration{
								Metadata: &weave.Metadata{Schema: 1},
								Owner:    bobCond.Address(),
								Admin:    charlieCond.Address(),
								Bonuses: []DepositBonus{
									{LockinPeriod: asDays(1), BonusPercentage: 10},
								},
							},
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
			},
		},
		"when a deposit is released all wallet funds are sent, not only originally allocated ones": {
			Funds: []AccountBalance{
				{Wallet: bobCond.Address(), Amount: coin.NewCoin(100, 0, "IOV")},
				{Wallet: charlieCond.Address(), Amount: coin.NewCoin(100, 0, "IOV")},
			},
			Requests: []Request{
				{
					Now:        now,
					Conditions: []weave.Condition{adminCond},
					Tx: &weavetest.Tx{
						Msg: &CreateDepositContractMsg{
							Metadata:   &weave.Metadata{Schema: 1},
							ValidSince: now,
							ValidUntil: now.Add(2 * time.Hour),
						},
					},
					BlockHeight: 100,
					WantErr:     nil,
				},
				{
					Now:        now + 1,
					Conditions: []weave.Condition{bobCond},
					Tx: &weavetest.Tx{
						Msg: &DepositMsg{
							Metadata:          &weave.Metadata{Schema: 1},
							DepositContractID: weavetest.SequenceID(1),
							Amount:            coin.NewCoin(11, 0, "IOV"),
							Depositor:         bobCond.Address(),
						},
					},
					BlockHeight: 101,
					WantErr:     nil,
				},
				// Transfer additional tokens to the deposit wallet.
				{
					Now:        now + 2,
					Conditions: []weave.Condition{charlieCond},
					Tx: &weavetest.Tx{
						Msg: &cash.SendMsg{
							Metadata:    &weave.Metadata{Schema: 1},
							Source:      charlieCond.Address(),
							Destination: depositAccount(weavetest.SequenceID(2)),
							Amount:      coin.NewCoinp(7, 0, "IOV"),
						},
					},
					BlockHeight: 102,
					WantErr:     nil,
				},
				{
					Now: now + 1000000,
					Tx: &weavetest.Tx{
						Msg: &ReleaseDepositMsg{
							Metadata:  &weave.Metadata{Schema: 1},
							DepositID: weavetest.SequenceID(2),
						},
					},
					BlockHeight: 103,
					WantErr:     nil,
				},
			},
			AfterTest: func(t *testing.T, db weave.KVStore) {
				// After releasing deposit, all funds from the
				// wallet must go back to the depositor
				// account.
				assertFunds(t, db, bobCond.Address(), coin.NewCoin(107, 0, "IOV"))
				assertFunds(t, db, charlieCond.Address(), coin.NewCoin(93, 0, "IOV"))
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, "termdeposit", "cash")

			rt := app.NewRouter()
			auth := &weavetest.CtxAuth{Key: "auth"}
			ctrl := cash.NewController(cash.NewBucket())
			RegisterRoutes(rt, auth, ctrl)

			// Required for transferring tokens request.
			cash.RegisterRoutes(rt, auth, ctrl)

			for _, b := range tc.Funds {
				if err := ctrl.CoinMint(db, b.Wallet, b.Amount); err != nil {
					t.Fatalf("cannot mint coins for %q: %s", b.Wallet, err)
				}
			}

			bonuses := tc.Bonuses
			if len(bonuses) == 0 {
				bonuses = []DepositBonus{{LockinPeriod: asDays(1), BonusPercentage: 10}}
			}

			config := Configuration{
				Metadata: &weave.Metadata{Schema: 1},
				Owner:    adminCond.Address(),
				Admin:    adminCond.Address(),
				Bonuses:  bonuses,
			}
			if err := gconf.Save(db, "termdeposit", &config); err != nil {
				t.Fatalf("cannot save configuration: %s", err)
			}

			for i, req := range tc.Requests {
				ctx := weave.WithHeight(context.Background(), req.BlockHeight)
				ctx = weave.WithChainID(ctx, "testchain-123")
				ctx = auth.SetConditions(ctx, req.Conditions...)
				ctx = weave.WithBlockTime(ctx, req.Now.Time())

				cache := db.CacheWrap()
				if _, err := rt.Check(ctx, cache, req.Tx); !req.WantErr.Is(err) {
					t.Fatalf("unexpected %d check error: want %q, got %+v", i, req.WantErr, err)
				}
				cache.Discard()
				if _, err := rt.Deliver(ctx, db, req.Tx); !req.WantErr.Is(err) {
					t.Fatalf("unexpected %d deliver error: want %q, got %+v", i, req.WantErr, err)
				} else if err == nil {
					if err := cache.Write(); err != nil {
						t.Fatalf("cannot write cache: %s", err)
					}
				}
			}

			if tc.AfterTest != nil {
				tc.AfterTest(t, db)
			}
		})
	}
}

func assertFunds(t testing.TB, db weave.KVStore, wallet weave.Address, funds coin.Coin) {
	t.Helper()

	ctrl := cash.NewController(cash.NewBucket())
	coins, err := ctrl.Balance(db, wallet)
	if err != nil {
		t.Fatalf("balance: %s", err)
	}
	if len(coins) != 1 {
		t.Fatalf("want %q funds, found %d coins: %q", funds, len(coins), coins)
	}
	if !coins[0].Equals(funds) {
		t.Fatalf("unexpected funds found: %q", coins[0])
	}
}

func TestDepositRateComputation(t *testing.T) {
	cases := map[string]struct {
		contract DepositContract
		conf     Configuration
		now      time.Time
		wantFrac Frac
		wantErr  *errors.Error
	}{
		"deposit duration between bonuses": {
			contract: DepositContract{
				ValidSince: 946684800, // 1 Jan 2000
				ValidUntil: 951004800, // 20 Feb 2000
			},
			conf: Configuration{
				Bonuses: []DepositBonus{
					{LockinPeriod: asDays(10), BonusPercentage: 10},
					{LockinPeriod: asDays(30), BonusPercentage: 30},
					{LockinPeriod: asDays(40), BonusPercentage: 50},
					{LockinPeriod: asDays(80), BonusPercentage: 80},
				},
			},
			now: asTime(t, "1 Feb 2000"),

			wantFrac: Frac{Numerator: 19, Denominator: 100},
			wantErr:  nil,
		},
		"deposit duration below minimal bonus": {
			contract: DepositContract{
				ValidSince: 946684800, // 1 Jan 2000
				ValidUntil: 951004800, // 20 Feb 2000
			},
			conf: Configuration{
				Bonuses: []DepositBonus{
					{LockinPeriod: asDays(10), BonusPercentage: 10},
					{LockinPeriod: asDays(20), BonusPercentage: 30},
				},
			},
			now: asTime(t, "15 Feb 2000"),

			wantFrac: Frac{Numerator: 10, Denominator: 100},
			wantErr:  nil,
		},
		"deposit duration above maximum bonus": {
			contract: DepositContract{
				ValidSince: 946684800, // 1 Jan 2000
				ValidUntil: 951004800, // 20 Feb 2000
			},
			conf: Configuration{
				Bonuses: []DepositBonus{
					{LockinPeriod: asDays(1), BonusPercentage: 10},
					{LockinPeriod: asDays(4), BonusPercentage: 80},
				},
			},
			now: asTime(t, "2 Jan 2000"),

			wantFrac: Frac{Numerator: 80, Denominator: 100},
			wantErr:  nil,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			f, err := depositRate(&tc.contract, tc.conf, tc.now)
			if !tc.wantErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}
			if f.Numerator != tc.wantFrac.Numerator || f.Denominator != tc.wantFrac.Denominator {
				t.Fatalf("unexpected result: %d/%d", f.Numerator, f.Denominator)
			}
		})
	}
}

func asTime(t testing.TB, s string) time.Time {
	t.Helper()
	tm, err := time.Parse("2 Jan 2006", s)
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

func asDays(days int) weave.UnixDuration {
	return weave.AsUnixDuration(time.Duration(days) * 24 * time.Hour)
}
