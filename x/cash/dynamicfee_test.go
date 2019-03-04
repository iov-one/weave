package cash

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
)

func TestDynamicFeeDecorator(t *testing.T) {
	perm1 := weave.NewCondition("sigs", "ed25519", []byte{1, 2, 3})
	//perm2 := weave.NewCondition("sigs", "ed25519", []byte{3, 4, 5})
	perm3 := weave.NewCondition("custom", "type", []byte{0xAB})

	collectorAddr := perm3.Address()

	coinp := func(w, f int64, t string) *x.Coin {
		c := x.NewCoin(w, f, t)
		return &c
	}

	walletObj := func(a weave.Address, w, f int64, ticker string) orm.Object {
		t.Helper()
		obj, err := WalletWith(a, coinp(w, f, ticker))
		if err != nil {
			t.Fatalf("cannot create a wallet: %s", err)
		}
		return obj
	}

	cases := map[string]struct {
		signers    []weave.Condition
		handler    *handlerMock
		minimumFee x.Coin
		txFee      x.Coin
		// Wallet state created before running Check
		initWallets []orm.Object
		// Wallet state applied after running Check but before running Deliver
		updateWallets []orm.Object

		wantCheckErr     error
		wantCheckTxFee   x.Coin
		wantDeliverErr   error
		wantDeliverTxFee x.Coin
		wantGasPayment   int64
	}{
		"on success full transaction fee is charged": {
			signers: []weave.Condition{perm1},
			handler: &handlerMock{},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "BTC"),
			},
			minimumFee:       x.NewCoin(0, 23, "BTC"),
			txFee:            x.NewCoin(0, 421, "BTC"),
			wantCheckTxFee:   x.NewCoin(0, 421, "BTC"),
			wantDeliverTxFee: x.NewCoin(0, 421, "BTC"),
			wantGasPayment:   421,
		},
		"on a handler check failure minimum fee is charged": {
			signers: []weave.Condition{perm1},
			handler: &handlerMock{checkErr: ErrTestingError},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "BTC"),
			},
			minimumFee:     x.NewCoin(0, 23, "BTC"),
			txFee:          x.NewCoin(0, 421, "BTC"),
			wantCheckErr:   ErrTestingError,
			wantCheckTxFee: x.NewCoin(0, 23, "BTC"),
		},
		"on insufficient fee funds minimum fee is charged": {
			signers: []weave.Condition{perm1},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 0, 100, "BTC"),
			},
			minimumFee:     x.NewCoin(0, 23, "BTC"),
			txFee:          x.NewCoin(0, 421, "BTC"), // Wallet has not enough.
			wantCheckErr:   errors.ErrInsufficientAmount,
			wantCheckTxFee: x.NewCoin(0, 23, "BTC"),
		},
		"on inssuficient funds minimum fee withdraw fails": {
			signers: []weave.Condition{perm1},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 0, 1, "BTC"),
			},
			minimumFee:     x.NewCoin(0, 23, "BTC"),  // Wallet has not enough.
			txFee:          x.NewCoin(0, 421, "BTC"), // Wallet has not enough.
			wantCheckErr:   errors.ErrInsufficientAmount,
			wantCheckTxFee: x.Coin{},
		},
		"on transaction fee ticker mismatch minimum fee with no currency accepts anything": {
			signers: []weave.Condition{perm1},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "BTC"),
			},
			minimumFee:     x.NewCoin(0, 23, ""),
			txFee:          x.NewCoin(0, 421, "ETH"),
			wantCheckErr:   errors.ErrInsufficientAmount,
			wantCheckTxFee: x.NewCoin(0, 23, "BTC"),
		},
		"on a handler deliver failure only minimum fee is charged": {
			signers: []weave.Condition{perm1},
			handler: &handlerMock{deliverErr: ErrTestingError},
			initWallets: []orm.Object{
				walletObj(perm1.Address(), 1, 0, "BTC"),
			},
			minimumFee:       x.NewCoin(0, 11, "BTC"),
			txFee:            x.NewCoin(0, 44, "BTC"),
			wantGasPayment:   44, // This assumes that transaction fee was charged.
			wantCheckTxFee:   x.NewCoin(0, 44, "BTC"),
			wantDeliverErr:   ErrTestingError,
			wantDeliverTxFee: x.NewCoin(0, 11, "BTC"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			auth := helpers.Authenticate(tc.signers...)
			bucket := NewBucket()
			ctrl := NewController(bucket)
			h := NewDynamicFeeDecorator(auth, ctrl)

			tx := &txMock{info: &FeeInfo{Fees: &tc.txFee}}

			db := store.MemStore()

			gconf.SetValue(db, GconfCollectorAddress, collectorAddr)
			gconf.SetValue(db, GconfMinimalFee, tc.minimumFee)

			ensureWallets(t, db, tc.initWallets)

			cache := db.CacheWrap()

			cRes, err := h.Check(nil, cache, tx, tc.handler)
			if !errors.Is(tc.wantCheckErr, err) {
				t.Fatalf("got check error: %v", err)
			}
			if tc.wantGasPayment != cRes.GasPayment {
				t.Errorf("gas payment: %d", cRes.GasPayment)
			}

			assertCharged(t, cache, ctrl, tc.wantCheckTxFee)

			ensureWallets(t, cache, tc.updateWallets)

			// If the check failed, deliver must not be called.
			if tc.wantCheckErr != nil {
				return
			}

			cache.Discard()

			if _, err = h.Deliver(nil, cache, tx, tc.handler); !errors.Is(tc.wantDeliverErr, err) {
				t.Fatalf("got deliver error: %v", err)
			}

			assertCharged(t, cache, ctrl, tc.wantDeliverTxFee)
		})
	}
}

var helpers x.TestHelpers

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
func assertCharged(t *testing.T, db weave.KVStore, ctrl Controller, want x.Coin) {
	t.Helper()

	minimumFee := gconf.Coin(db, GconfMinimalFee)
	collectorAddr := gconf.Address(db, GconfCollectorAddress)

	switch chargedFee, err := ctrl.Balance(db, collectorAddr); {
	case err == nil:
		wantTx := x.Coins{&want}
		if !wantTx.Equals(chargedFee) {
			t.Errorf("charged fee: %v", chargedFee)
		}
	case errors.Is(errors.ErrNotFound, err):
		if minimumFee.IsZero() {
			// Minimal fee is zero so the collector account is zero
			// as well (not even created). All good.
		} else {
			if want.IsZero() {
				// This is a weird case when a transaction was
				// submitted but the signer does not have
				// enough funds to pay the minimum (anty spam)
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

type handlerMock struct {
	weave.Handler

	checkRes weave.CheckResult
	checkErr error

	deliverRes weave.DeliverResult
	deliverErr error
}

func (m *handlerMock) Check(weave.Context, weave.KVStore, weave.Tx) (weave.CheckResult, error) {
	return m.checkRes, m.checkErr
}

func (m *handlerMock) Deliver(weave.Context, weave.KVStore, weave.Tx) (weave.DeliverResult, error) {
	return m.deliverRes, m.deliverErr
}
