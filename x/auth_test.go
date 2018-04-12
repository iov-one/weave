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
		mainSigner weave.Permission
		has        weave.Permission
		notHave    weave.Permission
		all        []weave.Permission
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
			[]weave.Permission{a},
		},
		{
			ChainAuth(
				helper.Authenticate(b),
				helper.Authenticate(a)),
			b,
			b,
			c,
			[]weave.Permission{b, a},
		},
	}

	ctx := context.Background()
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			assert.Equal(t, tc.mainSigner, MainSigner(ctx, tc.auth))
			if tc.has != nil {
				assert.True(t, tc.auth.HasAddress(ctx, tc.has.Address()))
			}
			assert.False(t, tc.auth.HasAddress(ctx, tc.notHave.Address()))

			all := tc.auth.GetPermissions(ctx)
			assert.Equal(t, tc.all, all)
			assert.True(t, HasAllPermissions(ctx, tc.auth, all))
			assert.False(t, HasAllPermissions(ctx, tc.auth, append(all, tc.notHave)))
			if len(all) > 0 {
				assert.True(t, HasNPermissions(ctx, tc.auth, all, len(all)-1))
				assert.False(t, HasNPermissions(ctx, tc.auth, all, len(all)+1))
			}
		})
	}
}
