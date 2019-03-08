package x

import (
	"context"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest"
	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	a := weavetest.NewCondition()
	b := weavetest.NewCondition()
	c := weavetest.NewCondition()

	ctx1 := &weavetest.CtxAuth{Key: "foo"}
	ctx2 := &weavetest.CtxAuth{Key: "bar"}

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
			&weavetest.Auth{},
			nil,
			nil,
			b,
			nil,
		},
		{
			context.Background(),
			&weavetest.Auth{Signer: a},
			a,
			a,
			b,
			[]weave.Condition{a},
		},
		{
			context.Background(),
			ChainAuth(
				&weavetest.Auth{Signer: b},
				&weavetest.Auth{Signer: a}),
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
