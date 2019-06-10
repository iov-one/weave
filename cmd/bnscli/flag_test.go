package main

import (
	"bytes"
	"flag"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
)

func TestTimeFlag(t *testing.T) {
	now := time.Now()

	cases := map[string]struct {
		setup   func(fl *flag.FlagSet) *flagTime
		args    []string
		wantDie int
		wantVal time.Time
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
			val := tc.setup(fl)
			err := fl.Parse(tc.args)
			t.Logf("parse error: %+v", err)
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
		setup   func(fl *flag.FlagSet) *flagbytes
		args    []string
		wantDie int
		wantVal []byte
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
			args:    []string{"-x", "RRR"},
			wantDie: 0,
			wantVal: fromHex(t, "1122"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			cnt, cleanup := observeFlagDie(t)
			defer cleanup()

			fl := flag.NewFlagSet("", flag.ContinueOnError)
			val := tc.setup(fl)
			err := fl.Parse(tc.args)
			t.Logf("parse error: %+v", err)
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
		setup   func(fl *flag.FlagSet) *coin.Coin
		args    []string
		wantDie int
		wantVal coin.Coin
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
			args:    []string{"-x", "ZZZ"},
			wantDie: 0,
			wantVal: coin.NewCoin(1, 0, "IOV"),
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			cnt, cleanup := observeFlagDie(t)
			defer cleanup()

			fl := flag.NewFlagSet("", flag.ContinueOnError)
			c := tc.setup(fl)
			err := fl.Parse(tc.args)
			t.Logf("parse error: %+v", err)
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
		setup   func(fl *flag.FlagSet) *weave.Address
		args    []string
		wantDie int
		wantVal weave.Address
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
			addr := tc.setup(fl)
			err := fl.Parse(tc.args)
			t.Logf("parse error: %+v", err)
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
		t.Logf("flagDie called: "+s, args...)
		cnt++
	}
	cleanup := func() {
		flagDie = original
	}
	return &cnt, cleanup
}
