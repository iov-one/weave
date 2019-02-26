package validators

import (
	"testing"

	"github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	keyEd25519 := ed25519.GenPrivKey().PubKey().(ed25519.PubKeyEd25519)

	specs := map[string]struct {
		src      SetValidatorsMsg
		expError bool
	}{
		"All good": {
			src: SetValidatorsMsg{ValidatorUpdates: []*ValidatorUpdate{
				{Pubkey: Pubkey{Data: keyEd25519[:], Type: "ed25519"}, Power: 10},
			}},
			expError: false,
		},
		"Power can be 0": {
			src: SetValidatorsMsg{ValidatorUpdates: []*ValidatorUpdate{
				{Pubkey: Pubkey{Data: keyEd25519[:], Type: "ed25519"}, Power: 0},
			}},
			expError: false,
		},
		"PubKey data too short": {
			src: SetValidatorsMsg{ValidatorUpdates: []*ValidatorUpdate{
				{Pubkey: Pubkey{Data: []byte("too short"), Type: "ed25519"}, Power: 10},
			}},
			expError: true,
		},
		"Power must not be negative": {
			src: SetValidatorsMsg{ValidatorUpdates: []*ValidatorUpdate{
				{Pubkey: Pubkey{Data: keyEd25519[:], Type: "ed25519"}, Power: -1},
			}},
			expError: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			if spec.expError {
				assert.Error(t, spec.src.Validate())
			} else {
				assert.NoError(t, spec.src.Validate())
			}
		})
	}
}
