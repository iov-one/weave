package multisig

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestContext(t *testing.T) {
	contract1 := weavetest.SequenceID(1)
	sig1 := MultiSigCondition(contract1).Address()

	contract2 := weavetest.SequenceID(2)
	sig2 := MultiSigCondition(contract2).Address()

	contract3 := weave.NewAddress(weavetest.SequenceID(3))
	sig3 := MultiSigCondition(contract3).Address()

	bg := context.Background()
	cases := map[string]struct {
		ctx        weave.Context
		wantPerms  []weave.Condition
		wantAddr   []weave.Address
		wantNoAddr []weave.Address
	}{
		"empty context": {
			ctx:        bg,
			wantNoAddr: []weave.Address{sig1, sig2, contract3},
		},
		"context with a single contract": {
			ctx: withMultisig(bg, contract1),
			wantPerms: []weave.Condition{
				MultiSigCondition(contract1),
			},
			wantAddr:   []weave.Address{sig1},
			wantNoAddr: []weave.Address{sig2, contract3, sig3},
		},
		"context with a another single contract": {
			ctx: withMultisig(bg, contract2),
			wantPerms: []weave.Condition{
				MultiSigCondition(contract2),
			},
			wantAddr:   []weave.Address{sig2},
			wantNoAddr: []weave.Address{sig1, contract3, sig3},
		},
		"context with two contracts": {
			ctx: withMultisig(withMultisig(bg, contract1), contract2),
			wantPerms: []weave.Condition{
				MultiSigCondition(contract1),
				MultiSigCondition(contract2),
			},
			wantAddr:   []weave.Address{sig1, sig2},
			wantNoAddr: []weave.Address{contract3, sig3},
		},
	}

	var auth Authenticate
	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			wantPerms := auth.GetConditions(tc.ctx)
			assert.Equal(t, tc.wantPerms, wantPerms)

			for _, a := range tc.wantAddr {
				if !auth.HasAddress(tc.ctx, a) {
					t.Errorf("missing address: %q", a)
				}
			}

			for _, a := range tc.wantNoAddr {
				if auth.HasAddress(tc.ctx, a) {
					t.Errorf("unexpected address: %q", a)
				}
			}
		})
	}
}
