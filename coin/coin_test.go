package coin

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestCompareCoin(t *testing.T) {
	cases := map[string]struct {
		a       Coin
		b       Coin
		wantRes int
	}{
		"a greater than b": {
			a:       NewCoin(20, 1234, "ABC"),
			b:       NewCoin(19, 999999999, "ABC"),
			wantRes: 1,
		},
		"a smaller than b": {
			a:       NewCoin(0, -2, "FOO"),
			b:       NewCoin(0, 1, "FOO"),
			wantRes: -1,
		},
		"a greater than b and both negative": {
			a:       NewCoin(-4, -2456, "BAR"),
			b:       NewCoin(-4, -4567, "BAR"),
			wantRes: 1,
		},
		"zero value coins": {
			a:       Coin{},
			b:       Coin{},
			wantRes: 0,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			res := tc.a.Compare(tc.b)
			assert.Equal(t, res, tc.wantRes)
		})
	}
}

func TestCoinNegative(t *testing.T) {
	a := NewCoin(456, 985, "ABC")

	n := a.Negative()

	assert.Equal(t, a.Ticker, n.Ticker)
	assert.Equal(t, a.Whole, -n.Whole)
	assert.Equal(t, a.Fractional, -n.Fractional)

	if nn := a.Negative().Negative(); !a.Equals(nn) {
		t.Fatal("double negation malformed the coin")
	}
}

func TestIsZero(t *testing.T) {
	cases := map[string]struct {
		c    Coin
		want bool
	}{
		"zero": {
			c:    NewCoin(0, 0, "foo"),
			want: true,
		},
		"positive": {
			c:    NewCoin(0, 1, "foo"),
			want: false,
		},
		"negative": {
			c:    NewCoin(0, -1, "foo"),
			want: false,
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			assert.Equal(t, tc.c.IsZero(), tc.want)
		})
	}
}

func TestIsPositive(t *testing.T) {
	cases := map[string]struct {
		c    Coin
		want bool
	}{
		"zero": {
			c:    NewCoin(0, 0, "foo"),
			want: false,
		},
		"positive": {
			c:    NewCoin(0, 1, "foo"),
			want: true,
		},
		"negative": {
			c:    NewCoin(0, -1, "foo"),
			want: false,
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			assert.Equal(t, tc.c.IsPositive(), tc.want)
		})
	}
}

func TestIsNonNegative(t *testing.T) {
	cases := map[string]struct {
		c    Coin
		want bool
	}{
		"zero": {
			c:    NewCoin(0, 0, "foo"),
			want: true,
		},
		"positive": {
			c:    NewCoin(0, 1, "foo"),
			want: true,
		},
		"negative": {
			c:    NewCoin(0, -1, "foo"),
			want: false,
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			assert.Equal(t, tc.c.IsNonNegative(), tc.want)
		})
	}
}

func TestCoinIsPositive(t *testing.T) {
	cases := map[string]struct {
		c    Coin
		want bool
	}{
		"zero value coin": {
			c:    NewCoin(0, 0, "FOO"),
			want: false,
		},
		"negative value coin": {
			c:    NewCoin(1, 2, "FOO").Negative(),
			want: false,
		},
		"positive value coin": {
			c:    NewCoin(1, 2, "FOO"),
			want: true,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if got := tc.c.IsPositive(); got != tc.want {
				t.Fatal("unexpected result")
			}
		})
	}
}

func TestCoinValidationAndNormalization(t *testing.T) {
	cases := map[string]struct {
		coin                 Coin
		wantValErr           *errors.Error
		wantNormalized       Coin
		wantNormalizationErr *errors.Error
		wantNormValErr       *errors.Error
	}{
		"valid coin with a negative fractional": {
			coin:                 NewCoin(0, -100, "DIN"),
			wantValErr:           nil,
			wantNormalized:       NewCoin(0, -100, "DIN"),
			wantNormalizationErr: nil,
			wantNormValErr:       nil,
		},
		"integer and fraction with different sign": {
			coin:                 NewCoin(4, -123456789, "FOO"),
			wantValErr:           errors.ErrState,
			wantNormalized:       NewCoin(3, 876543211, "FOO"),
			wantNormValErr:       nil,
			wantNormalizationErr: nil,
		},
		"invalid ticker": {
			coin:                 NewCoin(1, 2, "eth2"),
			wantValErr:           errors.ErrCurrency,
			wantNormalized:       NewCoin(1, 2, "eth2"),
			wantNormalizationErr: nil,
			wantNormValErr:       errors.ErrCurrency,
		},
		"make sure issuer is maintained throughout": {
			coin:                 NewCoin(2, -1500500500, "ABC"),
			wantValErr:           errors.ErrOverflow,
			wantNormalized:       NewCoin(0, 499499500, "ABC"),
			wantNormalizationErr: nil,
			wantNormValErr:       nil,
		},
		"from negative to positive rollover": {
			coin:                 NewCoin(-1, 1777888111, "ABC"),
			wantValErr:           errors.ErrOverflow,
			wantNormalized:       NewCoin(0, 777888111, "ABC"),
			wantNormalizationErr: nil,
			wantNormValErr:       nil,
		},
		"overflow": {
			coin:                 NewCoin(MaxInt, FracUnit+4, "DIN"),
			wantValErr:           errors.ErrOverflow,
			wantNormalized:       Coin{},
			wantNormalizationErr: errors.ErrOverflow,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.coin.Validate(); !tc.wantValErr.Is(err) {
				t.Fatalf("unexpected coin validation error: %s", err)
			}

			normalized, err := tc.coin.normalize()
			if !tc.wantNormalizationErr.Is(err) {
				t.Fatalf("unexpected normalization error: %s", err)
			}
			if tc.wantNormalizationErr != nil {
				return
			}

			if err := normalized.Validate(); !tc.wantNormValErr.Is(err) {
				t.Fatalf("unexpected normalized coin validation error: %s", err)
			}

			if !tc.wantNormalized.Equals(normalized) {
				t.Fatalf("unexpected normalized coin value: %#v", normalized)
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
			wantErr: errors.ErrCurrency,
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
			wantErr: errors.ErrOverflow,
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
			wantErr: errors.ErrCurrency,
		},
		"adding to non zero coin without a ticker": {
			a:       NewCoin(1, 0, ""),
			b:       NewCoin(1, 0, "DOGE"),
			wantErr: errors.ErrCurrency,
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
	cases := map[string]struct {
		a, b Coin
		want Coin
	}{
		"positive result": {a: NewCoin(3, 0, "X"), b: NewCoin(1, 0, "X"), want: NewCoin(2, 0, "X")},
		"zero result":     {a: NewCoin(1, 0, "X"), b: NewCoin(1, 0, "X"), want: NewCoin(0, 0, "X")},
		"negative result": {a: NewCoin(1, 0, "X"), b: NewCoin(5, 0, "X"), want: NewCoin(-4, 0, "X")},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
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
			wantErr:  errors.ErrInput,
		},
		"negative pieces": {
			total:    NewCoin(999, 0, "BTC"),
			pieces:   -1,
			wantOne:  NewCoin(0, 0, "BTC"),
			wantRest: NewCoin(0, 0, "BTC"),
			wantErr:  errors.ErrInput,
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

func TestCoinDeserialization(t *testing.T) {
	cases := map[string]struct {
		serialized string
		wantErr    bool
		wantCoin   Coin
	}{
		"old format coin, that maps to fields directly": {
			serialized: `{"whole": 1, "fractional": 2, "ticker": "IOV"}`,
			wantCoin:   NewCoin(1, 2, "IOV"),
		},
		"old format coin, only whole": {
			serialized: `{"whole": 1}`,
			wantCoin:   NewCoin(1, 0, ""),
		},
		"old format coin, only fractional": {
			serialized: `{"fractional": 1}`,
			wantCoin:   NewCoin(0, 1, ""),
		},
		"old format coin, only ticker": {
			serialized: `{"ticker": "IOV"}`,
			wantCoin:   NewCoin(0, 0, "IOV"),
		},
		"old format empty coin, that maps to fields directly": {
			serialized: `{}`,
			wantCoin:   NewCoin(0, 0, ""),
		},
		"human readable format, whole without fractional": {
			serialized: `"1IOV"`,
			wantCoin:   NewCoin(1, 0, "IOV"),
		},
		"human readable format, whole without fractional, ticker space separated": {
			serialized: `"1        IOV"`,
			wantCoin:   NewCoin(1, 0, "IOV"),
		},
		"human readable format, whole and fractional": {
			serialized: `"1.000000002IOV"`,
			wantCoin:   NewCoin(1, 2, "IOV"),
		},
		"human readable format, whole and fractional, ticker space separated": {
			serialized: `"1.000000002 IOV"`,
			wantCoin:   NewCoin(1, 2, "IOV"),
		},
		"human readable format, zero whole and fractional": {
			serialized: `"0.000000002IOV"`,
			wantCoin:   NewCoin(0, 2, "IOV"),
		},
		"human readable format, missing whole": {
			serialized: `".0000000002IOV"`,
			wantErr:    true,
		},
		"human readable format, only whole": {
			serialized: `"1"`,
			wantErr:    true,
		},
		"human readable format, missing ticker": {
			serialized: `"1.0000000002"`,
			wantErr:    true,
		},
		"human readable format, only ticker": {
			serialized: `"IOV"`,
			wantErr:    true,
		},
		"human readable format, ticker too short": {
			serialized: `"1 AB"`,
			wantErr:    true,
		},
		"human readable format, ticker too long": {
			serialized: `"1 ABCDE"`,
			wantErr:    true,
		},
		"human readable format, negative value": {
			serialized: `"-4.000000002 IOV"`,
			wantCoin:   NewCoin(4, 2, "IOV").Negative(),
		},
		"human readable format, negative value, no whole": {
			serialized: `"-0.000000002 IOV"`,
			wantCoin:   NewCoin(0, 2, "IOV").Negative(),
		},
		"human readable format, negative zero": {
			serialized: `"-0 IOV"`,
			wantCoin:   NewCoin(0, 0, "IOV").Negative(),
		},
		"human readable format, zero": {
			serialized: `"0 IOV"`,
			wantCoin:   NewCoin(0, 0, "IOV"),
		},
		"human readable format, double negative": {
			serialized: `"--1 IOV"`,
			wantErr:    true,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var got Coin
			if err := json.Unmarshal([]byte(tc.serialized), &got); err != nil {
				if !tc.wantErr {
					t.Fatalf("cannot unmarshal: %s", err)
				}
				return
			}

			if !tc.wantCoin.Equals(got) {
				t.Fatalf("unexpected coin result: %#v", got)
			}
		})
	}
}

func TestCoinString(t *testing.T) {
	cases := map[string]struct {
		c    Coin
		want string
	}{
		"zero coin": {
			c:    Coin{},
			want: "0",
		},
		"zero coin with a ticker": {
			c:    Coin{Ticker: "FOO"},
			want: "0 FOO",
		},
		"one IOV": {
			c:    NewCoin(1, 0, "IOV"),
			want: "1 IOV",
		},
		"fifty IOV": {
			c:    NewCoin(50, 0, "IOV"),
			want: "50 IOV",
		},
		"minus one IOV": {
			c:    NewCoin(1, 0, "IOV").Negative(),
			want: "-1 IOV",
		},
		"minus fifty IOV": {
			c:    NewCoin(50, 0, "IOV").Negative(),
			want: "-50 IOV",
		},
		"an IOV penny": {
			c:    NewCoin(0, FracUnit/100, "IOV"),
			want: "0.01 IOV",
		},
		"one fractional": {
			c:    NewCoin(0, 1, "IOV"),
			want: "0.000000001 IOV",
		},
		"biggest coin": {
			c:    NewCoin(MaxInt, MaxFrac, "IOV"),
			want: "999999999999999.999999999 IOV",
		},
		"smallest coin": {
			c:    NewCoin(MinInt, MinFrac, "IOV"),
			want: "-999999999999999.999999999 IOV",
		},
		"one without a ticker": {
			c:    NewCoin(1, 0, ""),
			want: "1",
		},
		"one and one penny without a ticker": {
			c:    NewCoin(1, 1, ""),
			want: "1.000000001",
		},
		"not normalized": {
			c:    NewCoin(2, int64(102.3*float64(FracUnit)), "FOO"),
			want: "104.3 FOO",
		},
		"whole value overflow": {
			c:    NewCoin(MaxInt+1, 0, "FOO"),
			want: fmt.Sprintf("%d FOO", MaxInt+1),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if got := tc.c.String(); got != tc.want {
				t.Fatalf("unexpected string representation: %q", got)
			}
		})
	}
}
