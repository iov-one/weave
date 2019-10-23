package weave_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestAddressPrinting(t *testing.T) {
	b := []byte("ABCD123456LHB")
	addr := weave.Address(b)

	if addr.String() == fmt.Sprintf("%X", addr) {
		t.Fatal("address String() is expected to produce a different result than hex print")
	}

	cond := weave.NewCondition("12", "32", []byte("ABCD123456LHB"))

	if cond.String() == fmt.Sprintf("%X", cond) {
		t.Fatal("condition String() is expected to produce a different result than hex print")
	}

}

func TestAddressBech32Printing(t *testing.T) {
	cases := map[string]struct {
		hex    string
		hrp    string
		bech32 string
	}{
		"success lower case hex tiov hrp": {
			hex:    "e774b6e08e3c9ad9d35a7830654db7906b0b02d5",
			hrp:    "tiov",
			bech32: "tiov1ua6tdcyw8jddn5660qcx2ndhjp4skqk4rr48rw",
		},
		"success lower case hex iov hrp": {
			hex:    "e774b6e08e3c9ad9d35a7830654db7906b0b02d5",
			hrp:    "iov",
			bech32: "iov1ua6tdcyw8jddn5660qcx2ndhjp4skqk4dkurrl",
		},
		"success upper case hex tiov hrp": {
			hex:    "C1888F21C55E9EF00B0220C7CAECBE862C4591F0",
			hrp:    "tiov",
			bech32: "tiov1cxyg7gw9t600qzczyrru4m97sckyty0s4cmkzm",
		},
		"success uppeer case hex iov hrp": {
			hex:    "C1888F21C55E9EF00B0220C7CAECBE862C4591F0",
			hrp:    "iov",
			bech32: "iov1cxyg7gw9t600qzczyrru4m97sckyty0smdjjz2",
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			addr, err := weave.ParseAddress(tc.hex)
			assert.Nil(t, err)

			got, err := addr.Bech32String(tc.hrp)
			assert.Nil(t, err)
			assert.Equal(t, tc.bech32, got)
		})
	}
}

func TestAddressUnmarshalJSON(t *testing.T) {
	fromHex := func(s string) []byte {
		b, err := hex.DecodeString(s)
		if err != nil {
			panic(err)
		}
		return b
	}

	cases := map[string]struct {
		json     string
		wantErr  *errors.Error
		wantAddr weave.Address
	}{
		"default decoding": {
			json:     `"8d0d55645f1241a7a16d84fc9561a51d518c0d36"`,
			wantAddr: weave.Address(fromHex("8d0d55645f1241a7a16d84fc9561a51d518c0d36")),
		},
		"hex decoding": {
			json:     `"hex:8d0d55645f1241a7a16d84fc9561a51d518c0d36"`,
			wantAddr: weave.Address(fromHex("8d0d55645f1241a7a16d84fc9561a51d518c0d36")),
		},
		"cond decoding": {
			json:     `"cond:foo/bar/636f6e646974696f6e64617461"`,
			wantAddr: weave.NewCondition("foo", "bar", []byte("conditiondata")).Address(),
		},
		"seq decoding": {
			json:     `"seq:with/seq/12345"`,
			wantAddr: weave.NewCondition("with", "seq", []byte{0, 0, 0, 0, 0, 0, 0x30, 0x39}).Address(),
		},
		"bech32 decoding": {
			json:     `"bech32:tiov135x42ezlzfq60gtdsn7f2cd9r4gccrfk6md5xz"`,
			wantAddr: weave.Address(fromHex("8d0d55645f1241a7a16d84fc9561a51d518c0d36")),
		},
		"invalid condition format": {
			json:    `"cond:foo/636f6e646974696f6e64617461"`,
			wantErr: errors.ErrInput,
		},
		"invalid condition data": {
			json:    `"cond:foo/bar/zzzzz"`,
			wantErr: errors.ErrInput,
		},
		"unknown format": {
			json:    `"foobar:xxx"`,
			wantErr: errors.ErrType,
		},
		"zero address": {
			json:     `""`,
			wantAddr: nil,
		},
		"zero hex address": {
			json:     `"hex:"`,
			wantAddr: nil,
		},
		"zero cond address": {
			json:     `"cond:"`,
			wantAddr: nil,
		},
		"address to short (19 bytes)": {
			json:    `"b339b5f6ae69570a1fd4d6c561c3ec1ce13450"`,
			wantErr: errors.ErrInput,
		},
		"address to long (21 bytes)": {
			json:    `"0a6e36d3553a0abfe7896243386b47b5215cb24312"`,
			wantErr: errors.ErrInput,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var a weave.Address
			err := json.Unmarshal([]byte(tc.json), &a)
			if !tc.wantErr.Is(err) {
				t.Fatalf("got error: %+v", err)
			}
			if err == nil && !reflect.DeepEqual(a, tc.wantAddr) {
				t.Fatalf("got address: %q (want %q)", a, tc.wantAddr)
			}
		})
	}
}

func TestConditionUnmarshalJSON(t *testing.T) {
	cases := map[string]struct {
		json          string
		wantErr       *errors.Error
		wantCondition weave.Condition
	}{
		"default decoding": {
			json:          `"foo/bar/636f6e646974696f6e64617461"`,
			wantCondition: weave.NewCondition("foo", "bar", []byte("conditiondata")),
		},
		"invalid condition format": {
			json:    `"foo/636f6e646974696f6e64617461"`,
			wantErr: errors.ErrInput,
		},
		"invalid condition data": {
			json:    `"foo/bar/zzzzz"`,
			wantErr: errors.ErrInput,
		},
		"zero address": {
			json:          `""`,
			wantCondition: nil,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var got weave.Condition
			err := json.Unmarshal([]byte(tc.json), &got)
			if !tc.wantErr.Is(err) {
				t.Fatalf("got error: %+v", err)
			}
			if err == nil && !got.Equals(tc.wantCondition) {
				t.Fatalf("expected %q but got condition: %q", tc.wantCondition, got)
			}
		})
	}
}

func TestConditionMarshalJSON(t *testing.T) {
	cases := map[string]struct {
		source   weave.Condition
		wantJson string
	}{
		"cond encoding": {
			source:   weave.NewCondition("foo", "bar", []byte("conditiondata")),
			wantJson: `"foo/bar/636F6E646974696F6E64617461"`,
		},
		"nil encoding": {
			source:   nil,
			wantJson: `""`,
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			got, err := json.Marshal(tc.source)
			assert.Nil(t, err)
			assert.Equal(t, tc.wantJson, string(got))
		})
	}
}
