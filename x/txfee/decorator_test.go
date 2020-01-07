package txfee

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
)

func TestDecorator(t *testing.T) {
	cases := map[string]struct {
		Conf           Configuration
		Tx             weave.Tx
		Handler        weave.Handler
		WantCheckErr   *errors.Error
		WantCheckFee   coin.Coin
		WantDeliverErr *errors.Error
		WantDeliverFee coin.Coin
	}{
		"transaction for free": {
			Conf: Configuration{
				Metadata:  &weave.Metadata{Schema: 1},
				Owner:     weavetest.NewCondition().Address(),
				FreeBytes: 5,
				BaseFee:   asCoin(t, "123456 DOGE"),
			},
			Handler:        &weavetest.Handler{},
			Tx:             &txMock{Raw: []byte{1, 2, 3, 4, 5}},
			WantCheckFee:   coin.Coin{},
			WantDeliverFee: coin.Coin{},
		},
		"tiny transaction with a minimal fee": {
			Conf: Configuration{
				Metadata:  &weave.Metadata{Schema: 1},
				Owner:     weavetest.NewCondition().Address(),
				FreeBytes: 2,
				BaseFee:   asCoin(t, "1.1 IOV"),
			},
			Handler:        &weavetest.Handler{},
			Tx:             &txMock{Raw: []byte{1, 2, 3}},
			WantCheckFee:   asCoin(t, "1.1 IOV"),
			WantDeliverFee: asCoin(t, "1.1 IOV"),
		},
		"huge transaction with a big fee": {
			Conf: Configuration{
				Metadata:  &weave.Metadata{Schema: 1},
				Owner:     weavetest.NewCondition().Address(),
				FreeBytes: 1,
				BaseFee:   asCoin(t, "12345.6789 IOV"),
			},
			Handler: &weavetest.Handler{},
			Tx:      &txMock{Raw: make([]byte, 100000)},
			// The below is:
			//   ((100000 - 1) ^ 2) * 12345.6789
			WantCheckFee:   asCoin(t, "123454319876565.6789 IOV"),
			WantDeliverFee: asCoin(t, "123454319876565.6789 IOV"),
		},
		"fee overflow": {
			Conf: Configuration{
				Metadata:  &weave.Metadata{Schema: 1},
				Owner:     weavetest.NewCondition().Address(),
				FreeBytes: 1,
				BaseFee:   asCoin(t, "123456789 IOV"),
			},
			Handler:        &weavetest.Handler{},
			Tx:             &txMock{Raw: make([]byte, 1000000)},
			WantCheckErr:   errors.ErrOverflow,
			WantDeliverErr: errors.ErrOverflow,
		},
		"additional fee to already existing one": {
			Conf: Configuration{
				Metadata:  &weave.Metadata{Schema: 1},
				Owner:     weavetest.NewCondition().Address(),
				FreeBytes: 1,
				BaseFee:   asCoin(t, "7 IOV"),
			},
			Handler: &weavetest.Handler{
				CheckResult:   weave.CheckResult{RequiredFee: coin.NewCoin(5, 0, "IOV")},
				DeliverResult: weave.DeliverResult{RequiredFee: coin.NewCoin(5, 0, "IOV")},
			},
			Tx:             &txMock{Raw: make([]byte, 2)},
			WantCheckFee:   asCoin(t, "12 IOV"),
			WantDeliverFee: asCoin(t, "12 IOV"),
		},
		"incompatible fee tickers": {
			Conf: Configuration{
				Metadata:  &weave.Metadata{Schema: 1},
				Owner:     weavetest.NewCondition().Address(),
				FreeBytes: 1,
				BaseFee:   asCoin(t, "7 DOGE"),
			},
			Handler: &weavetest.Handler{
				CheckResult:   weave.CheckResult{RequiredFee: coin.NewCoin(5, 0, "BTC")},
				DeliverResult: weave.DeliverResult{RequiredFee: coin.NewCoin(5, 0, "BTC")},
			},
			Tx:             &txMock{Raw: make([]byte, 2)},
			WantCheckErr:   errors.ErrCurrency,
			WantDeliverErr: errors.ErrCurrency,
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			decorator := NewDecorator()
			db := store.MemStore()
			migration.MustInitPkg(db, "txfee")

			if err := gconf.Save(db, "txfee", &tc.Conf); err != nil {
				t.Fatalf("save configuration: %s", err)
			}

			cres, err := decorator.Check(nil, db, tc.Tx, tc.Handler)
			if !tc.WantCheckErr.Is(err) {
				t.Fatalf("check returned an unexpected error: %v", err)
			}
			if tc.WantCheckErr == nil && !tc.WantCheckFee.Equals(cres.RequiredFee) {
				t.Fatalf("unexpected check fee: %v", cres.RequiredFee)
			}

			dres, err := decorator.Deliver(nil, db, tc.Tx, tc.Handler)
			if !tc.WantDeliverErr.Is(err) {
				t.Fatalf("deliver returned an unexpected error: %v", err)
			}
			if tc.WantDeliverErr == nil && !tc.WantDeliverFee.Equals(dres.RequiredFee) {
				t.Fatalf("unexpected deliver fee: %v", dres.RequiredFee)
			}
		})
	}
}

type txMock struct {
	weave.Tx
	Raw []byte
}

func (m *txMock) Marshal() ([]byte, error) {
	return m.Raw, nil
}

func asCoin(t testing.TB, s string) coin.Coin {
	t.Helper()
	c, err := coin.ParseHumanFormat(s)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDecoratorWithoutConfiguration(t *testing.T) {
	decorator := NewDecorator()
	db := store.MemStore()
	migration.MustInitPkg(db, "txfee")

	tx := &txMock{Raw: []byte{1, 2, 3, 4, 5}}
	handler := &weavetest.Handler{}

	// Without configuration, decorator should be a no-op wrapper.

	cres, err := decorator.Check(nil, db, tx, handler)
	if err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if !cres.RequiredFee.IsZero() {
		t.Fatalf("unexpected check fee: %v", cres.RequiredFee)
	}

	dres, err := decorator.Deliver(nil, db, tx, handler)
	if err != nil {
		t.Fatalf("unexpected deliver error: %v", err)
	}
	if !dres.RequiredFee.IsZero() {
		t.Fatalf("unexpected deliver fee: %v", dres.RequiredFee)
	}
}
