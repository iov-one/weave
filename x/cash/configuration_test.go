package cash

import (
	"testing"

	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestConfigurationHandler(t *testing.T) {
	owner := weavetest.NewCondition()
	ownerAddr := owner.Address()
	other := weavetest.NewCondition()
	otherAddr := other.Address()

	pkg := "cash"

	cases := map[string]struct {
		init     Configuration
		auth     weave.Condition
		update   UpdateConfigurationMsg
		expected Configuration
	}{
		"set all fields": {
			init: Configuration{
				Owner:            ownerAddr,
				CollectorAddress: otherAddr,
				MinimalFee:       coin.NewCoin(0, 20, "IOV"),
			},
			auth: owner,
			update: UpdateConfigurationMsg{
				Patch: &Configuration{
					Owner:            otherAddr,
					CollectorAddress: ownerAddr,
					MinimalFee:       coin.NewCoin(0, 40, "ETH"),
				},
			},
			expected: Configuration{
				Owner:            otherAddr,
				CollectorAddress: ownerAddr,
				MinimalFee:       coin.NewCoin(0, 40, "ETH"),
			},
		},
		"some empty fields": {
			init: Configuration{
				Owner:            ownerAddr,
				CollectorAddress: otherAddr,
				MinimalFee:       coin.NewCoin(0, 20, "IOV"),
			},
			auth: owner,
			update: UpdateConfigurationMsg{
				Patch: &Configuration{
					MinimalFee: coin.NewCoin(0, 40, "ETH"),
				},
			},
			expected: Configuration{
				Owner:            ownerAddr,
				CollectorAddress: otherAddr,
				// only change one field
				MinimalFee: coin.NewCoin(0, 40, "ETH"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			auth := &weavetest.Auth{Signer: tc.auth}
			h := NewConfigHandler(auth)

			kv := store.MemStore()
			// store initial data
			err := gconf.Save(kv, pkg, &tc.init)
			assert.Nil(t, err)

			// should be the same
			var load Configuration
			err = gconf.Load(kv, pkg, &load)
			assert.Nil(t, err)
			assert.Equal(t, tc.init, load)

			// call deliver
			_, err = h.Deliver(nil, kv, &weavetest.Tx{Msg: &tc.update})
			assert.Nil(t, err)

			// should update stored config
			var final Configuration
			err = gconf.Load(kv, pkg, &final)
			assert.Nil(t, err)
			assert.Equal(t, tc.expected, final)
		})
	}

}
