package coins

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cmp int

const (
	neg  cmp = -1
	zero     = 0
	pos      = 1
)

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
		i := strconv.Itoa(idx)

		// make sure both show proper results
		assert.Equal(t, tc.a.IsZero(), tc.aState == zero, i)
		assert.Equal(t, tc.a.IsPositive(), tc.aState == pos, i)
		assert.Equal(t, !tc.a.IsNonNegative(), tc.aState == neg, i)

		assert.Equal(t, tc.b.IsZero(), tc.bState == zero, i)
		assert.Equal(t, tc.b.IsPositive(), tc.bState == pos, i)
		assert.Equal(t, !tc.b.IsNonNegative(), tc.bState == neg, i)

		// make sure compare is correct
		assert.Equal(t, tc.a.Compare(tc.b), tc.expect, i)

		assert.True(t, tc.a.SameType(tc.b), i)
	}
}

func TestValidCoin(t *testing.T) {
	cases := []struct {
		coin            Coin
		valid           bool
		normalized      Coin
		normalizedValid bool
	}{
		{
			NewCoin(4, -123456789, "FOO"),
			false,
			NewCoin(3, 876543211, "FOO"),
			true,
		},
		{
			NewCoin(1, 0, "eth2"),
			false,
			NewCoin(1, 0, "eth2"),
			false,
		},
		{
			NewCoin(2, -1500500500, "ABC"),
			false,
			NewCoin(0, 499499500, "ABC"),
			true,
		},
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
	}

	for idx, tc := range cases {
		i := strconv.Itoa(idx)

		// Validate this one
		err := tc.coin.Validate()
		// normalize and check if there are still errors
		nrm, nerr := tc.coin.normalize()
		if nerr == nil {
			nerr = nrm.Validate()
		}

		if tc.valid {
			assert.NoError(t, err, i)
		} else {
			assert.Error(t, err, i)
		}

		assert.Equal(t, tc.normalized, nrm, i)
		assert.True(t, tc.normalized.Equals(nrm), i)

		if tc.normalizedValid {
			assert.NoError(t, nerr, i)
		} else {
			assert.Error(t, nerr, i)
		}
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
		// normal math
		{
			NewCoin(7, 5000, "ABC"),
			NewCoin(-4, -12000, "ABC"),
			NewCoin(2, 999993000, "ABC"),
			false,
		},
		// overflow
		{
			NewCoin(500500500, 0, "SEE"),
			NewCoin(500500500, 0, "SEE"),
			Coin{},
			true,
		},
	}

	for idx, tc := range cases {
		i := strconv.Itoa(idx)

		c, err := tc.a.Add(tc.b)
		if tc.bad {
			assert.Error(t, err, i)
		} else {
			assert.NoError(t, err, i)
			assert.Equal(t, tc.res, c, i)
		}
	}
}

func TestMakeSet(t *testing.T) {
	// TODO: verify constructor checks well for errors

	cases := []struct {
		inputs   []Coin
		isEmpty  bool
		isNonNeg bool
		has      []Coin // <= the wallet
		dontHave []Coin // > or outside the wallet
		isErr    bool
	}{
		// empty
		{
			nil,
			true,
			true,
			nil,
			[]Coin{NewCoin(0, 0, "")},
			false,
		},
		// ignore 0
		{
			[]Coin{NewCoin(0, 0, "FOO")},
			true,
			true,
			nil,
			[]Coin{NewCoin(0, 0, "FOO")},
			false,
		},
		// simple
		{
			[]Coin{NewCoin(40, 0, "FUD")},
			false,
			true,
			[]Coin{NewCoin(10, 0, "FUD"), NewCoin(40, 0, "FUD")},
			[]Coin{NewCoin(40, 1, "FUD"), NewCoin(40, 0, "FUN")},
			false,
		},
		// out of order, with negative
		{
			[]Coin{NewCoin(-20, -3, "FIN"), NewCoin(40, 5, "BON")},
			false,
			false,
			[]Coin{NewCoin(40, 4, "BON"), NewCoin(-30, 0, "FIN")},
			[]Coin{NewCoin(40, 6, "BON"), NewCoin(-20, 0, "FIN")},
			false,
		},
		// combine and remove
		{
			[]Coin{NewCoin(-123, -456, "BOO"), NewCoin(123, 456, "BOO")},
			true,
			true,
			nil,
			[]Coin{NewCoin(0, 0, "BOO")},
			false,
		},
		// safely combine
		{
			[]Coin{NewCoin(12, 0, "ADA"), NewCoin(-123, -456, "BOO"), NewCoin(124, 756, "BOO")},
			false,
			true,
			[]Coin{NewCoin(12, 0, "ADA"), NewCoin(1, 300, "BOO")},
			[]Coin{NewCoin(13, 0, "ADA"), NewCoin(1, 400, "BOO")},
			false,
		},
		// verify invalid input cur -> error
		{
			[]Coin{NewCoin(1, 2, "AL2")},
			false, false, nil, nil,
			true,
		},
		// verify invalid input values -> error
		{
			[]Coin{NewCoin(maxInt+3, 2, "AND")},
			false, false, nil, nil,
			true,
		},
		// if we can combine invalid inputs, then acceptable?
		{
			[]Coin{NewCoin(maxInt+3, 2, "AND"), NewCoin(-10, 0, "AND")},
			false,
			true,
			[]Coin{NewCoin(maxInt-8, 0, "AND")},
			nil,
			false,
		},
	}

	for idx, tc := range cases {
		i := strconv.Itoa(idx)

		s, err := NewSet(tc.inputs...)
		if tc.isErr {
			assert.Error(t, err, i)
			continue
		}

		require.NoError(t, err, i)
		assert.NoError(t, s.Validate(), i)
		assert.Equal(t, tc.isEmpty, s.IsEmpty(), i)
		assert.Equal(t, tc.isNonNeg, s.IsNonNegative(), i)

		for _, h := range tc.has {
			assert.True(t, s.Contains(h), i)
		}
		for _, d := range tc.dontHave {
			assert.False(t, s.Contains(d), i)
		}
	}
}

// TestCombine checks combine and equals
// and thereby checks add
func TestCombine(t *testing.T) {
	cases := []struct {
		a, b  Set
		comb  Set
		isErr bool
	}{
		// empty
		{
			mustNewSet(), mustNewSet(), mustNewSet(), false,
		},
		// one plus one
		{
			mustNewSet(NewCoin(maxInt, 5, "ABC")),
			mustNewSet(NewCoin(-maxInt, -4, "ABC")),
			mustNewSet(NewCoin(0, 1, "ABC")),
			false,
		},
		// multiple
		{
			mustNewSet(NewCoin(7, 8, "FOO"), NewCoin(8, 9, "BAR")),
			mustNewSet(NewCoin(5, 4, "APE"), NewCoin(2, 1, "FOO")),
			mustNewSet(NewCoin(5, 4, "APE"), NewCoin(8, 9, "BAR"), NewCoin(9, 9, "FOO")),
			false,
		},
		// overflows
		{
			mustNewSet(NewCoin(maxInt, 0, "ADA")),
			mustNewSet(NewCoin(2, 0, "ADA")),
			Set{},
			true,
		},
	}

	for idx, tc := range cases {
		i := strconv.Itoa(idx)

		ac := tc.a.Count()
		bc := tc.b.Count()

		res, err := tc.a.Combine(tc.b)
		// don't modify original sets
		assert.Equal(t, ac, tc.a.Count(), i)
		assert.Equal(t, bc, tc.b.Count(), i)
		if tc.isErr {
			assert.Error(t, err, i)
		} else {
			require.NoError(t, err, i)
			assert.NoError(t, res.Validate(), i)
			assert.True(t, tc.comb.Equals(res), i)
			// result should only be the same as an input
			// if the other input was empty
			assert.Equal(t, tc.a.IsEmpty(),
				tc.b.Equals(res), i)
			assert.Equal(t, tc.b.IsEmpty(),
				tc.a.Equals(res), i)
		}
	}
}
