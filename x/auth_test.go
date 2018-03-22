package x

import (
	"context"
	"fmt"
	"testing"

	"github.com/confio/weave"
	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	var helper TestHelpers
	_, a := helper.MakeKey()
	_, b := helper.MakeKey()
	_, c := helper.MakeKey()

	cases := []struct {
		auth       Authenticator
		mainSigner weave.Address
		has        weave.Address
		notHave    weave.Address
		all        []weave.Address
	}{
		0: {
			helper.Authenticate(),
			nil,
			nil,
			b,
			nil,
		},
		{
			helper.Authenticate(a),
			a,
			a,
			b,
			[]weave.Address{a},
		},
		{
			ChainAuth(
				helper.Authenticate(b),
				helper.Authenticate(a)),
			b,
			b,
			c,
			[]weave.Address{b, a},
		},
	}

	ctx := context.Background()
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			assert.Equal(t, tc.mainSigner, MainSigner(ctx, tc.auth))
			if tc.has != nil {
				assert.True(t, tc.auth.HasPermission(ctx, tc.has))
			}
			assert.False(t, tc.auth.HasPermission(ctx, tc.notHave))

			all := tc.auth.GetPermissions(ctx)
			assert.Equal(t, tc.all, all)
			assert.True(t, HasAllSigners(ctx, tc.auth, all))
			assert.False(t, HasAllSigners(ctx, tc.auth, append(all, tc.notHave)))
			if len(all) > 0 {
				assert.True(t, HasNSigners(ctx, tc.auth, all, len(all)-1))
				assert.False(t, HasNSigners(ctx, tc.auth, all, len(all)+1))
			}
		})
	}
}
