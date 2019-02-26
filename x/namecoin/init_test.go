package namecoin

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitState(t *testing.T) {
	// hardcoded-example
	bz := []byte(`{
    "wallets": [{
       "address":"0102030405060708090021222324252627282930",
       "name": "lolz1793",
       "coins":[{"whole":50,
       "fractional":1234567,
       "ticker":"FUN"}]
      }],
    "tokens": [{
      "ticker": "FUN",
      "name": "The most fun coin",
      "sig_figs": 7
    }]
  }`)
	opts := weave.Options{}
	err := json.Unmarshal(bz, &opts)
	require.NoError(t, err)
	// expected values
	addr := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x30}
	wallet := &Wallet{Name: "lolz1793", Coins: mustCombineCoins(coin.NewCoin(50, 1234567, "FUN"))}
	ticker := "FUN"
	token := &Token{Name: "The most fun coin", SigFigs: 7}

	// valid data
	addr2 := []byte("12345678901234567890")
	wallet2 := &Wallet{Coins: mustCombineCoins(coin.NewCoin(100, 5, "ATM"), coin.NewCoin(50, 0, "ETH"))}
	ticker2 := "ATM"
	token2 := &Token{Name: "At the moment", SigFigs: 3}
	ticker2a := "ETH"
	token2a := &Token{Name: "Eat the haters", SigFigs: 9}
	opts2, err := BuildGenesis(
		[]GenesisAccount{{Address: addr2, Wallet: wallet2}},
		[]GenesisToken{ToGenesisToken(ticker2, token2), ToGenesisToken(ticker2a, token2a)})
	require.NoError(t, err)

	// invalid wallet
	badCoin := coin.NewCoin(100, -5000, "ATM")
	walletBad := &Wallet{Coins: []*coin.Coin{&badCoin}}
	opts3, err := BuildGenesis(
		[]GenesisAccount{{Address: addr2, Wallet: walletBad}},
		[]GenesisToken{ToGenesisToken(ticker2, token2)})
	require.NoError(t, err)

	// invalid token
	tickerBad := "LONGER"
	opts4, err := BuildGenesis(
		[]GenesisAccount{{Address: addr2, Wallet: wallet2}},
		[]GenesisToken{ToGenesisToken(tickerBad, token2)})
	require.NoError(t, err)

	cases := []struct {
		opts    weave.Options
		isError bool
		acct    weave.Address
		wallet  *Wallet
		tickers []string
		tokens  []*Token
	}{
		// missing address
		0: {weave.Options{"wallets": []byte(`[{"coins": 123}]`)}, true, nil, nil, nil, nil},
		// hard-coded, ensure it parses
		1: {opts, false, addr, wallet, []string{ticker}, []*Token{token}},
		// hand-built, should pass
		2: {opts2, false, addr2, wallet2,
			[]string{ticker2, ticker2a},
			[]*Token{token2, token2a}},
		// hand-built invalid coins
		3: {opts3, true, nil, nil, nil, nil},
		// hand-built invalid tickers
		4: {opts4, true, nil, nil, nil, nil},
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

			for j, tick := range tc.tickers {
				bucket := NewTokenBucket()
				token, err := bucket.Get(kv, tick)
				require.NoError(t, err, tick)
				if assert.NotNil(t, token, tick) {
					assert.EqualValues(t, tc.tokens[j], AsToken(token), tick)
				}
			}
			// TODO: check tokens
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
