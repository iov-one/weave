package cash

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
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
	return nil, errors.ErrInternalLegacy("TODO: not implemented")
}

func (f *feeTx) Unmarshal([]byte) error {
	return errors.ErrInternalLegacy("TODO: not implemented")
}

type okHandler struct{}

var _ weave.Handler = okHandler{}

func (okHandler) Check(weave.Context, weave.KVStore,
	weave.Tx) (weave.CheckResult, error) {
	return weave.CheckResult{}, nil
}

func (okHandler) Deliver(weave.Context, weave.KVStore,
	weave.Tx) (weave.DeliverResult, error) {
	return weave.DeliverResult{}, nil
}

func must(obj orm.Object, err error) orm.Object {
	if err != nil {
		panic(err)
	}
	return obj
}

func TestFees(t *testing.T) {
	var helpers x.TestHelpers

	cash := x.NewCoin(50, 0, "FOO")
	min := x.NewCoin(0, 1234, "FOO")
	perm := weave.NewCondition("sigs", "ed25519", []byte{1, 2, 3})
	perm2 := weave.NewCondition("sigs", "ed25519", []byte{3, 4, 5})
	perm3 := weave.NewCondition("custom", "type", []byte{0xAB})

	cases := []struct {
		signers   []weave.Condition
		initState []orm.Object
		fee       *FeeInfo
		min       x.Coin
		expect    checkErr
	}{
		// no fee given, nothing expected
		0: {nil, nil, nil, x.Coin{}, noErr},
		// no fee given, something expected
		1: {nil, nil, nil, min, errors.ErrInsufficientAmount.Is},
		// no signer given
		2: {nil, nil, &FeeInfo{Fees: &min}, min, errors.IsUnrecognizedAddressErr},
		// use default signer, but not enough money
		3: {
			[]weave.Condition{perm},
			nil,
			&FeeInfo{Fees: &min},
			min,
			errors.ErrEmpty.Is,
		},
		// signer can cover min, but not pledge
		4: {
			[]weave.Condition{perm},
			[]orm.Object{must(WalletWith(perm.Address(), &min))},
			&FeeInfo{Fees: &cash},
			min,
			errors.ErrInsufficientAmount.Is,
		},
		// all proper
		5: {
			[]weave.Condition{perm},
			[]orm.Object{must(WalletWith(perm.Address(), &cash))},
			&FeeInfo{Fees: &min},
			min,
			noErr,
		},
		// trying to pay from wrong account
		6: {
			[]weave.Condition{perm},
			[]orm.Object{must(WalletWith(perm2.Address(), &cash))},
			&FeeInfo{Payer: perm2.Address(), Fees: &min},
			min,
			errors.IsUnauthorizedErr,
		},
		// can pay in any fee
		7: {
			[]weave.Condition{perm},
			[]orm.Object{must(WalletWith(perm.Address(), &cash))},
			&FeeInfo{Fees: &min},
			x.NewCoin(0, 1000, ""),
			noErr,
		},
		// wrong currency checked
		8: {
			[]weave.Condition{perm},
			[]orm.Object{must(WalletWith(perm.Address(), &cash))},
			&FeeInfo{Fees: &min},
			x.NewCoin(0, 1000, "NOT"),
			x.ErrInvalidCurrency.Is,
		},
		// has the cash, but didn't offer enough fees
		9: {
			[]weave.Condition{perm},
			[]orm.Object{must(WalletWith(perm.Address(), &cash))},
			&FeeInfo{Fees: &min},
			x.NewCoin(0, 45000, "FOO"),
			errors.ErrInsufficientAmount.Is,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			auth := helpers.Authenticate(tc.signers...)
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

			_, err := h.Check(nil, kv, tx, okHandler{})
			assert.True(t, tc.expect(err), "%+v", err)
			_, err = h.Deliver(nil, kv, tx, okHandler{})
			assert.True(t, tc.expect(err), "%+v", err)
		})
	}
}
