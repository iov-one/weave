package coin

import (
	"fmt"
	"math"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/stretchr/testify/assert"
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
			NewCoin(2, -1500500500, "ABC"),
			false,
			NewCoin(0, 499499500, "ABC"),
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
	cases := map[string]struct {
		a, b    Coin
		wantRes Coin
		wantErr *errors.Error
	}{
		"plus and minus equals 0": {
			a:       base,
			b:       base.Negative(),
			wantRes: NewCoin(0, 0, "DEF"),
		},
		"wrong types": {
			a:       NewCoin(1, 2, "FOO"),
			b:       NewCoin(2, 3, "BAR"),
			wantRes: Coin{},
			wantErr: ErrInvalidCurrency,
		},
		"normal math": {
			a:       NewCoin(7, 5000, "ABC"),
			b:       NewCoin(-4, -12000, "ABC"),
			wantRes: NewCoin(2, 999993000, "ABC"),
		},
		"overflow": {
			a:       NewCoin(500500500123456, 0, "SEE"),
			b:       NewCoin(500500500123456, 0, "SEE"),
			wantRes: NewCoin(0, 0, ""),
			wantErr: ErrInvalidCoin,
		},
		"adding to zero coin": {
			a:       NewCoin(0, 0, ""),
			b:       NewCoin(1, 0, "DOGE"),
			wantRes: NewCoin(1, 0, "DOGE"),
		},
		"adding a zero coin": {
			a:       NewCoin(1, 0, "DOGE"),
			b:       NewCoin(0, 0, ""),
			wantRes: NewCoin(1, 0, "DOGE"),
		},
		"adding a non zero coin without a ticker": {
			a:       NewCoin(1, 0, "DOGE"),
			b:       NewCoin(1, 0, ""),
			wantErr: ErrInvalidCurrency,
		},
		"adding to non zero coin without a ticker": {
			a:       NewCoin(1, 0, ""),
			b:       NewCoin(1, 0, "DOGE"),
			wantErr: ErrInvalidCurrency,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			res, err := tc.a.Add(tc.b)
			if !tc.wantErr.Is(err) {
				t.Fatalf("got error: %v", err)
			}
			if tc.wantErr == nil && !tc.wantRes.Equals(res) {
				t.Fatalf("unexepcted result: %v", res)
			}
		})
	}
}

func TestCoinGTE(t *testing.T) {
	cases := map[string]struct {
		coin    Coin
		other   Coin
		wantGte bool
	}{
		"greater by fraction": {
			coin:    NewCoin(1, 1, "DOGE"),
			other:   NewCoin(1, 0, "DOGE"),
			wantGte: true,
		},
		"greater by whole": {
			coin:    NewCoin(2, 0, "DOGE"),
			other:   NewCoin(1, 0, "DOGE"),
			wantGte: true,
		},
		"equal": {
			coin:    NewCoin(1, 2, "DOGE"),
			other:   NewCoin(1, 2, "DOGE"),
			wantGte: true,
		},
		"different type": {
			coin:    NewCoin(1, 2, "DOGE"),
			other:   NewCoin(1, 2, "BTC"),
			wantGte: false,
		},
		"less than": {
			coin:    NewCoin(0, 2, "DOGE"),
			other:   NewCoin(1, 2, "DOGE"),
			wantGte: false,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if tc.coin.IsGTE(tc.other) != tc.wantGte {
				t.Errorf("want greaterequal = %v", tc.wantGte)
			}
		})
	}
}

func TestCoinSubtract(t *testing.T) {
	cases := []struct {
		a, b Coin
		want Coin
	}{
		{a: NewCoin(3, 0, "X"), b: NewCoin(1, 0, "X"), want: NewCoin(2, 0, "X")},
		{a: NewCoin(1, 0, "X"), b: NewCoin(1, 0, "X"), want: NewCoin(0, 0, "X")},
		{a: NewCoin(1, 0, "X"), b: NewCoin(5, 0, "X"), want: NewCoin(-4, 0, "X")},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			res, err := tc.a.Subtract(tc.b)
			if err != nil {
				t.Fatal(err)
			}
			if !res.Equals(tc.want) {
				t.Fatalf("%+v - %+v = %+v", tc.a, tc.b, res)
			}
		})
	}
}

func TestCoinDivide(t *testing.T) {
	cases := map[string]struct {
		total    Coin
		pieces   int64
		wantOne  Coin
		wantRest Coin
		wantErr  *errors.Error
	}{
		"split into one piece": {
			total:    NewCoin(7, 11, "BTC"),
			pieces:   1,
			wantOne:  NewCoin(7, 11, "BTC"),
			wantRest: NewCoin(0, 0, "BTC"),
		},
		"split into two pieces with no rest": {
			total:    NewCoin(4, 0, "BTC"),
			pieces:   2,
			wantOne:  NewCoin(2, 0, "BTC"),
			wantRest: NewCoin(0, 0, "BTC"),
		},
		"split into two pieces with fractional division and no rest": {
			total:    NewCoin(5, 0, "BTC"),
			pieces:   2,
			wantOne:  NewCoin(2, 500000000, "BTC"),
			wantRest: NewCoin(0, 0, "BTC"),
		},
		"split into two pieces with a leftover": {
			total:    NewCoin(0, 3, "BTC"),
			pieces:   2,
			wantOne:  NewCoin(0, 1, "BTC"),
			wantRest: NewCoin(0, 1, "BTC"),
		},
		"split into two pieces with a fractional division and a leftover": {
			total:    NewCoin(1, 0, "BTC"),
			pieces:   3,
			wantOne:  NewCoin(0, 333333333, "BTC"),
			wantRest: NewCoin(0, 1, "BTC"),
		},
		"zero pieces": {
			total:    NewCoin(666, 0, "BTC"),
			pieces:   0,
			wantOne:  NewCoin(0, 0, "BTC"),
			wantRest: NewCoin(0, 0, "BTC"),
			wantErr:  errors.ErrHuman,
		},
		"negative pieces": {
			total:    NewCoin(999, 0, "BTC"),
			pieces:   -1,
			wantOne:  NewCoin(0, 0, "BTC"),
			wantRest: NewCoin(0, 0, "BTC"),
			wantErr:  errors.ErrHuman,
		},
		"split fractional 2 by 3 should return 2 as leftover": {
			total:    NewCoin(0, 2, "BTC"),
			pieces:   3,
			wantOne:  NewCoin(0, 0, "BTC"),
			wantRest: NewCoin(0, 2, "BTC"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			gotOne, gotRest, err := tc.total.Divide(tc.pieces)
			if !gotOne.Equals(tc.wantOne) {
				t.Errorf("got one %v", gotOne)
			}
			if !gotRest.Equals(tc.wantRest) {
				t.Errorf("got rest %v", gotRest)
			}
			if !tc.wantErr.Is(err) {
				t.Errorf("got err %+v", err)
			}
		})
	}
}

func TestCoinMultiply(t *testing.T) {
	cases := map[string]struct {
		coin    Coin
		times   int64
		want    Coin
		wantErr *errors.Error
	}{
		"zero value coin": {
			coin:  NewCoin(0, 0, "DOGE"),
			times: 666,
			want:  NewCoin(0, 0, "DOGE"),
		},
		"simple multiply": {
			coin:  NewCoin(1, 0, "DOGE"),
			times: 3,
			want:  NewCoin(3, 0, "DOGE"),
		},
		"multiply with normalization": {
			coin:  NewCoin(0, FracUnit/2, "DOGE"),
			times: 3,
			want:  NewCoin(1, FracUnit/2, "DOGE"),
		},
		"multiply zero times": {
			coin:  NewCoin(1, 1, "DOGE"),
			times: 0,
			want:  NewCoin(0, 0, "DOGE"),
		},
		"multiply negative times": {
			coin:  NewCoin(1, 1, "DOGE"),
			times: -2,
			want:  NewCoin(-2, -2, "DOGE"),
		},
		"overflow of a negative and a positive value": {
			coin:    NewCoin(math.MaxInt64, 0, "DOGE"),
			times:   -math.MaxInt64,
			wantErr: errors.ErrOverflow,
		},
		"overflow of two negative values": {
			coin:    NewCoin(-math.MaxInt64, 0, "DOGE"),
			times:   -math.MaxInt64,
			wantErr: errors.ErrOverflow,
		},
		"overflow of two positive values": {
			coin:    NewCoin(math.MaxInt64, 0, "DOGE"),
			times:   math.MaxInt64,
			wantErr: errors.ErrOverflow,
		},
		"overflow with a big multiply": {
			coin:    NewCoin(1000, 0, "DOGE"),
			times:   math.MaxInt64 / 10,
			wantErr: errors.ErrOverflow,
		},
		"overflow with a small multiply": {
			coin:    NewCoin(math.MaxInt64/10, 0, "DOGE"),
			times:   1000,
			wantErr: errors.ErrOverflow,
		},
		"overflow when normalizing": {
			coin:    NewCoin(math.MaxInt64-1, math.MaxInt64-1, "DOGE"),
			times:   1,
			wantErr: errors.ErrOverflow,
		},
		"overflow when normalizing 2": {
			coin:  NewCoin(1, 230000000, "DOGE"),
			times: 10,
			want:  NewCoin(12, 300000000, "DOGE"),
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			got, err := tc.coin.Multiply(tc.times)
			if !tc.wantErr.Is(err) {
				t.Logf("got coin: %+v", got)
				t.Fatalf("got error %v", err)
			}
			if !got.Equals(tc.want) {
				t.Fatalf("got %v", got)
			}
		})
	}
}
