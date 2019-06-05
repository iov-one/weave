package app

import (
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/namecoin"
	"github.com/iov-one/weave/x/sigs"
)

// Examples generates some example structs to dump out with testgen
func Examples() []commands.Example {
	wallet := &namecoin.Wallet{
		Name: "example",
		Coins: []*coin.Coin{
			{Whole: 50000, Ticker: "ETH"},
			{Whole: 150, Fractional: 567000, Ticker: "BTC"},
		},
	}

	token := &namecoin.Token{
		Name:    "My special coin",
		SigFigs: 8,
	}

	priv := crypto.GenPrivKeyEd25519()
	pub := priv.PublicKey()
	user := &sigs.UserData{
		Pubkey:   pub,
		Sequence: 17,
	}

	dst := crypto.GenPrivKeyEd25519().PublicKey().Address()
	amt := coin.NewCoin(250, 0, "ETH")
	msg := &cash.SendMsg{
		Amount: &amt,
		Dest:   dst,
		Src:    pub.Address(),
		Memo:   "Test payment",
	}

	nameMsg := &namecoin.SetWalletNameMsg{
		Address: pub.Address(),
		Name:    "myname",
	}

	tokenMsg := &namecoin.NewTokenMsg{
		Ticker:  "ATM",
		Name:    "At the moment",
		SigFigs: 3,
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
		{Filename: "wallet", Obj: wallet},
		{Filename: "token", Obj: token},
		{Filename: "priv_key", Obj: priv},
		{Filename: "pub_key", Obj: pub},
		{Filename: "user", Obj: user},
		{Filename: "send_msg", Obj: msg},
		{Filename: "name_msg", Obj: nameMsg},
		{Filename: "token_msg", Obj: tokenMsg},
		{Filename: "unsigned_tx", Obj: &unsigned},
		{Filename: "signed_tx", Obj: &tx},
	}
}
