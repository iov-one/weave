package app

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/namecoin"
	"github.com/iov-one/weave/x/sigs"
)

// we fix the private keys here for deterministic output with the same encoding
// these are not secure at all, but the only point is to check the format,
// which is easier when everything is reproduceable.
var (
	sender = makePrivKey("1234567890")
	dst    = makePrivKey("F00BA411").PublicKey().Address()
	guest  = makePrivKey("00CAFE00F00D").PublicKey().Address()
)

// makePrivKey repeats the string as long as needed to get 64 digits, then
// parses it as hex. It uses this repeated string as a "random" seed
// for the private key.
//
// nothing random about it, but at least it gives us variety
func makePrivKey(seed string) *crypto.PrivateKey {
	rep := 64/len(seed) + 1
	in := strings.Repeat(seed, rep)[:64]
	bin, err := hex.DecodeString(in)
	if err != nil {
		panic(err)
	}
	return crypto.PrivKeyEd25519FromSeed(bin)
}

// Examples generates some example structs to dump out with testgen
func Examples() []commands.Example {
	wallet := &namecoin.Wallet{
		Metadata: &weave.Metadata{Schema: 1},
		Name:     "example",
		Coins: []*coin.Coin{
			{Whole: 50000, Ticker: "ETH"},
			{Whole: 150, Fractional: 567000, Ticker: "BTC"},
		},
	}

	eth := &coin.Coin{Whole: 50000, Fractional: 12345, Ticker: "ETH"}

	token := &namecoin.Token{
		Metadata: &weave.Metadata{Schema: 1},
		Name:     "My special coin",
		SigFigs:  8,
	}

	pub := sender.PublicKey()
	addr := pub.Address()
	user := &sigs.UserData{
		Metadata: &weave.Metadata{Schema: 1},
		Pubkey:   pub,
		Sequence: 17,
	}

	amt := coin.NewCoin(250, 0, "ETH")
	msg := &cash.SendMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Amount:   &amt,
		Dest:     dst,
		Src:      addr,
		Memo:     "Test payment",
	}

	nameMsg := &namecoin.SetWalletNameMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Address:  addr,
		Name:     "myname",
	}

	tokenMsg := &namecoin.NewTokenMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Ticker:   "ATM",
		Name:     "At the moment",
		SigFigs:  3,
	}

	unsigned := Tx{
		Sum: &Tx_SendMsg{msg},
	}
	tx := unsigned
	sig, err := sigs.SignTx(sender, &tx, "test-123", 17)
	if err != nil {
		panic(err)
	}
	tx.Signatures = []*sigs.StdSignature{sig}

	registerUsernameTokenMsg := &username.RegisterUsernameTokenMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Username: "alice*iov",
		Targets: []username.BlockchainAddress{
			{BlockchainID: "myNet", Address: []byte("myChainAddress")},
		},
	}
	registerUsernameTokenTx := &Tx{
		Sum: &Tx_RegisterUsernameTokenMsg{registerUsernameTokenMsg},
	}

	changeUsernameTokenTargetsMsg := &username.ChangeUsernameTokenTargetsMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Username: "alice*iov",
		NewTargets: []username.BlockchainAddress{
			{
				BlockchainID: "myNet",
				Address:      []byte("myChainAddress"),
			},
		},
	}

	fmt.Printf("Address: %s\n", addr)
	return []commands.Example{
		{Filename: "wallet", Obj: wallet},
		{Filename: "coin", Obj: eth},
		{Filename: "token", Obj: token},
		{Filename: "priv_key", Obj: sender},
		{Filename: "pub_key", Obj: pub},
		{Filename: "user", Obj: user},
		{Filename: "send_msg", Obj: msg},
		{Filename: "name_msg", Obj: nameMsg},
		{Filename: "token_msg", Obj: tokenMsg},
		{Filename: "unsigned_tx", Obj: &unsigned},
		{Filename: "signed_tx", Obj: &tx},
		{Filename: "register_username_token_msg", Obj: registerUsernameTokenMsg},
		{Filename: "register_username_token_tx", Obj: registerUsernameTokenTx},
		{Filename: "change_username_token_targets_msg", Obj: changeUsernameTokenTargetsMsg},
	}
}
