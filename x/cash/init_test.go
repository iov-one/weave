package cash

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestInitState(t *testing.T) {
	// test data
	addr := []byte("12345678901234567890")
	coins := Set{Coins: mustCombineCoins(coin.NewCoin(100, 5, "ATM"), coin.NewCoin(50, 0, "ETH"))}
	accts := []GenesisAccount{{Address: addr, Set: coins}}

	bz, err := json.Marshal(accts)
	assert.Nil(t, err)

	// hardcode
	bz2 := []byte(`[{"address":"0102030405060708090021222324252627282930",
                "coins":[{"whole":50,
                "fractional":1234567,
                "ticker":"FOO"
              }]}]`)
	coins2 := Set{Coins: mustCombineCoins(coin.NewCoin(50, 1234567, "FOO"))}
	addr2 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x30}

	// use a valid configuration so it doesn't all fail
	config := map[string]interface{}{
		"cash": Configuration{
			CollectorAddress: weave.NewAddress([]byte("foo")),
			MinimalFee:       coin.NewCoin(0, 20, "IOV"),
		},
	}
	rawConfig, err := json.Marshal(config)
	assert.Nil(t, err)

	badConfig := map[string]interface{}{
		"cash": Configuration{
			MinimalFee: coin.NewCoin(0, 20, "food"),
		},
	}
	rawInvalid, err := json.Marshal(badConfig)
	assert.Nil(t, err)

	cases := map[string]struct {
		opts    weave.Options
		isError bool
		acct    []byte
		wallet  Set
	}{
		"no prob if no data":       {weave.Options{"conf": rawConfig}, false, nil, Set{}},
		"but need the config":      {weave.Options{}, true, nil, Set{}},
		"enforces valid config":    {weave.Options{"conf": rawInvalid}, true, nil, Set{}},
		"ignore random key":        {weave.Options{"foo": []byte(`"bar"`), "conf": rawConfig}, false, nil, Set{}},
		"unknown key":              {weave.Options{"foo": []byte(`[{"address": "1234"}]`), "conf": rawConfig}, false, nil, Set{}},
		"bad address":              {weave.Options{"cash": []byte(`[{"coins": 123}]`), "conf": rawConfig}, true, nil, Set{}},
		"get a real account":       {weave.Options{"cash": bz, "conf": rawConfig}, false, addr, coins},
		"get another real account": {weave.Options{"cash": bz2, "conf": rawConfig}, false, addr2, coins2},
	}

	init := Initializer{}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			kv := store.MemStore()
			migration.MustInitPkg(kv, "cash")
			bucket := NewBucket()
			err := init.FromGenesis(tc.opts, weave.GenesisParams{}, kv)
			if tc.isError {
				assert.Equal(t, true, err != nil)
			} else {
				assert.Nil(t, err)
			}

			if tc.acct != nil {
				acct, err := bucket.Get(kv, tc.acct)
				assert.Nil(t, err)
				assert.Equal(t, true, acct != nil)
				for i := range tc.wallet.Coins {
					assert.Equal(t, tc.wallet.Coins[i], AsCoins(acct)[i])
				}
			}
		})
	}
}

// mustCombineCoins has one return value for tests...
func mustCombineCoins(cs ...coin.Coin) coin.Coins {
	s, err := coin.CombineCoins(cs...)
	if err != nil {
		panic(err)
	}
	return s
}
