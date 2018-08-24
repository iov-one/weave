package hashlock

import (
	"context"
	"fmt"
	"testing"

	"github.com/confio/weave"
	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	// sig is a signature permission, not a hash
	foo := []byte("foo")
	sig := weave.NewCondition("sigs", "ed25519", foo).Address()
	// other is a permission for some "other" preimage
	other := PreimageCondition(foo).Address()
	random := weave.NewAddress(foo)

	bg := context.Background()
	cases := []struct {
		ctx   weave.Context
		perms []weave.Condition
		match []weave.Address
		not   []weave.Address
	}{
		{bg, nil, nil, []weave.Address{sig, other, random}},
		{
			withPreimage(bg, foo),
			[]weave.Condition{PreimageCondition(foo)},
			[]weave.Address{other},
			[]weave.Address{sig, random},
		},
		{
			withPreimage(bg, []byte("one more time")),
			[]weave.Condition{PreimageCondition([]byte("one more time"))},
			nil,
			[]weave.Address{sig, other, random},
		},
	}

	auth := Authenticate{}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			perms := auth.GetConditions(tc.ctx)
			assert.Equal(t, tc.perms, perms)

			for _, a := range tc.match {
				assert.True(t, auth.HasAddress(tc.ctx, a))
			}

			for _, a := range tc.not {
				assert.False(t, auth.HasAddress(tc.ctx, a))
			}
		})
	}
}
