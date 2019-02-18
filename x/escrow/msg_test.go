package escrow

import (
	"fmt"
	"strings"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/stretchr/testify/assert"
)

// mustCombineCoins has one return value for tests...
func mustCombineCoins(cs ...x.Coin) x.Coins {
	s, err := x.CombineCoins(cs...)
	if err != nil {
		panic(err)
	}
	return s
}

type checkErr func(error) bool

func noErr(err error) bool { return err == nil }

func TestCreateEscrowMsg(t *testing.T) {
	var helpers x.TestHelpers

	// good
	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()
	c := weave.NewCondition("monkey", "gelato", []byte("berry"))
	// invalid
	d := weave.Condition("foobar")

	// good
	plus := mustCombineCoins(x.NewCoin(100, 0, "FOO"))
	// invalid
	minus := mustCombineCoins(x.NewCoin(100, 0, "BAR"),
		x.NewCoin(-20, 0, "FIT"))
	mixed := x.Coins{{Whole: 100, Ticker: "bad"}}

	cases := []struct {
		msg   *CreateEscrowMsg
		check checkErr
	}{
		// nothing
		0: {new(CreateEscrowMsg), IsMissingConditionErr},
		// proper
		1: {
			&CreateEscrowMsg{
				Src:       a.Address(),
				Arbiter:   b,
				Recipient: c.Address(),
				Amount:    plus,
				Timeout:   333,
			},
			noErr,
		},
		// missing sender okay, dups okay
		2: {
			&CreateEscrowMsg{
				Arbiter:   c,
				Recipient: c.Address(),
				Amount:    plus,
				Timeout:   52,
				Memo:      "some string",
			},
			noErr,
		},
		// invalid permissions
		3: {
			&CreateEscrowMsg{
				Arbiter:   d,
				Recipient: c.Address(),
				Amount:    plus,
				Timeout:   52,
			},
			IsInvalidConditionErr,
		},
		// negative amount
		4: {
			&CreateEscrowMsg{
				Arbiter:   b,
				Recipient: c.Address(),
				Amount:    minus,
				Timeout:   52,
			},
			cash.IsInvalidAmountErr,
		},
		// improperly formatted amount
		5: {
			&CreateEscrowMsg{
				Arbiter:   b,
				Recipient: c.Address(),
				Amount:    mixed,
				Timeout:   52,
			},
			x.InvalidCurrencyErr.Is,
		},
		// missing amount
		6: {
			&CreateEscrowMsg{
				Arbiter:   b,
				Recipient: c.Address(),
				Timeout:   52,
			},
			cash.IsInvalidAmountErr,
		},
		// invalid memo
		7: {
			&CreateEscrowMsg{
				Arbiter:   b,
				Recipient: c.Address(),
				Amount:    plus,
				Timeout:   52,
				Memo:      strings.Repeat("foo", 100),
			},
			IsInvalidMetadataErr,
		},
		// invalid timeout
		8: {
			&CreateEscrowMsg{
				Arbiter:   b,
				Recipient: c.Address(),
				Amount:    plus,
				Timeout:   -8,
			},
			IsInvalidMetadataErr,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			assert.Equal(t, pathCreateEscrowMsg, tc.msg.Path())
			err := tc.msg.Validate()
			assert.True(t, tc.check(err), "%+v", err)
		})
	}
}

func TestReleaseEscrowMsg(t *testing.T) {
	// valid: fixed 8 byte id
	escrow := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	// invalid: other size id
	scarecrow := []byte{1, 2, 3, 4}

	// good
	plus := mustCombineCoins(x.NewCoin(100, 0, "FOO"))
	// invalid
	minus := mustCombineCoins(x.NewCoin(100, 0, "BAR"),
		x.NewCoin(-20, 0, "FIT"))
	mixed := x.Coins{{Whole: 100, Ticker: "bad"}}

	cases := []struct {
		msg   *ReleaseEscrowMsg
		check checkErr
	}{
		// nothing
		0: {new(ReleaseEscrowMsg), IsInvalidMetadataErr},
		// proper: valid amount
		1: {
			&ReleaseEscrowMsg{
				EscrowId: escrow,
				Amount:   plus,
			},
			noErr,
		},
		// missing amount okay
		2: {
			&ReleaseEscrowMsg{
				EscrowId: escrow,
			},
			noErr,
		},
		// invalid id
		3: {
			&ReleaseEscrowMsg{
				EscrowId: scarecrow,
			},
			IsInvalidMetadataErr,
		},
		// missing id
		4: {
			&ReleaseEscrowMsg{
				Amount: plus,
			},
			IsInvalidMetadataErr,
		},
		// negative amount
		5: {
			&ReleaseEscrowMsg{
				EscrowId: escrow,
				Amount:   minus,
			},
			cash.IsInvalidAmountErr,
		},
		// improperly formatted amount
		6: {
			&ReleaseEscrowMsg{
				EscrowId: escrow,
				Amount:   mixed,
			},
			x.InvalidCurrencyErr.Is,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			assert.Equal(t, pathReleaseEscrowMsg, tc.msg.Path())
			err := tc.msg.Validate()
			assert.True(t, tc.check(err), "%+v", err)
		})
	}
}

func TestReturnEscrowMsg(t *testing.T) {
	// valid: fixed 8 byte id
	escrow := []byte{0xff, 0, 1, 3, 6, 6, 6, 6}
	// invalid: other size id
	scarecrow := []byte{1, 2, 3, 4}

	cases := []struct {
		msg   *ReturnEscrowMsg
		check checkErr
	}{
		// missing id
		0: {new(ReturnEscrowMsg), IsInvalidMetadataErr},
		// proper: valid id
		1: {
			&ReturnEscrowMsg{
				EscrowId: escrow,
			},
			noErr,
		},
		// invalid id
		2: {
			&ReturnEscrowMsg{
				EscrowId: scarecrow,
			},
			IsInvalidMetadataErr,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			assert.Equal(t, pathReturnEscrowMsg, tc.msg.Path())
			err := tc.msg.Validate()
			assert.True(t, tc.check(err), "%+v", err)
		})
	}
}

func TestUpdateEscrowMsg(t *testing.T) {
	var helpers x.TestHelpers

	// valid: fixed 8 byte id
	escrow := []byte{0xf, 0, 0, 0xb, 0xa, 0xd, 7, 7}
	// invalid: other size id
	scarecrow := []byte{1, 2, 3, 4}

	// good
	_, a := helpers.MakeKey()
	_, b := helpers.MakeKey()
	c := weave.NewCondition("monkey", "gelato", []byte("berry"))
	// invalid
	d := weave.Condition("foobar")

	cases := []struct {
		msg   *UpdateEscrowPartiesMsg
		check checkErr
	}{
		// nothing
		0: {new(UpdateEscrowPartiesMsg), IsInvalidMetadataErr},
		// proper: valid id, one valid permission
		1: {
			&UpdateEscrowPartiesMsg{
				EscrowId: escrow,
				Sender:   a.Address(),
			},
			noErr,
		},
		// valid escrow, no permissions
		2: {
			&UpdateEscrowPartiesMsg{
				EscrowId: escrow,
			},
			IsMissingConditionErr,
		},
		// invalid escrow, proper permissions
		3: {
			&UpdateEscrowPartiesMsg{
				EscrowId: scarecrow,
				Sender:   a.Address(),
			},
			IsInvalidMetadataErr,
		},
		// allow multiple permissions
		4: {
			&UpdateEscrowPartiesMsg{
				EscrowId:  escrow,
				Recipient: b.Address(),
				Arbiter:   c,
			},
			noErr,
		},
		// check for valid permissions
		5: {
			&UpdateEscrowPartiesMsg{
				EscrowId: escrow,
				Arbiter:  d,
			},
			IsInvalidConditionErr,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			assert.Equal(t, pathUpdateEscrowPartiesMsg, tc.msg.Path())
			err := tc.msg.Validate()
			assert.True(t, tc.check(err), "%+v", err)
		})
	}
}
