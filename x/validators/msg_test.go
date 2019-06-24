package validators

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestValidateSetValidatorMsg(t *testing.T) {
	pubkey := weave.PubKey{
		Data: weavetest.NewKey().PublicKey().GetEd25519(),
		Type: "ed25519",
	}

	cases := map[string]struct {
		Msg     weave.Msg
		WantErr *errors.Error
	}{
		"valid model": {
			Msg: &ApplyDiffMsg{
				Metadata: &weave.Metadata{Schema: 1},
				ValidatorUpdates: []weave.ValidatorUpdate{
					{Power: 4, PubKey: pubkey},
					{Power: 3, PubKey: pubkey},
				},
			},
			WantErr: nil,
		},
		"missing metadata": {
			Msg: &ApplyDiffMsg{
				ValidatorUpdates: []weave.ValidatorUpdate{
					{Power: 4, PubKey: pubkey},
					{Power: 3, PubKey: pubkey},
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
