package coins

import (
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/store"
	"github.com/stretchr/testify/assert"
)

type checkErr func(error) bool

func noErr(err error) bool { return err == nil }

type auther struct {
	signers []weave.Address
}

func (a auther) GetSigners(weave.Context) []weave.Address {
	return a.signers
}

type mockTx struct {
	msg weave.Msg
}

var _ weave.Tx = mockTx{}

func (m mockTx) GetMsg() weave.Msg {
	return m.msg
}

func TestSend(t *testing.T) {
	foo := NewCoin(100, 0, "FOO")
	addr := weave.NewAddress([]byte{1, 2, 3})
	addr2 := weave.NewAddress([]byte{4, 5, 6})

	cases := [...]struct {
		signers       []weave.Address
		initState     []Wallet // just key and set (store can be nil)
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
			&SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			errors.IsUnauthorizedErr,
			errors.IsUnauthorizedErr,
		},
		// sender has no account
		4: {
			[]weave.Address{addr},
			nil,
			&SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			noErr, // we don't check funds
			IsEmptyAccountErr,
		},
		// sender too poor
		5: {
			[]weave.Address{addr},
			[]Wallet{{
				key: NewKey(addr),
				Set: mustNewSet(NewCoin(300, 0, "SOME")),
			}},
			&SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			noErr, // we don't check funds
			IsInsufficientFundsErr,
		},
		// sender got cash
		6: {
			[]weave.Address{addr},
			[]Wallet{{
				key: NewKey(addr),
				Set: mustNewSet(foo),
			}},
			&SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			noErr,
			noErr,
		},
	}

	for i, tc := range cases {
		auth := auther{tc.signers}.GetSigners
		h := NewSendHandler(auth)

		kv := store.MemStore()
		for _, wallet := range tc.initState {
			wallet.store = kv
			wallet.Save()
		}

		tx := mockTx{tc.msg}

		_, err := h.Check(nil, kv, tx)
		assert.True(t, tc.expectCheck(err), "%d: %+v", i, err)
		_, err = h.Deliver(nil, kv, tx)
		assert.True(t, tc.expectDeliver(err), "%d: %+v", i, err)
	}
}
