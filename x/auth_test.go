package x

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestAuth(t *testing.T) {
	a := weavetest.NewCondition()
	b := weavetest.NewCondition()
	c := weavetest.NewCondition()

	ctx1 := &weavetest.CtxAuth{Key: "foo"}
	ctx2 := &weavetest.CtxAuth{Key: "bar"}

	cases := map[string]struct {
		ctx          weave.Context
		auth         Authenticator
		mainSigner   weave.Condition
		wantInCtx    weave.Condition
		wantNotInCtx weave.Condition
		wantAll      []weave.Condition
	}{
		"empty context": {
			ctx:          context.Background(),
			auth:         &weavetest.Auth{},
			wantNotInCtx: b,
		},
		"signer a": {
			ctx:          context.Background(),
			auth:         &weavetest.Auth{Signer: a},
			mainSigner:   a,
			wantInCtx:    a,
			wantNotInCtx: b,
			wantAll:      []weave.Condition{a},
		},
		"signer b": {
			ctx: context.Background(),
			auth: ChainAuth(
				&weavetest.Auth{Signer: b},
				&weavetest.Auth{Signer: a}),
			mainSigner:   b,
			wantInCtx:    b,
			wantNotInCtx: c,
			wantAll:      []weave.Condition{b, a},
		},
		"ctxAuth checks what is set by same key": {
			ctx:          ctx1.SetConditions(context.Background(), a, b),
			auth:         ctx1,
			mainSigner:   a,
			wantInCtx:    b,
			wantNotInCtx: c,
			wantAll:      []weave.Condition{a, b},
		},
		"ctxAuth with different key sees nothing": {
			ctx:          ctx1.SetConditions(context.Background(), a, b),
			auth:         ctx2,
			wantNotInCtx: a,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			assert.Equal(t, tc.mainSigner, MainSigner(tc.ctx, tc.auth))
			if tc.wantInCtx != nil && !tc.auth.HasAddress(tc.ctx, tc.wantInCtx.Address()) {
				t.Fatal("condition address that was expected in context not found")
			}

			if tc.wantNotInCtx != nil && tc.auth.HasAddress(tc.ctx, tc.wantNotInCtx.Address()) {
				t.Fatal("condition address that was expected not to be in context found")
			}

			all := tc.auth.GetConditions(tc.ctx)
			assert.Equal(t, tc.wantAll, all)

			if !HasAllConditions(tc.ctx, tc.auth, all) {
				t.Fatal("has all conditions check failed")
			}
			if HasAllConditions(tc.ctx, tc.auth, append(all, tc.wantNotInCtx)) {
				t.Fatal("has all condition succeeded after adding non existing condition")
			}

			if len(all) > 0 {
				if !HasNConditions(tc.ctx, tc.auth, all, len(all)-1) {
					t.Fatal("want condition check of a subset to succeed")
				}
				if HasNConditions(tc.ctx, tc.auth, all, len(all)+1) {
					t.Fatal("want condition check of a superset to fail")
				}
			}
		})
	}
}
