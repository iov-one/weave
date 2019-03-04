package weave_test

import (
	"encoding/json"
	"fmt"
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
