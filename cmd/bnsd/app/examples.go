package app

import (
	"fmt"

	"github.com/iov-one/weave/commands"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/namecoin"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/username"
	"github.com/iov-one/weave/x/sigs"
)

// Examples generates some example structs to dump out with testgen
func Examples() []commands.Example {
	wallet := &namecoin.Wallet{
		Name: "example",
		Coins: []*x.Coin{
			&x.Coin{Whole: 50000, Ticker: "ETH"},
			&x.Coin{Whole: 150, Fractional: 567000, Ticker: "BTC"},
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
	amt := x.NewCoin(250, 0, "ETH")
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
		Id:    []byte("alice@example.com"),
		Owner: addr,
		Details: username.TokenDetails{
			Addresses: []username.ChainAddress{
				{[]byte("myNet"), []byte("myChainAddress")},
			},
		},
		Approvals: []nft.ActionApprovals{
			{"update", []nft.Approval{
				{guest, nft.ApprovalOptions{Count: nft.UnlimitedCount}},
			}},
		},
	}
	issueUsernameTx := &Tx{
		Sum: &Tx_IssueUsernameNftMsg{issueUsernameMsg},
	}

	addAddressMsg := &username.AddChainAddressMsg{
		Id:      []byte("alice@example.com"),
		ChainID: []byte("myNet"),
		Address: []byte("myChainAddress"),
	}

	issueBlockchainMsg := &blockchain.IssueTokenMsg{
		Id:      []byte("test-chain-123456"),
		Owner:   addr,
		Details: blockchain.TokenDetails{},
	}
	issueBlockchainTx := &Tx{
		Sum: &Tx_IssueBlockchainNftMsg{issueBlockchainMsg},
	}

	fmt.Printf("Address: %s\n", addr)
	return []commands.Example{
		{"wallet", wallet},
		{"token", token},
		{"priv_key", priv},
		{"pub_key", pub},
		{"user", user},
		{"send_msg", msg},
		{"name_msg", nameMsg},
		{"token_msg", tokenMsg},
		{"unsigned_tx", &unsigned},
		{"signed_tx", &tx},
		{"issue_username_msg", issueUsernameMsg},
		{"issue_username_tx", issueUsernameTx},
		{"issue_blockchain_msg", issueBlockchainMsg},
		{"issue_blockchain_tx", issueBlockchainTx},
		{"add_addr_msg", addAddressMsg},
	}
}
