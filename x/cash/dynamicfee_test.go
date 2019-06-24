package cash

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestCacheWriteFail(t *testing.T) {
	auth := &weavetest.Auth{
		Signer: weavetest.NewCondition(),
	}

	store := store.MemStore()
	migration.MustInitPkg(store, "cash")
	config := Configuration{
		CollectorAddress: weavetest.NewCondition().Address(),
		MinimalFee:       coin.NewCoin(0, 1, "IOV"),
	}
	if err := gconf.Save(store, "cash", &config); err != nil {
		t.Fatalf("cannot save configuration: %s", err)
	}
	bucket := NewBucket()
	ctrl := NewController(bucket)
	if err := ctrl.CoinMint(store, auth.Signer.Address(), coin.NewCoin(100, 0, "IOV")); err != nil {
		t.Fatalf("cannot mint: %s", err)
	}

	handler := &weavetest.Handler{
		CheckResult:   weave.CheckResult{Log: "all good"},
		DeliverResult: weave.DeliverResult{Log: "all good"},
	}
	tx := &txMock{info: &FeeInfo{Fees: coin.NewCoinp(1, 0, "IOV")}}

	// Register an error that is guaranteed to be unique.
	myerr := errors.Register(921928, "my error")

	db := &cacheableStoreMock{
		CacheableKVStore: store,
		err:              myerr,
	}

	decorator := NewDynamicFeeDecorator(auth, ctrl)

	if _, err := decorator.Check(context.TODO(), db, tx, handler); !myerr.Is(err) {
		t.Fatalf("unexpected check result error: %+v", err)
	}

	if _, err := decorator.Deliver(context.TODO(), db, tx, handler); !myerr.Is(err) {
		t.Fatalf("unexpected deliver result error: %+v", err)
	}
}

// cacheableStoreMock is a mock of a store and a cache wrap. Use it to pass
// through all operation to wrapped CacheableKVStore. Write call returns
// defined error.
type cacheableStoreMock struct {
	weave.CacheableKVStore
	err error
}

// CachceWrap overwrites wrapped store method in order to return
// self-reference. cacheableStoreMock implements KVCacheWrap interface as well.
func (s *cacheableStoreMock) CacheWrap() weave.KVCacheWrap {
	return s
}

// Write implements KVCacheWrap interface.
func (c *cacheableStoreMock) Write() error {
	return c.err
}

// Discard implements KVCacheWrap interface.
func (cacheableStoreMock) Discard() {}

func TestDynamicFeeDecorator(t *testing.T) {
	perm1 := weave.NewCondition("sigs", "ed25519", []byte{1, 2, 3})
	//perm2 := weave.NewCondition("sigs", "ed25519", []byte{3, 4, 5})
	perm3 := weave.NewCondition("custom", "type", []byte{0xAB})

	collectorAddr := perm3.Address()

	walletObj := func(a weave.Address, w, f int64, ticker string) orm.Object {
		t.Helper()
		obj, err := WalletWith(a, coin.NewCoinp(w, f, ticker))
		if err != nil {
			t.Fatalf("cannot create a wallet: %s", err)
		}
		return obj
	}

	cases := map[string]struct {
		signers    []weave.Condition
		handler    *weavetest.Handler
		minimumFee coin.Coin
		txFee      coin.Coin
		// Wallet state created before running Check
		initWallets []orm.Object
		// Wallet state applied after running Check but before running Deliver
		updateWallets []orm.Object

		wantCheckErr     *errors.Error
		wantCheckTxFee   coin.Coin
		wantDeliverErr   *errors.Error
		wantDeliverTxFee coin.Coin
		wantGasPayment   int64
	}{
		"on success full transaction fee is charged": {
			signers: []weave.Condition{perm1},
			handler: &weavetest.Handler{},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "BTC"),
			},
			minimumFee:       coin.NewCoin(0, 23, "BTC"),
			txFee:            coin.NewCoin(0, 421, "BTC"),
			wantCheckTxFee:   coin.NewCoin(0, 421, "BTC"),
			wantDeliverTxFee: coin.NewCoin(0, 421, "BTC"),
			wantGasPayment:   421,
		},
		"on a handler check failure minimum fee is charged": {
			signers: []weave.Condition{perm1},
			handler: &weavetest.Handler{CheckErr: ErrTestingError},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "BTC"),
			},
			minimumFee:     coin.NewCoin(0, 23, "BTC"),
			txFee:          coin.NewCoin(0, 421, "BTC"),
			wantCheckErr:   ErrTestingError,
			wantCheckTxFee: coin.NewCoin(0, 23, "BTC"),
		},
		"on insufficient fee funds minimum fee is charged": {
			signers: []weave.Condition{perm1},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 0, 100, "BTC"),
			},
			minimumFee:     coin.NewCoin(0, 23, "BTC"),
			txFee:          coin.NewCoin(0, 421, "BTC"), // Wallet has not enough.
			wantCheckErr:   errors.ErrAmount,
			wantCheckTxFee: coin.NewCoin(0, 23, "BTC"),
		},
		"on insufficient funds minimum fee withdraw fails": {
			signers: []weave.Condition{perm1},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 0, 1, "BTC"),
			},
			minimumFee:     coin.NewCoin(0, 23, "BTC"),  // Wallet has not enough.
			txFee:          coin.NewCoin(0, 421, "BTC"), // Wallet has not enough.
			wantCheckErr:   errors.ErrAmount,
			wantCheckTxFee: coin.Coin{},
		},
		/*
			// this now triggers an error on initialize - invalid data :)
			"on transaction fee ticker mismatch minimum fee with no currency accepts anything": {
				signers: []weave.Condition{perm1},
				initWallets: []orm.Object{
					walletObj(perm1.Address(), 1, 0, "BTC"),
				},
				minimumFee:   coin.NewCoin(0, 23, ""),
				txFee:        coin.NewCoin(0, 421, "ETH"),
				wantCheckErr: errors.ErrHuman,
			},
		*/
		"on a handler deliver failure only minimum fee is charged": {
			signers: []weave.Condition{perm1},
			handler: &weavetest.Handler{DeliverErr: ErrTestingError},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "BTC"),
			},
			minimumFee:       coin.NewCoin(0, 11, "BTC"),
			txFee:            coin.NewCoin(0, 44, "BTC"),
			wantGasPayment:   44, // This assumes that transaction fee was charged.
			wantCheckTxFee:   coin.NewCoin(0, 44, "BTC"),
			wantDeliverErr:   ErrTestingError,
			wantDeliverTxFee: coin.NewCoin(0, 11, "BTC"),
		},
		"success if we pay exactly required fee": {
			signers: []weave.Condition{perm1},
			handler: &weavetest.Handler{
				DeliverResult: weave.DeliverResult{RequiredFee: coin.NewCoin(0, 421, "IOV")},
			},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "IOV"),
			},
			minimumFee:       coin.NewCoin(0, 23, "IOV"),
			txFee:            coin.NewCoin(0, 421, "IOV"),
			wantCheckTxFee:   coin.NewCoin(0, 421, "IOV"),
			wantDeliverTxFee: coin.NewCoin(0, 421, "IOV"),
			wantGasPayment:   421,
		},
		"success if we pay more than required fee": {
			signers: []weave.Condition{perm1},
			handler: &weavetest.Handler{
				DeliverResult: weave.DeliverResult{RequiredFee: coin.NewCoin(0, 77, "IOV")},
			},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "IOV"),
			},
			minimumFee:       coin.NewCoin(0, 23, "IOV"),
			txFee:            coin.NewCoin(0, 421, "IOV"),
			wantCheckTxFee:   coin.NewCoin(0, 421, "IOV"),
			wantDeliverTxFee: coin.NewCoin(0, 421, "IOV"),
			wantGasPayment:   421,
		},
		"failure if we pay less than required fee": {
			signers: []weave.Condition{perm1},
			handler: &weavetest.Handler{
				CheckResult: weave.CheckResult{RequiredFee: coin.NewCoin(1, 0, "IOV")},
			},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "IOV"),
			},
			minimumFee:     coin.NewCoin(0, 23, "IOV"),
			txFee:          coin.NewCoin(0, 421, "IOV"),
			wantCheckErr:   errors.ErrAmount,
			wantCheckTxFee: coin.NewCoin(0, 23, "IOV"),
		},
		"failure if we pay different currency than required fee": {
			signers: []weave.Condition{perm1},
			handler: &weavetest.Handler{
				CheckResult: weave.CheckResult{RequiredFee: coin.NewCoin(0, 72, "ETH")},
			},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "IOV"),
			},
			minimumFee:     coin.NewCoin(0, 23, "IOV"),
			txFee:          coin.NewCoin(0, 421, "IOV"),
			wantCheckErr:   errors.ErrAmount,
			wantCheckTxFee: coin.NewCoin(0, 23, "IOV"),
		},
		"failure if we pay less than required fee also in delivettx": {
			signers: []weave.Condition{perm1},
			handler: &weavetest.Handler{
				DeliverResult: weave.DeliverResult{RequiredFee: coin.NewCoin(1, 0, "IOV")},
			},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "IOV"),
			},
			minimumFee:       coin.NewCoin(0, 23, "IOV"),
			txFee:            coin.NewCoin(0, 421, "IOV"),
			wantCheckTxFee:   coin.NewCoin(0, 421, "IOV"),
			wantGasPayment:   421,
			wantDeliverErr:   errors.ErrAmount,
			wantDeliverTxFee: coin.NewCoin(0, 23, "IOV"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			auth := &weavetest.Auth{Signers: tc.signers}
			bucket := NewBucket()
			ctrl := NewController(bucket)
			h := NewDynamicFeeDecorator(auth, ctrl)

			tx := &txMock{info: &FeeInfo{Fees: &tc.txFee}}

			db := store.MemStore()
			migration.MustInitPkg(db, "cash")

			config := Configuration{
				CollectorAddress: collectorAddr,
				MinimalFee:       tc.minimumFee,
			}
			if err := gconf.Save(db, "cash", &config); err != nil {
				t.Fatalf("cannot save configuration: %s", err)
			}

			ensureWallets(t, db, tc.initWallets)

			cache := db.CacheWrap()

			cRes, err := h.Check(nil, cache, tx, tc.handler)
			if !tc.wantCheckErr.Is(err) {
				t.Fatalf("got check error: %v", err)
			}
			if err == nil && tc.wantGasPayment != cRes.GasPayment {
				t.Errorf("gas payment: %d", cRes.GasPayment)
			}

			assertCharged(t, cache, ctrl, tc.wantCheckTxFee)

			ensureWallets(t, cache, tc.updateWallets)

			// If the check failed, deliver must not be called.
			if tc.wantCheckErr != nil {
				return
			}

			cache.Discard()

			if _, err = h.Deliver(nil, cache, tx, tc.handler); !tc.wantDeliverErr.Is(err) {
				t.Fatalf("got deliver error: %v", err)
			}

			assertCharged(t, cache, ctrl, tc.wantDeliverTxFee)
		})
	}
}

// ensureWallets persist state of given wallet objects in the database. If
// a wallet already exist it is overwritten.
func ensureWallets(t *testing.T, db weave.KVStore, wallets []orm.Object) {
	t.Helper()

	bucket := NewBucket()
	for i, w := range wallets {
		if err := bucket.Save(db, w); err != nil {
			t.Fatalf("cannot set %d wallet: %s", i, err)
		}
	}
}

// assertCharged check that given account was charged according to the fee
// configuration.
func assertCharged(t *testing.T, db weave.KVStore, ctrl Controller, want coin.Coin) {
	t.Helper()

	minimumFee := mustLoadConf(db).MinimalFee
	collectorAddr := mustLoadConf(db).CollectorAddress

	switch chargedFee, err := ctrl.Balance(db, collectorAddr); {
	case err == nil:
		wantTx := coin.Coins{&want}
		if !wantTx.Equals(chargedFee) {
			t.Errorf("charged fee: %v", chargedFee)
		}
	case errors.ErrNotFound.Is(err):
		if minimumFee.IsZero() {
			// Minimal fee is zero so the collector account is zero
			// as well (not even created). All good.
		} else {
			if want.IsZero() {
				// This is a weird case when a transaction was
				// submitted but the signer does not have
				// enough funds to pay the minimum (anti spam)
				// fee.
			} else {
				t.Error("no fee charged")
			}
		}
	default:
		t.Errorf("cannot check collector account balance: %s", err)
	}
}

type txMock struct {
	weave.Tx
	FeeTx
	info *FeeInfo
}

func (m *txMock) GetFees() *FeeInfo {
	return m.info
}

// Declare a unique error that can be matched in tests. This error is declared
// only in tests so there is no way it can be returned by the implementation by
// an accident.
var ErrTestingError = errors.Register(123456789, "testing error")
