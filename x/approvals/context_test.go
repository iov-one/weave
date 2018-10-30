package approvals

// import (
// 	"context"
// 	"encoding/binary"
// 	"fmt"
// 	"testing"

// 	"github.com/iov-one/weave"
// 	"github.com/stretchr/testify/assert"
// )

// func TestContext(t *testing.T) {
// 	id := func(i int64) []byte {
// 		bz := make([]byte, 8)
// 		binary.BigEndian.PutUint64(bz, uint64(i))
// 		return bz
// 	}

// 	// sig is a signature permission for contractID, not a contract ID
// 	contractID := id(1)
// 	sig := MultiSigCondition(contractID).Address()

// 	// other is a signature permission for some "other" contract ID
// 	otherContractID := id(2)
// 	other := MultiSigCondition(otherContractID).Address()

// 	// random address which does not represent anything in particular
// 	random := weave.NewAddress(id(3))

// 	bg := context.Background()
// 	cases := []struct {
// 		ctx   weave.Context
// 		perms []weave.Condition
// 		match []weave.Address
// 		not   []weave.Address
// 	}{
// 		{bg, nil, nil, []weave.Address{sig, other, random}},
// 		{
// 			withMultisig(bg, contractID),
// 			[]weave.Condition{MultiSigCondition(contractID)},
// 			[]weave.Address{sig},
// 			[]weave.Address{other, random},
// 		},
// 		{
// 			withMultisig(bg, otherContractID),
// 			[]weave.Condition{MultiSigCondition(otherContractID)},
// 			[]weave.Address{other},
// 			[]weave.Address{sig, random},
// 		},
// 		{
// 			// add multisig conditions for both contractID and otherContractID to the context
// 			withMultisig(withMultisig(bg, contractID), otherContractID),
// 			[]weave.Condition{MultiSigCondition(contractID), MultiSigCondition(otherContractID)},
// 			[]weave.Address{sig, other},
// 			[]weave.Address{random},
// 		},
// 		{
// 			withMultisig(bg, id(3)),
// 			[]weave.Condition{MultiSigCondition(id(3))},
// 			nil,
// 			[]weave.Address{sig, other, random},
// 		},
// 	}

// 	auth := Authenticate{}
// 	for i, tc := range cases {
// 		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
// 			perms := auth.GetConditions(tc.ctx)
// 			assert.Equal(t, tc.perms, perms)

// 			for _, a := range tc.match {
// 				assert.True(t, auth.HasAddress(tc.ctx, a))
// 			}

// 			for _, a := range tc.not {
// 				assert.False(t, auth.HasAddress(tc.ctx, a))
// 			}
// 		})
// 	}
// }
