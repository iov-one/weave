package x

import (
	"context"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	var helper TestHelpers
	_, a := helper.MakeKey()
	_, b := helper.MakeKey()
	_, c := helper.MakeKey()

	ctx1 := helper.CtxAuth("foo")
	ctx2 := helper.CtxAuth("bar")

	cases := []struct {
		ctx        weave.Context
		auth       Authenticator
		mainSigner weave.Condition
		has        weave.Condition
		notHave    weave.Condition
		all        []weave.Condition
	}{
		0: {
			context.Background(),
			helper.Authenticate(),
			nil,
			nil,
			b,
			nil,
		},
		{
			context.Background(),
			helper.Authenticate(a),
			a,
			a,
			b,
			[]weave.Condition{a},
		},
		{
			context.Background(),
			ChainAuth(
				helper.Authenticate(b),
				helper.Authenticate(a)),
			b,
			b,
			c,
			[]weave.Condition{b, a},
		},
		// ctxAuth checks what is set by same key
		{
			ctx1.SetConditions(context.Background(), a, b),
			ctx1,
			a,
			b,
			c,
			[]weave.Condition{a, b},
		},
		// ctxAuth with different key sees nothing
		{
			ctx1.SetConditions(context.Background(), a, b),
			ctx2,
			nil,
			nil,
			a,
			nil,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			ctx := tc.ctx
			assert.Equal(t, tc.mainSigner, MainSigner(ctx, tc.auth))
			if tc.has != nil {
				assert.True(t, tc.auth.HasAddress(ctx, tc.has.Address()))
			}
			assert.False(t, tc.auth.HasAddress(ctx, tc.notHave.Address()))

			all := tc.auth.GetConditions(ctx)
			assert.Equal(t, tc.all, all)
			assert.True(t, HasAllConditions(ctx, tc.auth, all))
			assert.False(t, HasAllConditions(ctx, tc.auth, append(all, tc.notHave)))
			if len(all) > 0 {
				assert.True(t, HasNConditions(ctx, tc.auth, all, len(all)-1))
				assert.False(t, HasNConditions(ctx, tc.auth, all, len(all)+1))
			}
		})
	}
}
