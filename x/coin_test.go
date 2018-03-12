package x

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type cmp int

const (
	neg  cmp = -1
	zero     = 0
	pos      = 1
)

func TestIssuer(t *testing.T) {
	cases := []struct {
		a        Coin
		id       string
		b        Coin
		sameType bool
	}{
		{NewCoin(1, 2, "FOO"), "FOO", NewCoin(12, 0, "FOO"), true},
		{NewCoin(1, 2, "BAR"), "BAR", NewCoin(12, 0, "FOO"), false},
		{
			NewCoin(1, 2, "FOO").WithIssuer("chain1"),
			"chain1/FOO",
			NewCoin(12, 0, "FOO"),
			false,
		},
		{
			NewCoin(1, 2, "FOO"),
			"FOO",
			NewCoin(12, 0, "FOO").WithIssuer("chain1"),
			false,
		},
		{
			NewCoin(1, 2, "FOO").WithIssuer("chain1"),
			"chain1/FOO",
			NewCoin(12, 0, "FOO").WithIssuer("chain1"),
			true,
		},
		{
			NewCoin(1, 2, "WIN").WithIssuer("my-chain").Negative(),
			"my-chain/WIN",
			NewCoin(12, 0, "WIN").WithIssuer("my-chain"),
			true,
		},
	}

	for idx, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", idx), func(t *testing.T) {
			assert.Equal(t, tc.id, tc.a.ID())
			assert.Equal(t, tc.sameType, tc.a.SameType(tc.b))
		})
	}
}

func TestCompareCoin(t *testing.T) {

	cases := []struct {
		a      Coin
		b      Coin
		expect int
		aState cmp
		bState cmp
	}{
		{
			NewCoin(20, 1234, "ABC"),
			NewCoin(19, 999999999, "ABC"),
			1,
			pos,
			pos,
		},
		{
			NewCoin(0, -2, "FOO"),
			NewCoin(0, 1, "FOO"),
			-1,
			neg,
			pos,
		},
		{
			NewCoin(-4, -2456, "BAR"),
			NewCoin(-4, -4567, "BAR"),
			1,
			neg,
			neg,
		},
		{
			Coin{},
			Coin{},
			0,
			zero,
			zero,
		},
	}

	for idx, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", idx), func(t *testing.T) {
			// make sure both show proper results
			assert.Equal(t, tc.a.IsZero(), tc.aState == zero)
			assert.Equal(t, tc.a.IsPositive(), tc.aState == pos)
			assert.Equal(t, !tc.a.IsNonNegative(), tc.aState == neg)

			assert.Equal(t, tc.b.IsZero(), tc.bState == zero)
			assert.Equal(t, tc.b.IsPositive(), tc.bState == pos)
			assert.Equal(t, !tc.b.IsNonNegative(), tc.bState == neg)

			// make sure compare is correct
			assert.Equal(t, tc.a.Compare(tc.b), tc.expect)

			assert.True(t, tc.a.SameType(tc.b))
		})
	}
}

func TestValidCoin(t *testing.T) {
	cases := []struct {
		coin            Coin
		valid           bool
		normalized      Coin
		normalizedValid bool
	}{
		// interger and fraction with same sign
		{
			NewCoin(4, -123456789, "FOO"),
			false,
			NewCoin(3, 876543211, "FOO"),
			true,
		},
		// invalid coin id
		{
			NewCoin(1, 0, "eth2"),
			false,
			NewCoin(1, 0, "eth2"),
			false,
		},
		// make sure issuer is maintained throughout
		{
			NewCoin(2, -1500500500, "ABC").WithIssuer("my-chain"),
			false,
			NewCoin(0, 499499500, "ABC").WithIssuer("my-chain"),
			true,
		},
		// from negative to positive rollover
		{
			NewCoin(-1, 1777888111, "ABC"),
			false,
			NewCoin(0, 777888111, "ABC"),
			true,
		},
		{
			NewCoin(0, -100, "DIN"),
			true,
			NewCoin(0, -100, "DIN"),
			true,
		},
		{
			NewCoin(MaxInt, FracUnit+4, "DIN"),
			false,
			Coin{},
			false,
		},
	}

	for idx, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", idx), func(t *testing.T) {

			// Validate this one
			err := tc.coin.Validate()
			// normalize and check if there are still errors
			nrm, nerr := tc.coin.normalize()
			if nerr == nil {
				nerr = nrm.Validate()
			}

			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.Equal(t, tc.normalized, nrm)
			assert.True(t, tc.normalized.Equals(nrm))

			if tc.normalizedValid {
				assert.NoError(t, nerr)
			} else {
				assert.Error(t, nerr)
			}
		})
	}
}

func TestAddCoin(t *testing.T) {
	base := NewCoin(17, 2345566, "DEF")
	cases := []struct {
		a, b Coin
		res  Coin
		bad  bool
	}{
		// plus and minus equals 0
		{base, base.Negative(), NewCoin(0, 0, "DEF"), false},
		// wrong types
		{
			NewCoin(1, 2, "FOO"),
			NewCoin(2, 3, "BAR"),
			Coin{},
			true,
		},
		// wrong issuer
		{
			NewCoin(1, 2, "FOO").WithIssuer("chain-1"),
			NewCoin(2, 3, "FOO"),
			Coin{},
			true,
		},
		// negative hold issuer
		{
			NewCoin(7, 5000, "DEF").WithIssuer("lucky7"),
			NewCoin(5, 5000, "DEF").WithIssuer("lucky7").Negative(),
			NewCoin(2, 0, "DEF").WithIssuer("lucky7"),
			false,
		},
		// normal math
		{
			NewCoin(7, 5000, "ABC"),
			NewCoin(-4, -12000, "ABC"),
			NewCoin(2, 999993000, "ABC"),
			false,
		},
		// normal math with issuer
		{
			NewCoin(7, 5000, "ABC").WithIssuer("chain-1"),
			NewCoin(-4, -12000, "ABC").WithIssuer("chain-1"),
			NewCoin(2, 999993000, "ABC").WithIssuer("chain-1"),
			false,
		},
		// overflow
		{
			NewCoin(500500500123456, 0, "SEE"),
			NewCoin(500500500123456, 0, "SEE"),
			Coin{},
			true,
		},
	}

	for idx, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", idx), func(t *testing.T) {
			c, err := tc.a.Add(tc.b)
			if tc.bad {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.res, c)
			}
		})
	}
}
