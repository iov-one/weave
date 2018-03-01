package app

import (
	"github.com/confio/weave/commands"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
)

// Examples generates some example structs to dump out with testgen
func Examples() []commands.Example {
	acct := &cash.Set{
		Coins: []*x.Coin{
			&x.Coin{Integer: 50000, CurrencyCode: "ETH"},
			&x.Coin{Integer: 150, Fractional: 567000, CurrencyCode: "BTC"},
		},
	}
	return []commands.Example{
		{"account", acct},
	}
}
