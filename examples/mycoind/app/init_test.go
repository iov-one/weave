package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

func TestGenInitOptions(t *testing.T) {
	cases := []struct {
		args []string
		cur  string
		addr string
	}{
		{nil, "MYC", ""},
		{[]string{"ONE"}, "ONE", ""},
		{[]string{"TWO", "1234567890"}, "TWO", "1234567890"},
		{[]string{"THR", "5238975983695", "FOO"}, "THR", "5238975983695"},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			val, err := GenInitOptions(tc.args)
			assert.Nil(t, err)

			cc := fmt.Sprintf(`"ticker":"%s"`, tc.cur)
			if !strings.Contains(string(val), cc) {
				t.Fatalf("Ticker not in genesis file")
			}

			ca := fmt.Sprintf(`"address":"%s"`, tc.addr)
			if tc.addr == "" {
				// we just know there is an address, not what it is
				ca = ca[:len(ca)-1]
			}
			if !strings.Contains(string(val), ca) {
				t.Fatalf("Address not in genesis file")
			}
		})
	}
}
