package cash

import (
	"testing"

	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type feeTx struct {
	info *FeeInfo
}

var _ weave.Tx = (*feeTx)(nil)
var _ FeeTx = feeTx{}

func (feeTx) GetMsg() (weave.Msg, error) {
	return nil, nil
}

func (f feeTx) GetFees() *FeeInfo {
	return f.info
}

func (f feeTx) Marshal() ([]byte, error) {
	return nil, errors.ErrHuman.New("TODO: not implemented")
}

func (f *feeTx) Unmarshal([]byte) error {
	return errors.ErrHuman.New("TODO: not implemented")
}

func must(obj orm.Object, err error) orm.Object {
	if err != nil {
		panic(err)
	}
	return obj
}

func TestFees(t *testing.T) {
	cash := coin.NewCoin(50, 0, "FOO")
	min := coin.NewCoin(0, 1234, "FOO")
	perm := weave.NewCondition("sigs", "ed25519", []byte{1, 2, 3})
	perm2 := weave.NewCondition("sigs", "ed25519", []byte{3, 4, 5})
	perm3 := weave.NewCondition("custom", "type", []byte{0xAB})

	cases := map[string]struct {
		signers   []weave.Condition
		initState []orm.Object
		fee       *FeeInfo
		min       coin.Coin
		expect    checkErr
	}{
		"no fee given, nothing expected": {
			min:    coin.Coin{},
			expect: noErr,
		},
		"no fee given, something expected": {
			min:    min,
			expect: errors.ErrInsufficientAmount.Is,
		},
		"no signer given": {
			fee:    &FeeInfo{Fees: &min},
			min:    min,
			expect: errors.ErrInvalidInput.Is,
		},
		"use default signer, but not enough money": {
			signers: []weave.Condition{perm},
			fee:     &FeeInfo{Fees: &min},
			min:     min,
			expect:  errors.ErrEmpty.Is,
		},
		"signer can cover min, but not pledge": {
			signers:   []weave.Condition{perm},
			initState: []orm.Object{must(WalletWith(perm.Address(), &min))},
			fee:       &FeeInfo{Fees: &cash},
			min:       min,
			expect:    errors.ErrInsufficientAmount.Is,
		},
		"all proper": {
			signers:   []weave.Condition{perm},
			initState: []orm.Object{must(WalletWith(perm.Address(), &cash))},
			fee:       &FeeInfo{Fees: &min},
			min:       min,
			expect:    noErr,
		},
		"trying to pay from wrong account": {
			signers:   []weave.Condition{perm},
			initState: []orm.Object{must(WalletWith(perm2.Address(), &cash))},
			fee:       &FeeInfo{Payer: perm2.Address(), Fees: &min},
			min:       min,
			expect:    errors.ErrUnauthorized.Is,
		},
		"fee without an empty ticker is not accepted": {
			signers:   []weave.Condition{perm},
			initState: []orm.Object{must(WalletWith(perm.Address(), &cash))},
			fee:       &FeeInfo{Fees: &min},
			min:       coin.NewCoin(0, 1000, ""),
			expect:    coin.ErrInvalidCurrency.Is,
		},
		"no fee (zero value) is acceptable": {
			signers:   []weave.Condition{perm},
			initState: []orm.Object{must(WalletWith(perm.Address(), &cash))},
			fee:       &FeeInfo{Fees: coin.NewCoinp(0, 1, "FOO")},
			min:       coin.NewCoin(0, 0, ""),
			expect:    noErr,
		},
		"wrong currency checked": {
			signers:   []weave.Condition{perm},
			initState: []orm.Object{must(WalletWith(perm.Address(), &cash))},
			fee:       &FeeInfo{Fees: &min},
			min:       coin.NewCoin(0, 1000, "NOT"),
			expect:    coin.ErrInvalidCurrency.Is,
		},
		"has the cash, but didn't offer enough fees": {
			signers:   []weave.Condition{perm},
			initState: []orm.Object{must(WalletWith(perm.Address(), &cash))},
			fee:       &FeeInfo{Fees: &min},
			min:       coin.NewCoin(0, 45000, "FOO"),
			expect:    errors.ErrInsufficientAmount.Is,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			auth := &weavetest.Auth{Signers: tc.signers}
			controller := NewController(NewBucket())
			h := NewFeeDecorator(auth, controller)

			kv := store.MemStore()

			gconf.SetValue(kv, GconfCollectorAddress, perm3.Address())
			gconf.SetValue(kv, GconfMinimalFee, tc.min)

			bucket := NewBucket()
			for _, wallet := range tc.initState {
				err := bucket.Save(kv, wallet)
				require.NoError(t, err)
			}

			tx := &feeTx{tc.fee}

			_, err := h.Check(nil, kv, tx, &weavetest.Handler{})
			assert.True(t, tc.expect(err), "%+v", err)
			_, err = h.Deliver(nil, kv, tx, &weavetest.Handler{})
			assert.True(t, tc.expect(err), "%+v", err)
		})
	}
}
