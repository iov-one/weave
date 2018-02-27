package coins

import (
	"fmt"
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
	"github.com/stretchr/testify/assert"
)

type checkErr func(error) bool

func noErr(err error) bool { return err == nil }

type auther struct {
	signers []weave.Address
}

var _ x.Authenticator = auther{}

func (a auther) GetPermissions(weave.Context) []weave.Address {
	return a.signers
}

func (a auther) HasPermission(ctx weave.Context, addr weave.Address) bool {
	for _, s := range a.signers {
		if addr.Equals(s) {
			return true
		}
	}
	return false
}

type mockTx struct {
	msg weave.Msg
}

var _ weave.Tx = (*mockTx)(nil)

func (m mockTx) GetMsg() (weave.Msg, error) {
	return m.msg, nil
}

func (m mockTx) Marshal() ([]byte, error) {
	return nil, errors.ErrInternal("TODO: not implemented")
}

func (m *mockTx) Unmarshal([]byte) error {
	return errors.ErrInternal("TODO: not implemented")
}

func TestSend(t *testing.T) {
	foo := x.NewCoin(100, 0, "FOO")
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
				Set: Set{mustCombineCoins(x.NewCoin(300, 0, "SOME"))},
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
				Set: Set{mustCombineCoins(foo)},
			}},
			&SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			noErr,
			noErr,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			auth := auther{tc.signers}
			h := NewSendHandler(auth)

			kv := store.MemStore()
			for _, wallet := range tc.initState {
				wallet.store = kv
				wallet.Save()
			}

			tx := &mockTx{tc.msg}

			_, err := h.Check(nil, kv, tx)
			assert.True(t, tc.expectCheck(err), "%+v", err)
			_, err = h.Deliver(nil, kv, tx)
			assert.True(t, tc.expectDeliver(err), "%+v", err)
		})
	}
}
