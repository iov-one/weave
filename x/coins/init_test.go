package coins

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitState(t *testing.T) {
	// test data
	addr := []byte("12345678901234567890")
	coins := mustNewSet(NewCoin(100, 5, "ATM"))
	accts := []GenesisAccount{{Address: addr, Set: coins}}

	bz, err := json.Marshal(accts)
	require.NoError(t, err)

	cases := [...]struct {
		opts    weave.Options
		isError bool
		acct    []byte
		wallet  Set
	}{
		// no prob if no data
		0: {weave.Options{}, false, nil, Set{}},
		1: {weave.Options{"foo": []byte(`"bar"`)}, false, nil, Set{}},
		// bad address
		2: {weave.Options{"coins": []byte(`[{"address": "1234"}]`)}, true, nil, Set{}},
		// get a real account
		3: {weave.Options{"coins": bz}, false, addr, coins},
	}

	init := Initializer{}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			kv := store.MemStore()
			err := init.InitState(tc.opts, kv)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.acct != nil {
				acct := GetWallet(kv, NewKey(tc.acct))
				if assert.NotNil(t, acct) {
					assert.Equal(t, tc.wallet, acct.Set)
				}
			}
		})
	}

}
