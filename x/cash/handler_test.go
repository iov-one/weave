package cash

import (
	"fmt"
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/orm"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type checkErr func(error) bool

func noErr(err error) bool { return err == nil }

func TestSend(t *testing.T) {
	var helpers x.TestHelpers

	foo := x.NewCoin(100, 0, "FOO")
	some := x.NewCoin(300, 0, "SOME")

	perm := weave.NewCondition("sig", "ed25519", []byte{1, 2, 3})
	perm2 := weave.NewCondition("sig", "ed25519", []byte{4, 5, 6})

	cases := []struct {
		signers       []weave.Condition
		initState     []orm.Object
		msg           weave.Msg
		expectCheck   checkErr
		expectDeliver checkErr
	}{
		0: {nil, nil, nil, errors.IsUnknownTxTypeErr, errors.IsUnknownTxTypeErr},
		1: {nil, nil, new(SendMsg), IsInvalidAmountErr, IsInvalidAmountErr},
		2: {nil, nil, &SendMsg{Amount: &foo}, errors.IsUnrecognizedAddressErr, errors.IsUnrecognizedAddressErr},
		3: {
			nil,
			nil,
			&SendMsg{Amount: &foo, Src: perm.Address(), Dest: perm2.Address()},
			errors.IsUnauthorizedErr,
			errors.IsUnauthorizedErr,
		},
		// sender has no account
		4: {
			[]weave.Condition{perm},
			nil,
			&SendMsg{Amount: &foo, Src: perm.Address(), Dest: perm2.Address()},
			noErr, // we don't check funds
			IsEmptyAccountErr,
		},
		// sender too poor
		5: {
			[]weave.Condition{perm},
			[]orm.Object{must(WalletWith(perm.Address(), &some))},
			&SendMsg{Amount: &foo, Src: perm.Address(), Dest: perm2.Address()},
			noErr, // we don't check funds
			IsInsufficientFundsErr,
		},
		// sender got cash
		6: {
			[]weave.Condition{perm},
			[]orm.Object{must(WalletWith(perm.Address(), &foo))},
			&SendMsg{Amount: &foo, Src: perm.Address(), Dest: perm2.Address()},
			noErr,
			noErr,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			auth := helpers.Authenticate(tc.signers...)
			controller := NewController(NewBucket())
			h := NewSendHandler(auth, controller)

			kv := store.MemStore()
			bucket := NewBucket()
			for _, wallet := range tc.initState {
				err := bucket.Save(kv, wallet)
				require.NoError(t, err)
			}

			tx := helpers.MockTx(tc.msg)

			_, err := h.Check(nil, kv, tx)
			assert.True(t, tc.expectCheck(err), "%+v", err)
			_, err = h.Deliver(nil, kv, tx)
			assert.True(t, tc.expectDeliver(err), "%+v", err)
		})
	}
}
