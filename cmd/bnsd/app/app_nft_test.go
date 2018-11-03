package app_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x/approvals"

	weave_app "github.com/iov-one/weave/app"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/app/testdata/fixtures"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/ticker"
	"github.com/iov-one/weave/x/nft/username"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestIssueNfts(t *testing.T) {
	appFixture := fixtures.NewApp()
	isuserAddr := appFixture.GenesisKeyAddress
	issuerPrivKey := appFixture.GenesisKey

	myApp := appFixture.Build()
	myBlockchainID := []byte("myblockchain")

	// when blockchain nft issued
	tx := &app.Tx{
		Sum: &app.Tx_IssueBlockchainNftMsg{&blockchain.IssueTokenMsg{
			Id:      myBlockchainID,
			Owner:   isuserAddr,
			Details: blockchain.TokenDetails{Iov: blockchain.IOV{Codec: "asd"}},
		},
		},
	}

	res := signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, 0}}, appFixture.ChainID, 2)

	// then
	require.EqualValues(t, 0, res.Code)

	// and when username nft issued
	tx = &app.Tx{
		Sum: &app.Tx_IssueUsernameNftMsg{&username.IssueTokenMsg{
			Id:    []byte("anybody@example.com"),
			Owner: isuserAddr,
			Addresses: []*username.ChainAddress{
				&username.ChainAddress{ChainID: myBlockchainID, Address: []byte("myChainAddress")},
			}},
		},
	}

	res = signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, 1}}, appFixture.ChainID, 3)

	// then
	require.EqualValues(t, 0, res.Code)

	// and when ticker nft issued
	tx = &app.Tx{
		Sum: &app.Tx_IssueTickerNftMsg{&ticker.IssueTokenMsg{
			Id:      []byte("ANY"),
			Owner:   isuserAddr,
			Details: ticker.TokenDetails{myBlockchainID},
		},
		},
	}

	res = signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, 2}}, appFixture.ChainID, 4)

	// then
	require.EqualValues(t, 0, res.Code, res.Log)

	// // and query a username
	query := abci.RequestQuery{Path: "/nft/usernames/chainaddr", Data: []byte("myChainAddress*myblockchain")}
	qRes := myApp.Query(query)
	require.EqualValues(t, 0, qRes.Code, qRes.Log)
	var actual username.UsernameToken
	err := weave_app.UnmarshalOneResult(qRes.Value, &actual)
	require.NoError(t, err)
	require.Equal(t, []byte("anybody@example.com"), actual.GetId())
}
func TestNftApprovals(t *testing.T) {
	appFixture := fixtures.NewApp()
	issuerAddr := appFixture.GenesisKeyAddress
	issuerPrivKey := appFixture.GenesisKey
	myApp := appFixture.Build()

	// add blockchains
	for i := int64(0); i < 5; i++ {
		tx := newIssueBlockchainNftTx(t, []byte(fmt.Sprintf("blockchain%d", i)), issuerAddr)
		res := signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, i}}, appFixture.ChainID, i+2)
		require.EqualValues(t, 0, res.Code)
	}

	// check approvals
	guest1 := crypto.GenPrivKeyEd25519()
	tx := newIssueUsernameNftTx(t, "anybody@example.com", issuerAddr, []byte("blockchain1"),
		approvals.ApprovalCondition(guest1.PublicKey().Address(), "update"))
	res := signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, 5}}, appFixture.ChainID, 7)
	require.EqualValues(t, 0, res.Code)

	// by owner
	tx = newAddUsernameAddressNftTx(t, "anybody@example.com", []byte("blockchain2"))
	res = signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, 6}}, appFixture.ChainID, 8)
	require.EqualValues(t, 0, res.Code)

	// by approved guest1
	tx = newAddUsernameAddressNftTx(t, "anybody@example.com", []byte("blockchain3"))
	res = signAndCommit(t, myApp, tx, []Signer{{guest1, 0}}, appFixture.ChainID, 9)
	require.EqualValues(t, 0, res.Code)

	// by unapproved guest2
	// guest2 := crypto.GenPrivKeyEd25519()
	// tx = newAddUsernameAddressNftTx(t, "anybody@example.com", []byte("blockchain4"))
	// res = signAndCommit(t, myApp, tx, []Signer{{guest2, 0}}, appFixture.ChainID, 6)
	// require.NotEqual(t, 0, res.Code)
}

func newIssueBlockchainNftTx(t require.TestingT, blockchainID []byte, issuer weave.Address) *app.Tx {
	// when blockchain nft issued
	return &app.Tx{
		Sum: &app.Tx_IssueBlockchainNftMsg{&blockchain.IssueTokenMsg{
			Id:      blockchainID,
			Owner:   issuer,
			Details: blockchain.TokenDetails{Iov: blockchain.IOV{Codec: "asd"}},
		}},
	}
}

func newIssueUsernameNftTx(t require.TestingT, ID string, owner weave.Address, blockchainID []byte, approved ...[]byte) *app.Tx {
	return &app.Tx{
		Sum: &app.Tx_IssueUsernameNftMsg{
			&username.IssueTokenMsg{
				Id:        []byte(ID),
				Owner:     owner,
				Approvals: approved,
				Addresses: []*username.ChainAddress{{blockchainID, []byte("myChainAddress")}},
			}},
	}
}

func newAddUsernameAddressNftTx(t require.TestingT, ID string, blockchainID []byte) *app.Tx {
	return &app.Tx{
		Sum: &app.Tx_AddUsernameAddressNftMsg{
			&username.AddChainAddressMsg{
				Id:        []byte(ID),
				Addresses: &username.ChainAddress{ChainID: blockchainID, Address: []byte("myChainAddressID")},
			}},
	}
}
