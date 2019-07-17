package bnsd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

func TestGenInitOptions(t *testing.T) {
	cases := map[string]struct {
		args []string
		cur  string
		addr string
	}{
		"without args":                          {nil, "IOV", ""},
		"with currency only":                    {[]string{"ONE"}, "ONE", ""},
		"with currency and address":             {[]string{"TWO", "1234567890"}, "TWO", "1234567890"},
		"with currency, address and random arg": {[]string{"THR", "5238975983695", "FOO"}, "THR", "5238975983695"},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			val, err := GenInitOptions(tc.args)
			assert.Nil(t, err)

			cc := fmt.Sprintf(`"ticker": "%s"`, tc.cur)
			assert.Equal(t, true, strings.Contains(string(val), cc))

			ca := fmt.Sprintf(`"address": "%s"`, tc.addr)
			if tc.addr == "" {
				// we just know there is an address, not what it is
				ca = ca[:len(ca)-1]
			}
			assert.Equal(t, true, strings.Contains(string(val), ca))
		})
	}
}
