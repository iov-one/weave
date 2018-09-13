package multisig

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	id := func(i int64) []byte {
		bz := make([]byte, 8)
		binary.BigEndian.PutUint64(bz, uint64(i))
		return bz
	}

	// sig is a signature permission, not a contract ID
	foo := id(1)
	sig := weave.NewCondition("multisig", "usage", foo).Address()
	// other is a permission for some "other" contract ID
	other := MultiSigCondition(foo).Address()
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
			withMultisig(bg, foo),
			[]weave.Condition{MultiSigCondition(foo)},
			[]weave.Address{sig, other},
			[]weave.Address{random},
		},
		{
			withMultisig(bg, id(2)),
			[]weave.Condition{MultiSigCondition(id(2))},
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
