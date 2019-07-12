package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/gov"
)

func TestSeqFlag(t *testing.T) {
	cases := map[string]struct {
		setup     func(fl *flag.FlagSet) *flagseq
		args      []string
		wantDie   int
		wantError bool
		wantVal   []byte
	}{
		"use default value, decimal representation": {
			setup: func(fl *flag.FlagSet) *flagseq {
				return flSeq(fl, "x", "1", "")
			},
			args:    []string{},
			wantDie: 0,
			wantVal: sequenceID(1),
		},
		"parse decimal representation": {
			setup: func(fl *flag.FlagSet) *flagseq {
				return flSeq(fl, "x", "1", "")
			},
			args:    []string{"-x", "123"},
			wantDie: 0,
			wantVal: sequenceID(123),
		},
		"parse hex representation": {
			setup: func(fl *flag.FlagSet) *flagseq {
				return flSeq(fl, "x", "1", "")
			},
			args:    []string{"-x", "hex:" + hex.EncodeToString(sequenceID(987654))},
			wantDie: 0,
			wantVal: sequenceID(987654),
		},
		"parse base64 representation": {
			setup: func(fl *flag.FlagSet) *flagseq {
				return flSeq(fl, "x", "1", "")
			},
			args:    []string{"-x", "base64:" + base64.StdEncoding.EncodeToString(sequenceID(987654))},
			wantDie: 0,
			wantVal: sequenceID(987654),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			cnt, cleanup := observeFlagDie(t)
			defer cleanup()

			fl := flag.NewFlagSet("", flag.ContinueOnError)
			fl.SetOutput(ioutil.Discard)
			val := tc.setup(fl)
			err := fl.Parse(tc.args)
			if !tc.wantError {
				assert.Nil(t, err)
			} else if err == nil {
				t.Fatal("Expected error but got none")
			}
			if *cnt != tc.wantDie {
				t.Errorf("want %d flagDie calls, got %d", tc.wantDie, cnt)
			}
			if tc.wantDie == 0 && !bytes.Equal(*val, tc.wantVal) {
				t.Errorf("want %q value, got %q", tc.wantVal, *val)
			}
		})
	}
}

func TestTimeFlag(t *testing.T) {
	now := time.Now()

	cases := map[string]struct {
		setup     func(fl *flag.FlagSet) *flagTime
		args      []string
		wantDie   int
		wantError bool
		wantVal   time.Time
	}{
		"use default value": {
			setup: func(fl *flag.FlagSet) *flagTime {
				return flTime(fl, "x", func() time.Time { return now }, "")
			},
			args:    []string{},
			wantDie: 0,
			wantVal: now,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			cnt, cleanup := observeFlagDie(t)
			defer cleanup()

			fl := flag.NewFlagSet("", flag.ContinueOnError)
			fl.SetOutput(ioutil.Discard)
			val := tc.setup(fl)
			err := fl.Parse(tc.args)
			if !tc.wantError {
				assert.Nil(t, err)
			} else if err == nil {
				t.Fatal("Expected error but got none")
			}
			if *cnt != tc.wantDie {
				t.Errorf("want %d flagDie calls, got %d", tc.wantDie, cnt)
			}
			if tc.wantDie == 0 && !tc.wantVal.Equal(val.Time()) {
				t.Errorf("want %q value, got %q", tc.wantVal, val.Time())
			}
		})
	}
}

func TestHexFlag(t *testing.T) {
	cases := map[string]struct {
		setup     func(fl *flag.FlagSet) *flagbytes
		args      []string
		wantDie   int
		wantError bool
		wantVal   []byte
	}{
		"use default value": {
			setup: func(fl *flag.FlagSet) *flagbytes {
				return flHex(fl, "x", "1111", "")
			},
			args:    []string{},
			wantDie: 0,
			wantVal: fromHex(t, "1111"),
		},
		"use argument value": {
			setup: func(fl *flag.FlagSet) *flagbytes {
				return flHex(fl, "x", "aaaa", "")
			},
			args:    []string{"-x", "11dd"},
			wantDie: 0,
			wantVal: fromHex(t, "11dd"),
		},
		"invalid default value": {
			setup: func(fl *flag.FlagSet) *flagbytes {
				return flHex(fl, "x", "ZZZ", "")
			},
			wantDie: 1,
		},
		"invalid argument value": {
			setup: func(fl *flag.FlagSet) *flagbytes {
				return flHex(fl, "x", "1122", "")
			},
			args:      []string{"-x", "RRR"},
			wantDie:   0,
			wantError: true,
			wantVal:   fromHex(t, "1122"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			cnt, cleanup := observeFlagDie(t)
			defer cleanup()

			fl := flag.NewFlagSet("", flag.ContinueOnError)
			fl.SetOutput(ioutil.Discard)
			val := tc.setup(fl)
			err := fl.Parse(tc.args)
			if !tc.wantError {
				assert.Nil(t, err)
			} else if err == nil {
				t.Fatal("Expected error but got none")
			}
			if *cnt != tc.wantDie {
				t.Errorf("want %d flagDie calls, got %d", tc.wantDie, cnt)
			}
			if tc.wantDie == 0 && !bytes.Equal(*val, tc.wantVal) {
				t.Errorf("want %q value, got %q", tc.wantVal, *val)
			}
		})
	}
}

func TestCoinFlag(t *testing.T) {
	cases := map[string]struct {
		setup     func(fl *flag.FlagSet) *coin.Coin
		args      []string
		wantDie   int
		wantError bool
		wantVal   coin.Coin
	}{
		"use default value": {
			setup: func(fl *flag.FlagSet) *coin.Coin {
				return flCoin(fl, "x", "1 IOV", "")
			},
			args:    []string{},
			wantDie: 0,
			wantVal: coin.NewCoin(1, 0, "IOV"),
		},
		"use argument value": {
			setup: func(fl *flag.FlagSet) *coin.Coin {
				return flCoin(fl, "x", "1 IOV", "")
			},
			args:    []string{"-x", "4 IOV"},
			wantDie: 0,
			wantVal: coin.NewCoin(4, 0, "IOV"),
		},
		"invalid default value": {
			setup: func(fl *flag.FlagSet) *coin.Coin {
				return flCoin(fl, "x", "ZZZ", "")
			},
			wantDie: 1,
		},
		"invalid argument value": {
			setup: func(fl *flag.FlagSet) *coin.Coin {
				return flCoin(fl, "x", "1 IOV", "")
			},
			args:      []string{"-x", "ZZZ"},
			wantDie:   0,
			wantError: true,
			wantVal:   coin.NewCoin(1, 0, "IOV"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			cnt, cleanup := observeFlagDie(t)
			defer cleanup()

			fl := flag.NewFlagSet("", flag.ContinueOnError)
			fl.SetOutput(ioutil.Discard)
			c := tc.setup(fl)
			err := fl.Parse(tc.args)
			if !tc.wantError {
				assert.Nil(t, err)
			} else if err == nil {
				t.Fatal("Expected error but got none")
			}
			if *cnt != tc.wantDie {
				t.Errorf("want %d flagDie calls, got %d", tc.wantDie, cnt)
			}
			if tc.wantDie == 0 && !c.Equals(tc.wantVal) {
				t.Errorf("want %q coin, got %q", tc.wantVal, c)
			}
		})
	}
}

func TestAddressFlag(t *testing.T) {
	cases := map[string]struct {
		setup     func(fl *flag.FlagSet) *weave.Address
		args      []string
		wantDie   int
		wantError bool
		wantVal   weave.Address
	}{
		"use default value": {
			setup: func(fl *flag.FlagSet) *weave.Address {
				return flAddress(fl, "x", "8d0d55645f1241a7a16d84fc9561a51d518c0d36", "")
			},
			args:    []string{},
			wantDie: 0,
			wantVal: fromHex(t, "8d0d55645f1241a7a16d84fc9561a51d518c0d36"),
		},
		"use argument value": {
			setup: func(fl *flag.FlagSet) *weave.Address {
				return flAddress(fl, "x", "aaaaaaa45f1241a7a16d84fc9561a51d518c0d36", "")
			},
			args:    []string{"-x", "8d0d55645f1241a7a16d84fc9561a51d518c0d36"},
			wantDie: 0,
			wantVal: fromHex(t, "8d0d55645f1241a7a16d84fc9561a51d518c0d36"),
		},
		"invalid default value": {
			setup: func(fl *flag.FlagSet) *weave.Address {
				return flAddress(fl, "x", "zzzzzzzzzzzzzz", "")
			},
			wantDie: 1,
		},
		"invalid argument value": {
			setup: func(fl *flag.FlagSet) *weave.Address {
				return flAddress(fl, "x", "8d0d55645f1241a7a16d84fc9561a51d518c0d36", "")
			},
			args:    []string{"-x", "zzzzzzzzzzzzz"},
			wantDie: 0,
			wantVal: fromHex(t, "8d0d55645f1241a7a16d84fc9561a51d518c0d36"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			cnt, cleanup := observeFlagDie(t)
			defer cleanup()

			fl := flag.NewFlagSet("", flag.ContinueOnError)
			fl.SetOutput(ioutil.Discard)
			addr := tc.setup(fl)
			err := fl.Parse(tc.args)
			if !tc.wantError {
				assert.Nil(t, err)
			} else if err == nil {
				t.Fatal("Expected error but got none")
			}
			if *cnt != tc.wantDie {
				t.Errorf("want %d flagDie calls, got %d", tc.wantDie, cnt)
			}
			if tc.wantDie == 0 && !addr.Equals(tc.wantVal) {
				t.Errorf("want %q address, got %q", tc.wantVal, addr)
			}
		})
	}
}

// observeFlagDie returns a pointer to the counter of how many times flagDie
// was called. Until the cleanup function is called, flagDie execution does not
// terminate the program.
func observeFlagDie(t testing.TB) (*int, func()) {
	t.Helper()

	original := flagDie

	var cnt int
	flagDie = func(s string, args ...interface{}) {
		// t.Logf("flagDie called: "+s, args...)
		cnt++
	}
	cleanup := func() {
		flagDie = original
	}
	return &cnt, cleanup
}

func TestFractionFlag(t *testing.T) {
	cases := map[string]struct {
		setup   func(fl *flag.FlagSet) *flagfraction
		args    []string
		wantDie int
		wantVal *gov.Fraction
	}{
		"only numerator": {
			setup: func(fl *flag.FlagSet) *flagfraction {
				return flFraction(fl, "x", "2/3", "")
			},
			args:    []string{"-x", "44"},
			wantDie: 0,
			wantVal: &gov.Fraction{Numerator: 44, Denominator: 1},
		},
		"only 0 numerator": {
			setup: func(fl *flag.FlagSet) *flagfraction {
				return flFraction(fl, "x", "2/3", "")
			},
			args:    []string{"-x", "0"},
			wantDie: 0,
			wantVal: &gov.Fraction{Numerator: 0, Denominator: 0},
		},
		"value is nil when not set": {
			setup: func(fl *flag.FlagSet) *flagfraction {
				return flFraction(fl, "x", "", "")
			},
			args:    []string{},
			wantDie: 0,
			wantVal: nil,
		},
		"zero is value is zero": {
			setup: func(fl *flag.FlagSet) *flagfraction {
				return flFraction(fl, "x", "0", "")
			},
			args:    []string{},
			wantDie: 0,
			wantVal: &gov.Fraction{Numerator: 0, Denominator: 0},
		},
		"use default value": {
			setup: func(fl *flag.FlagSet) *flagfraction {
				return flFraction(fl, "x", "2/3", "")
			},
			args:    []string{},
			wantDie: 0,
			wantVal: &gov.Fraction{Numerator: 2, Denominator: 3},
		},
		"use argument value": {
			setup: func(fl *flag.FlagSet) *flagfraction {
				return flFraction(fl, "x", "2/3", "")
			},
			args:    []string{"-x", "5/7"},
			wantDie: 0,
			wantVal: &gov.Fraction{Numerator: 5, Denominator: 7},
		},
		"invalid default value": {
			setup: func(fl *flag.FlagSet) *flagfraction {
				return flFraction(fl, "x", "invalid", "")
			},
			wantDie: 1,
		},
		"invalid argument value": {
			setup: func(fl *flag.FlagSet) *flagfraction {
				return flFraction(fl, "x", "2/3", "")
			},
			args:    []string{"-x", "invalid"},
			wantDie: 0,
			wantVal: &gov.Fraction{Numerator: 2, Denominator: 3},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			cnt, cleanup := observeFlagDie(t)
			defer cleanup()

			fl := flag.NewFlagSet("", flag.ContinueOnError)
			frac := tc.setup(fl)
			err := fl.Parse(tc.args)
			t.Logf("parse error: %+v", err)
			if *cnt != tc.wantDie {
				t.Errorf("want %d flagDie calls, got %d", tc.wantDie, cnt)
			}
			if tc.wantDie == 0 && !reflect.DeepEqual(frac.frac, tc.wantVal) {
				t.Errorf("want %+v fraction, got %+v", tc.wantVal, frac.frac)
			}
		})
	}
}
