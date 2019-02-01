package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.NoError(t, err)

			cc := fmt.Sprintf(`"ticker":"%s"`, tc.cur)
			assert.Contains(t, string(val), cc)

			ca := fmt.Sprintf(`"address":"%s"`, tc.addr)
			if tc.addr == "" {
				// we just know there is an address, not what it is
				ca = ca[:len(ca)-1]
			}
			assert.Contains(t, string(val), ca)
		})
	}
}
