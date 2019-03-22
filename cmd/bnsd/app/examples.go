package app

import (
	"fmt"

	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/namecoin"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/sigs"
)

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

	priv := crypto.GenPrivKeyEd25519()
	pub := priv.PublicKey()
	addr := pub.Address()
	user := &sigs.UserData{
		Pubkey:   pub,
		Sequence: 17,
	}

	dst := crypto.GenPrivKeyEd25519().PublicKey().Address()
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
	sig, err := sigs.SignTx(priv, &tx, "test-123", 17)
	if err != nil {
		panic(err)
	}
	tx.Signatures = []*sigs.StdSignature{sig}

	guest := crypto.GenPrivKeyEd25519().PublicKey().Address()
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
		{Filename: "priv_key", Obj: priv},
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
