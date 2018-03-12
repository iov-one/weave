package namecoin

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/confio/weave"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
)

func TestInitState(t *testing.T) {
	// test data
	addr := []byte("12345678901234567890")
	coins := Wallet{Coins: mustCombineCoins(x.NewCoin(100, 5, "ATM"), x.NewCoin(50, 0, "ETH").WithIssuer("chain-1"))}
	accts := []GenesisAccount{{Address: addr, Wallet: coins}}

	bz, err := json.Marshal(accts)
	require.NoError(t, err)

	// hardcode
	bz2 := []byte(`[{"address":"0102030405060708090021222324252627282930",
                "name": "lolz1793",
                "coins":[{"whole":50,
                "fractional":1234567,
                "ticker":"FOO"
              }]}]`)
	coins2 := Wallet{Name: "lolz1793", Coins: mustCombineCoins(x.NewCoin(50, 1234567, "FOO"))}
	addr2 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x30}

	cases := [...]struct {
		opts    weave.Options
		isError bool
		acct    []byte
		wallet  *Wallet
	}{
		// no prob if no data
		0: {weave.Options{}, false, nil, nil},
		1: {weave.Options{"foo": []byte(`"bar"`)}, false, nil, nil},
		// bad format
		2: {weave.Options{"foo": []byte(`[{"address": "1234"}]`)}, false, nil, nil},
		// bad address
		3: {weave.Options{"wallets": []byte(`[{"coins": 123}]`)}, true, nil, nil},
		// get a real account
		4: {weave.Options{"wallets": bz}, false, addr, &coins},
		5: {weave.Options{"wallets": bz2}, false, addr2, &coins2},
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
				bucket := NewWalletBucket()
				acct, err := bucket.Get(kv, tc.acct)
				require.NoError(t, err)
				if assert.NotNil(t, acct) {
					assert.EqualValues(t, tc.wallet, AsWallet(acct))
				}
			}
			// TODO: check tokens
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
