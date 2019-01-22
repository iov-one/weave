package validators

import (
	"github.com/tendermint/tendermint/crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	keyEd25519 := ed25519.GenPrivKey().PubKey().(ed25519.PubKeyEd25519)
	var anUpdate = []*ValidatorUpdate{
		{Pubkey: Pubkey{Data: keyEd25519[:], Type: "ed25519"}, Power: 10},
	}

	specs := map[string]struct {
		src      SetValidatorsMsg
		expError bool
	}{
		"all good": {
			src:      SetValidatorsMsg{anUpdate},
			expError: false,
		},
		//"pubKey data too short": {
		//	src:      SetValidatorsMsg{anUpdate},
		//	expError: true,
		//},
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
