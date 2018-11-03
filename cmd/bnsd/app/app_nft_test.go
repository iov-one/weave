package app_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x/approvals"
	"github.com/iov-one/weave/x/multisig"

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
	var height int64 = 2
	var nbBlockchains int64 = 10
	issuerNonce := nbBlockchains
	for i := int64(0); i < nbBlockchains; i++ {
		tx := newIssueBlockchainNftTx([]byte(fmt.Sprintf("blockchain%d", i)), issuerAddr)
		res := signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, i}}, appFixture.ChainID, height)
		require.EqualValues(t, 0, res.Code)
		height++
	}

	pk1 := crypto.GenPrivKeyEd25519()
	pk2 := crypto.GenPrivKeyEd25519()
	contractID := createContract(t, myApp, appFixture.ChainID, height,
		[]Signer{{issuerPrivKey, issuerNonce}}, 1, pk1.PublicKey().Address(), pk2.PublicKey().Address())
	height++
	issuerNonce++

	// check approvals
	// guest1 can update by signing
	// pk1 OR pk2 can update by multisig
	guest1 := crypto.GenPrivKeyEd25519()
	tx := newIssueUsernameNftTx("anybody@example.com", issuerAddr, []byte("blockchain1"),
		approvals.ApprovalCondition(guest1.PublicKey().Address(), "update"),
		approvals.ApprovalCondition(multisig.MultiSigCondition(contractID).Address(), "update"),
	)
	res := signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, issuerNonce}}, appFixture.ChainID, height)
	require.EqualValues(t, 0, res.Code)
	height++
	issuerNonce++

	// by owner sig
	tx = newAddUsernameAddressNftTx("anybody@example.com", []byte("blockchain2"))
	res = signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, issuerNonce}}, appFixture.ChainID, height)
	require.EqualValues(t, 0, res.Code)
	height++
	issuerNonce++

	// with guest1 sig
	tx = newAddUsernameAddressNftTx("anybody@example.com", []byte("blockchain3"))
	res = signAndCommit(t, myApp, tx, []Signer{{guest1, 0}}, appFixture.ChainID, height)
	require.EqualValues(t, 0, res.Code)
	height++

	// with multisig via pk1
	tx = withMultisig(newAddUsernameAddressNftTx("anybody@example.com", []byte("blockchain5")), contractID)
	res = signAndCommit(t, myApp, tx, []Signer{{pk1, 0}}, appFixture.ChainID, height)
	require.EqualValues(t, 0, res.Code)
	height++

	// with multisig via pk2
	tx = withMultisig(newAddUsernameAddressNftTx("anybody@example.com", []byte("blockchain6")), contractID)
	res = signAndCommit(t, myApp, tx, []Signer{{pk2, 0}}, appFixture.ChainID, height)
	require.EqualValues(t, 0, res.Code)
	height++
}

func newIssueBlockchainNftTx(blockchainID []byte, issuer weave.Address) *app.Tx {
	// when blockchain nft issued
	return &app.Tx{
		Sum: &app.Tx_IssueBlockchainNftMsg{&blockchain.IssueTokenMsg{
			Id:      blockchainID,
			Owner:   issuer,
			Details: blockchain.TokenDetails{Iov: blockchain.IOV{Codec: "asd"}},
		}},
	}
}

func newIssueUsernameNftTx(ID string, owner weave.Address, blockchainID []byte, approved ...[]byte) *app.Tx {
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

func newAddUsernameAddressNftTx(ID string, blockchainID []byte) *app.Tx {
	return &app.Tx{
		Sum: &app.Tx_AddUsernameAddressNftMsg{
			&username.AddChainAddressMsg{
				Id:        []byte(ID),
				Addresses: &username.ChainAddress{ChainID: blockchainID, Address: []byte("myChainAddressID")},
			}},
	}
}

func withMultisig(tx *app.Tx, contractID ...[]byte) *app.Tx {
	tx.Multisig = contractID
	return tx
}
