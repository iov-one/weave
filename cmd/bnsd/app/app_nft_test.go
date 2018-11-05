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
	var height int64 = 2
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

	res := signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, 0}}, appFixture.ChainID, &height)

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

	res = signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, 1}}, appFixture.ChainID, &height)

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

	res = signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, 2}}, appFixture.ChainID, &height)

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
	for i := int64(0); i < nbBlockchains; i++ {
		tx := newIssueBlockchainNftTx([]byte(fmt.Sprintf("blockchain%d", i)), issuerAddr)
		res := signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, i}}, appFixture.ChainID, &height)
		require.EqualValues(t, 0, res.Code)
	}

	pk1 := crypto.GenPrivKeyEd25519()
	pk2 := crypto.GenPrivKeyEd25519()
	contractID := createContract(t, myApp, appFixture.ChainID, &height,
		[]Signer{{issuerPrivKey, nbBlockchains}}, 1, pk1.PublicKey().Address(), pk2.PublicKey().Address())

	// check approvals
	// guest1 can update by signing
	// admin can add approvals
	// guest2 can update once
	// pk1 OR pk2 can update by multisig
	guest1 := crypto.GenPrivKeyEd25519()
	guest2 := crypto.GenPrivKeyEd25519()
	admin := crypto.GenPrivKeyEd25519()
	tx := newIssueUsernameNftTx("anybody@example.com", issuerAddr, []byte("blockchain1"),
		approvals.ApprovalCondition(guest1.PublicKey().Address(), "update"),
		approvals.ApprovalCondition(admin.PublicKey().Address(), approvals.Admin),
		approvals.ApprovalConditionWithCount(guest2.PublicKey().Address(), "update", 1),
		approvals.ApprovalCondition(multisig.MultiSigCondition(contractID).Address(), "update"),
	)
	res := signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, nbBlockchains + 1}}, appFixture.ChainID, &height)
	require.EqualValues(t, 0, res.Code)

	// by owner sig
	tx = newAddUsernameAddressNftTx("anybody@example.com", []byte("blockchain2"))
	res = signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, nbBlockchains + 2}}, appFixture.ChainID, &height)
	require.EqualValues(t, 0, res.Code)

	// with guest1 sig
	tx = newAddUsernameAddressNftTx("anybody@example.com", []byte("blockchain3"))
	res = signAndCommit(t, myApp, tx, []Signer{{guest1, 0}}, appFixture.ChainID, &height)
	require.EqualValues(t, 0, res.Code)

	// with guest2 only once
	tx = newAddUsernameAddressNftTx("anybody@example.com", []byte("blockchain4"))
	res = signAndCommit(t, myApp, tx, []Signer{{guest2, 0}}, appFixture.ChainID, &height)
	require.EqualValues(t, 0, res.Code)
	cres := signAndCheck(t, myApp, tx, []Signer{{guest2, 1}}, appFixture.ChainID)
	require.NotEqual(t, uint32(0), cres.Code)

	// with guest3 not approved
	guest3 := crypto.GenPrivKeyEd25519()
	tx = newAddUsernameAddressNftTx("anybody@example.com", []byte("blockchain4"))
	cres = signAndCheck(t, myApp, tx, []Signer{{guest3, 0}}, appFixture.ChainID)
	require.NotEqual(t, uint32(0), cres.Code)

	// authorising guest3
	tx = newUsernameAddApprovalTx("anybody@example.com", approvals.ApprovalCondition(guest3.PublicKey().Address(), "update"))
	res = signAndCommit(t, myApp, tx, []Signer{{admin, 0}}, appFixture.ChainID, &height)
	require.EqualValues(t, 0, res.Code)

	// now guest3 is approved
	tx = newAddUsernameAddressNftTx("anybody@example.com", []byte("blockchain4"))
	cres = signAndCheck(t, myApp, tx, []Signer{{guest3, 0}}, appFixture.ChainID)
	require.EqualValues(t, 0, res.Code)

	// with multisig via pk1
	tx = withMultisig(newAddUsernameAddressNftTx("anybody@example.com", []byte("blockchain5")), contractID)
	res = signAndCommit(t, myApp, tx, []Signer{{pk1, 0}}, appFixture.ChainID, &height)
	require.EqualValues(t, 0, res.Code)

	// with multisig via pk2
	tx = withMultisig(newAddUsernameAddressNftTx("anybody@example.com", []byte("blockchain6")), contractID)
	res = signAndCommit(t, myApp, tx, []Signer{{pk2, 0}}, appFixture.ChainID, &height)
	require.EqualValues(t, 0, res.Code)
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

func newUsernameAddApprovalTx(ID string, perm []byte) *app.Tx {
	return &app.Tx{
		Sum: &app.Tx_UsernameAddApprovalMsg{
			&username.AddApprovalMsg{
				Id:       []byte(ID),
				Approval: perm,
			}},
	}
}

func withMultisig(tx *app.Tx, contractID ...[]byte) *app.Tx {
	tx.Multisig = contractID
	return tx
}
