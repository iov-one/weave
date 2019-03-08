package weave_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAddressPrinting(t *testing.T) {
	Convey("test hexademical address printing", t, func() {
		b := []byte("ABCD123456LHB")
		addr := weave.Address(b)

		So(addr.String(), ShouldNotEqual, fmt.Sprintf("%X", addr))
	})

	Convey("test hexademical condition printing", t, func() {
		cond := weave.NewCondition("12", "32", []byte("ABCD123456LHB"))

		So(cond.String(), ShouldNotEqual, fmt.Sprintf("%X", cond))
	})
}

func TestAddressUnmarshalJSON(t *testing.T) {
	cases := map[string]struct {
		json     string
		wantErr  error
		wantAddr weave.Address
	}{
		"default decoding": {
			json:     `"6865782d61646472"`,
			wantAddr: weave.Address("hex-addr"),
		},
		"hex decoding": {
			json:     `"hex:6865782d61646472"`,
			wantAddr: weave.Address("hex-addr"),
		},
		"cond decoding": {
			json:     `"cond:foo/bar/636f6e646974696f6e64617461"`,
			wantAddr: weave.NewCondition("foo", "bar", []byte("conditiondata")).Address(),
		},
		"invalid condition format": {
			json:    `"cond:foo/636f6e646974696f6e64617461"`,
			wantErr: errors.ErrInvalidInput,
		},
		"invalid condition data": {
			json:    `"cond:foo/bar/zzzzz"`,
			wantErr: errors.ErrInvalidInput,
		},
		"unknown format": {
			json:    `"foobar:xxx"`,
			wantErr: errors.ErrInvalidType,
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
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var a weave.Address
			err := json.Unmarshal([]byte(tc.json), &a)
			if !errors.Is(tc.wantErr, err) {
				t.Fatalf("got error: %+v", err)
			}
			if err == nil && !reflect.DeepEqual(a, tc.wantAddr) {
				t.Fatalf("got address: %q", a)
			}
		})
	}
}

func TestConditionUnmarshalJSON(t *testing.T) {
	cases := map[string]struct {
		json          string
		wantErr       error
		wantCondition weave.Condition
	}{
		"default decoding": {
			json:          `"666F6F2F6261722F636F6E646974696F6E64617461"`,
			wantCondition: weave.NewCondition("foo", "bar", []byte("conditiondata")),
		},
		"hex decoding": {
			json:          `"hex:666F6F2F6261722F636F6E646974696F6E64617461"`,
			wantCondition: weave.NewCondition("foo", "bar", []byte("conditiondata")),
		},
		"cond decoding": {
			json:          `"cond:foo/bar/636f6e646974696f6e64617461"`,
			wantCondition: weave.NewCondition("foo", "bar", []byte("conditiondata")),
		},
		"invalid condition format": {
			json:    `"cond:foo/636f6e646974696f6e64617461"`,
			wantErr: errors.ErrInvalidInput,
		},
		"invalid condition data": {
			json:    `"cond:foo/bar/zzzzz"`,
			wantErr: errors.ErrInvalidInput,
		},
		"unknown format": {
			json:    `"foobar:xxx"`,
			wantErr: errors.ErrInvalidType,
		},
		"zero address": {
			json:          `""`,
			wantCondition: nil,
		},
		"zero hex address": {
			json:          `"hex:"`,
			wantCondition: nil,
		},
		"zero cond address": {
			json:          `"cond:"`,
			wantCondition: nil,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var got weave.Condition
			err := json.Unmarshal([]byte(tc.json), &got)
			if !errors.Is(tc.wantErr, err) {
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
			wantJson: `"cond:foo/bar/636F6E646974696F6E64617461"`,
		},
		"nil encoding": {
			source:   nil,
			wantJson: `"cond:"`,
		},
	}
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			got, err := json.Marshal(tc.source)
			require.NoError(t, err)
			assert.Equal(t, tc.wantJson, string(got))
		})
	}
}
