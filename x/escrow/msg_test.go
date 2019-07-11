package escrow

import (
	"fmt"
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

	cases := []struct {
		msg   *CreateMsg
		check error
	}{
		// nothing
		0: {
			&CreateMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
			errors.ErrEmpty,
		},
		// proper
		1: {
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
		// missing source okay, dups okay
		2: {
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
		// negative amount
		3: {
			&CreateMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Arbiter:     b.Address(),
				Destination: c.Address(),
				Amount:      minus,
				Timeout:     timeout,
			},
			errors.ErrAmount,
		},
		// improperly formatted amount
		4: {
			&CreateMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Arbiter:     b.Address(),
				Destination: c.Address(),
				Amount:      mixed,
				Timeout:     timeout,
			},
			errors.ErrCurrency,
		},
		// missing amount
		5: {
			&CreateMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				Arbiter:     b.Address(),
				Destination: c.Address(),
				Timeout:     timeout,
			},
			errors.ErrAmount,
		},
		// invalid memo
		6: {
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
		// zero timeout
		7: {
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

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
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

	cases := []struct {
		msg   *ReleaseMsg
		check error
	}{
		// nothing
		0: {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
			errors.ErrInput,
		},
		// proper: valid amount
		1: {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
				Amount:   plus,
			},
			nil,
		},
		// missing amount okay
		2: {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
			},
			nil,
		},
		// invalid id
		3: {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: scarecrow,
			},
			errors.ErrInput,
		},
		// missing id
		4: {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Amount:   plus,
			},
			errors.ErrInput,
		},
		// negative amount
		5: {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
				Amount:   minus,
			},
			errors.ErrAmount,
		},
		// improperly formatted amount
		6: {
			&ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
				Amount:   mixed,
			},
			errors.ErrCurrency,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
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

	cases := []struct {
		msg   *ReturnMsg
		check error
	}{
		// missing id
		0: {
			&ReturnMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
			errors.ErrInput,
		},
		// proper: valid id
		1: {
			&ReturnMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
			},
			nil,
		},
		// invalid id
		2: {
			&ReturnMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: scarecrow,
			},
			errors.ErrInput,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
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

	cases := []struct {
		msg   *UpdatePartiesMsg
		check error
	}{
		// nothing
		0: {
			&UpdatePartiesMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
			errors.ErrInput,
		},
		// proper: valid id, one valid permission
		1: {
			&UpdatePartiesMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
				Source:   a.Address(),
			},
			nil,
		},
		// valid escrow, no permissions
		2: {
			&UpdatePartiesMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: escrow,
			},
			errors.ErrEmpty,
		},
		// invalid escrow, proper permissions
		3: {
			&UpdatePartiesMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: scarecrow,
				Source:   a.Address(),
			},
			errors.ErrInput,
		},
		// allow multiple permissions
		4: {
			&UpdatePartiesMsg{
				Metadata:    &weave.Metadata{Schema: 1},
				EscrowId:    escrow,
				Destination: b.Address(),
				Arbiter:     c.Address(),
			},
			nil,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			err := tc.msg.Validate()
			assert.IsErr(t, tc.check, err)
		})
	}
}
