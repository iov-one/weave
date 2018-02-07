package coins

import (
	"strings"
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/stretchr/testify/assert"
)

func TestValidateSendMsg(t *testing.T) {
	empty := new(SendMsg)
	assert.Equal(t, pathSendMsg, empty.Path())
	assert.Error(t, empty.Validate())

	addr := weave.NewAddress([]byte{1, 2})
	addr2 := weave.NewAddress([]byte{3, 4})
	addr3 := weave.NewAddress([]byte{5, 6})

	pos := NewCoin(10, 0, "FOO")
	noSrc := &SendMsg{
		Amount: &pos,
		Dest:   addr,
	}
	err := noSrc.Validate()
	assert.Error(t, err)
	assert.True(t, errors.IsUnrecognizedAddressErr(err))

	// add a default source, so it validates
	good := noSrc.DefaultSource(addr2)
	assert.EqualValues(t, addr2, good.GetSrc())
	assert.NoError(t, good.Validate())

	// don't change source if already set
	good2 := good.DefaultSource(addr3)
	assert.EqualValues(t, addr2, good2.GetSrc())
	assert.NoError(t, good2.Validate())

	// try various error coniditons by modifying a good state
	good2.Dest = []byte{1, 2, 3}
	assert.Error(t, good2.Validate())

	good3 := noSrc.DefaultSource(addr3)
	assert.NoError(t, good3.Validate())
	good3.Note = "kfjuhewiufhgqwegf"
	assert.NoError(t, good3.Validate())
	good3.Note = strings.Repeat("foo", 300)
	err = good3.Validate()
	assert.Error(t, err)
	assert.True(t, IsInvalidMemoErr(err))

	neg := NewCoin(-3, 0, "FOO")
	minus := &SendMsg{
		Amount: &neg,
		Dest:   addr2,
		Src:    addr3,
	}
	err = minus.Validate()
	assert.Error(t, err)
	assert.True(t, IsInvalidAmountErr(err))

	bad := NewCoin(3, 4, "fab9")
	ugly := &SendMsg{
		Amount: &bad,
		Dest:   addr2,
		Src:    addr3,
	}
	err = ugly.Validate()
	assert.Error(t, err)
	assert.True(t, IsInvalidCurrencyErr(err))

}

func TestValidateFeeTx(t *testing.T) {
	var empty *FeeInfo
	err := empty.Validate()
	assert.Error(t, err)
	assert.True(t, errors.IsUnrecognizedAddressErr(err))

	addr := weave.NewAddress([]byte{8, 8})
	addr2 := weave.NewAddress([]byte{7, 7})

	nofee := &FeeInfo{Payer: addr}
	err = nofee.Validate()
	assert.Error(t, err)
	assert.True(t, IsInvalidAmountErr(err))

	pos := NewCoin(10, 0, "FOO")
	plus := &FeeInfo{Fees: &pos}
	err = plus.Validate()
	assert.Error(t, err)
	assert.True(t, errors.IsUnrecognizedAddressErr(err))

	full := plus.DefaultPayer(addr)
	assert.NoError(t, full.Validate())
	assert.EqualValues(t, addr, full.GetPayer())

	full2 := full.DefaultPayer(addr2)
	assert.NoError(t, full2.Validate())
	assert.EqualValues(t, addr, full2.GetPayer())

	zero := &FeeInfo{
		Payer: addr2,
		Fees:  &Coin{CurrencyCode: "BAR"},
	}
	assert.NoError(t, zero.Validate())

	neg := NewCoin(-3, 0, "FOO")
	minus := &FeeInfo{
		Payer: addr,
		Fees:  &neg,
	}
	err = minus.Validate()
	assert.Error(t, err)
	assert.True(t, IsInvalidAmountErr(err))

	bad := NewCoin(3, 0, "fab9")
	ugly := &FeeInfo{
		Payer: addr,
		Fees:  &bad,
	}
	err = ugly.Validate()
	assert.Error(t, err)
	assert.True(t, IsInvalidCurrencyErr(err))

}
