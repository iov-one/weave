package cash

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitState(t *testing.T) {
	// test data
	addr := []byte("12345678901234567890")
	coins := Set{mustCombineCoins(x.NewCoin(100, 5, "ATM"), x.NewCoin(50, 0, "ETH").WithIssuer("chain-1"))}
	accts := []GenesisAccount{{Address: addr, Set: coins}}

	bz, err := json.Marshal(accts)
	require.NoError(t, err)

	// hardcode
	bz2 := []byte(`[{"address":"0102030405060708090021222324252627282930",
                "coins":[{"integer":50,
                "fractional":1234567,
                "currency_code":"FOO"
              }]}]`)
	coins2 := Set{mustCombineCoins(x.NewCoin(50, 1234567, "FOO"))}
	addr2 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x30}

	cases := [...]struct {
		opts    weave.Options
		isError bool
		acct    []byte
		wallet  Set
	}{
		// no prob if no data
		0: {weave.Options{}, false, nil, Set{}},
		1: {weave.Options{"foo": []byte(`"bar"`)}, false, nil, Set{}},
		// bad format
		2: {weave.Options{"foo": []byte(`[{"address": "1234"}]`)}, false, nil, Set{}},
		// bad address
		3: {weave.Options{"coins": []byte(`[{"coins": 123}]`)}, true, nil, Set{}},
		// get a real account
		4: {weave.Options{"coins": bz}, false, addr, coins},
		5: {weave.Options{"coins": bz2}, false, addr2, coins2},
	}

	init := Initializer{}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			kv := store.MemStore()
			err := init.FromGenesis(tc.opts, kv)
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

// mustCombineCoins has one return value for tests...
func mustCombineCoins(cs ...x.Coin) x.Coins {
	s, err := x.CombineCoins(cs...)
	if err != nil {
		panic(err)
	}
	return s
}
