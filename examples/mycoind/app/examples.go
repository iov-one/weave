package app

import (
	"github.com/confio/weave/commands"
	"github.com/confio/weave/crypto"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
	"github.com/confio/weave/x/sigs"
)

// Examples generates some example structs to dump out with testgen
func Examples() []commands.Example {
	wallet := &cash.Set{
		Coins: []*x.Coin{
			{Whole: 50000, Ticker: "ETH"},
			{Whole: 150, Fractional: 567000, Ticker: "BTC"},
		},
	}

	priv := crypto.GenPrivKeyEd25519()
	pub := priv.PublicKey()
	user := &sigs.UserData{
		PubKey:   pub,
		Sequence: 17,
	}

	dst := crypto.GenPrivKeyEd25519().PublicKey().Permission().Address()
	amt := x.NewCoin(250, 0, "ETH")
	msg := &cash.SendMsg{
		Amount: &amt,
		Dest:   dst,
		Src:    pub.Permission().Address(),
		Memo:   "Test payment",
	}

	unsigned := Tx{
		Sum: &Tx_SendMsg{msg},
	}
	tx := unsigned
	sig, err := sigs.SignTx(priv, &tx, "test-123", 17)
	if err != nil {
		panic(err)
	}
	tx.Signatures = []*sigs.StdSignature{sig}

	return []commands.Example{
		{"wallet", wallet},
		{"priv_key", priv},
		{"pub_key", pub},
		{"user", user},
		{"send_msg", msg},
		{"unsigned_tx", &unsigned},
		{"signed_tx", &tx},
	}
}
