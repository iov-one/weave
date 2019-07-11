package escrow

import (
	"strings"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

// mustCombineCoins has one return value for tests...
func mustCombineCoins(cs ...coin.Coin) coin.Coins {
	s, err := coin.CombineCoins(cs...)
	if err != nil {
		panic(err)
	}
	return s
}

func TestCreateMsg(t *testing.T) {
	// good
	a := weavetest.NewCondition()
	b := weavetest.NewCondition()
	c := weave.NewCondition("monkey", "gelato", []byte("berry"))

	timeout := weave.AsUnixTime(time.Now())

	// good
	plus := mustCombineCoins(coin.NewCoin(100, 0, "FOO"))
	// invalid
	minus := mustCombineCoins(coin.NewCoin(100, 0, "BAR"),
		coin.NewCoin(-20, 0, "FIT"))
	mixed := coin.Coins{{Whole: 100, Ticker: "bad"}}

	cases := map[string]struct {
		msg   *CreateMsg
		check error
	}{
		"nothing": {
			&CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
			errors.ErrEmpty,
		},
		"happy path": {
			&CreateMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Source:      a.Address(),
				Arbiter:     b.Address(),
				Destination: c.Address(),
				Amount:      plus,
				Timeout:     timeout,
			},
			nil,
		},
		"missing source okay, dups okay": {
			&CreateMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Arbiter:     c.Address(),
				Destination: c.Address(),
				Amount:      plus,
				Timeout:     timeout,
				Memo:        "some string",
			},
			nil,
		},
		"negative amount": {
			&CreateMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Arbiter:     b.Address(),
				Destination: c.Address(),
				Amount:      minus,
				Timeout:     timeout,
			},
			errors.ErrAmount,
		},
		"improperly formatted amount": {
			&CreateMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Arbiter:     b.Address(),
				Destination: c.Address(),
				Amount:      mixed,
				Timeout:     timeout,
			},
			errors.ErrCurrency,
		},
		"missing amount": {
			&CreateMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Arbiter:     b.Address(),
				Destination: c.Address(),
				Timeout:     timeout,
			},
			errors.ErrAmount,
		},
		"invalid memo": {
			&CreateMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Arbiter:     b.Address(),
				Destination: c.Address(),
				Amount:      plus,
				Timeout:     timeout,
				Memo:        strings.Repeat("foo", 100),
			},
			errors.ErrInput,
		},
		"zero timeout": {
			&CreateMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Arbiter:     b.Address(),
				Destination: c.Address(),
				Amount:      plus,
				Timeout:     0,
			},
			errors.ErrInput,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.msg.Validate()
			assert.IsErr(t, tc.check, err)
		})
	}
}

func TestReleaseMsg(t *testing.T) {
	// valid: fixed 8 byte id
	escrow := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	// invalid: other size id
	scarecrow := []byte{1, 2, 3, 4}

	// good
	plus := mustCombineCoins(coin.NewCoin(100, 0, "FOO"))
	// invalid
	minus := mustCombineCoins(coin.NewCoin(100, 0, "BAR"),
		coin.NewCoin(-20, 0, "FIT"))
	mixed := coin.Coins{{Whole: 100, Ticker: "bad"}}

	cases := map[string]struct {
		msg   *ReleaseMsg
		check error
	}{
		"nothing": {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
			errors.ErrInput,
		},
		"proper: valid amount": {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
				Amount:   plus,
			},
			nil,
		},
		"missing amount okay": {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
			},
			nil,
		},
		"invalid id": {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: scarecrow,
			},
			errors.ErrInput,
		},
		"missing id": {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Amount:   plus,
			},
			errors.ErrInput,
		},
		"negative amount": {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
				Amount:   minus,
			},
			errors.ErrAmount,
		},
		"improperly formatted amount": {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
				Amount:   mixed,
			},
			errors.ErrCurrency,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.msg.Validate()
			assert.IsErr(t, tc.check, err)
		})
	}
}

func TestReturnMsg(t *testing.T) {
	// valid: fixed 8 byte id
	escrow := []byte{0xff, 0, 1, 3, 6, 6, 6, 6}
	// invalid: other size id
	scarecrow := []byte{1, 2, 3, 4}

	cases := map[string]struct {
		msg   *ReturnMsg
		check error
	}{
		"missing id": {
			&ReturnMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
			errors.ErrInput,
		},
		"proper: valid id": {
			&ReturnMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
			},
			nil,
		},
		"invalid id": {
			&ReturnMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: scarecrow,
			},
			errors.ErrInput,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.msg.Validate()
			assert.IsErr(t, tc.check, err)
		})
	}
}

func TestUpdateEscrowMsg(t *testing.T) {
	// valid: fixed 8 byte id
	escrow := []byte{0xf, 0, 0, 0xb, 0xa, 0xd, 7, 7}
	// invalid: other size id
	scarecrow := []byte{1, 2, 3, 4}

	// good
	a := weavetest.NewCondition()
	b := weavetest.NewCondition()
	c := weave.NewCondition("monkey", "gelato", []byte("berry"))

	cases := map[string]struct {
		msg   *UpdatePartiesMsg
		check error
	}{
		"nothing": {
			&UpdatePartiesMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
			errors.ErrInput,
		},
		"proper: valid id, one valid permission": {
			&UpdatePartiesMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
				Source:   a.Address(),
			},
			nil,
		},
		"valid escrow, no permissions": {
			&UpdatePartiesMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
			},
			errors.ErrEmpty,
		},
		"invalid escrow, proper permissions": {
			&UpdatePartiesMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: scarecrow,
				Source:   a.Address(),
			},
			errors.ErrInput,
		},
		"allow multiple permissions": {
			&UpdatePartiesMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				EscrowId:    escrow,
				Destination: b.Address(),
				Arbiter:     c.Address(),
			},
			nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.msg.Validate()
			assert.IsErr(t, tc.check, err)
		})
	}
}
