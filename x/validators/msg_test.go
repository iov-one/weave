package validators

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestValidateSetValidatorMsg(t *testing.T) {
	pubkey := Pubkey{
		Data: weavetest.NewKey().PublicKey().GetEd25519(),
		Type: "ed25519",
	}

	cases := map[string]struct {
		Msg     weave.Msg
		WantErr *errors.Error
	}{
		"valid model": {
			Msg: &SetValidatorsMsg{
				Metadata: &weave.Metadata{Schema: 1},
				ValidatorUpdates: []*ValidatorUpdate{
					{Power: 4, Pubkey: pubkey},
					{Power: 3, Pubkey: pubkey},
				},
			},
			WantErr: nil,
		},
		"missing metadata": {
			Msg: &SetValidatorsMsg{
				ValidatorUpdates: []*ValidatorUpdate{
					{Power: 4, Pubkey: pubkey},
					{Power: 3, Pubkey: pubkey},
				},
			},
			WantErr: errors.ErrMetadata,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Msg.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected validation error: %s", err)
			}
		})
	}

}
