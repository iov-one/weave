package app

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/namecoin"
	"github.com/iov-one/weave/x/nft"
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

// this repeats the string as long as needed to get 64 digits, then
// parses it as hex. It uses this as a "random" seed for the private key.
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
		Name: "example",
		Coins: []*coin.Coin{
			&coin.Coin{Whole: 50000, Ticker: "ETH"},
			&coin.Coin{Whole: 150, Fractional: 567000, Ticker: "BTC"},
		},
	}

	token := &namecoin.Token{
		Name:    "My special coin",
		SigFigs: 8,
	}

	pub := sender.PublicKey()
	addr := pub.Address()
	user := &sigs.UserData{
		Pubkey:   pub,
		Sequence: 17,
	}

	amt := coin.NewCoin(250, 0, "ETH")
	msg := &cash.SendMsg{
		Amount: &amt,
		Dest:   dst,
		Src:    addr,
		Memo:   "Test payment",
	}

	nameMsg := &namecoin.SetWalletNameMsg{
		Address: addr,
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
	sig, err := sigs.SignTx(sender, &tx, "test-123", 17)
	if err != nil {
		panic(err)
	}
	tx.Signatures = []*sigs.StdSignature{sig}

	issueUsernameMsg := &username.IssueTokenMsg{
		ID:    []byte("alice@example.com"),
		Owner: addr,
		Details: username.TokenDetails{
			Addresses: []username.ChainAddress{
				{BlockchainID: []byte("myNet"), Address: "myChainAddress"},
			},
		},
		Approvals: []nft.ActionApprovals{
			{
				Action: "update",
				Approvals: []nft.Approval{
					{Address: guest, Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}},
				},
			},
		},
	}
	issueUsernameTx := &Tx{
		Sum: &Tx_IssueUsernameNftMsg{issueUsernameMsg},
	}

	addAddressMsg := &username.AddChainAddressMsg{
		UsernameID:   []byte("alice@example.com"),
		BlockchainID: []byte("myNet"),
		Address:      "myChainAddress",
	}

	fmt.Printf("Address: %s\n", addr)
	return []commands.Example{
		{Filename: "wallet", Obj: wallet},
		{Filename: "token", Obj: token},
		{Filename: "priv_key", Obj: sender},
		{Filename: "pub_key", Obj: pub},
		{Filename: "user", Obj: user},
		{Filename: "send_msg", Obj: msg},
		{Filename: "name_msg", Obj: nameMsg},
		{Filename: "token_msg", Obj: tokenMsg},
		{Filename: "unsigned_tx", Obj: &unsigned},
		{Filename: "signed_tx", Obj: &tx},
		{Filename: "issue_username_msg", Obj: issueUsernameMsg},
		{Filename: "issue_username_tx", Obj: issueUsernameTx},
		{Filename: "add_addr_msg", Obj: addAddressMsg},
	}
}
