package hashlock

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestContext(t *testing.T) {
	// sig is a signature permission, not a hash
	foo := []byte("foo")
	sig := weave.NewCondition("sigs", "ed25519", foo).Address()
	// other is a permission for some "other" preimage
	other := PreimageCondition(foo).Address()
	random := weave.NewAddress(foo)

	bg := context.Background()
	cases := map[string]struct {
		ctx   weave.Context
		perms []weave.Condition
		match []weave.Address
		not   []weave.Address
	}{
		"empty context": {
			ctx: bg,
			not: []weave.Address{sig, other, random},
		},
		"context with a preimage": {
			ctx:   withPreimage(bg, foo),
			perms: []weave.Condition{PreimageCondition(foo)},
			match: []weave.Address{other},
			not:   []weave.Address{sig, random},
		},
		"context with a preimage 2": {
			ctx:   withPreimage(bg, []byte("one more time")),
			perms: []weave.Condition{PreimageCondition([]byte("one more time"))},
			not:   []weave.Address{sig, other, random},
		},
	}

	auth := Authenticate{}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			assert.Equal(t, tc.perms, auth.GetConditions(tc.ctx))

			for _, a := range tc.match {
				if !auth.HasAddress(tc.ctx, a) {
					t.Fatalf("address %q was not present", a)
				}
			}

			for _, a := range tc.not {
				if auth.HasAddress(tc.ctx, a) {
					t.Fatalf("address %q must not be present", a)
				}
			}
		})
	}
}
