package app_test

import (
	"testing"

	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/app/testdata/fixtures"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/ticker"
	"github.com/iov-one/weave/x/nft/username"
	"github.com/stretchr/testify/require"
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
			Details: blockchain.TokenDetails{},
		},
		},
	}

	res := signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, 0}}, appFixture.ChainID, 2)

	// then
	require.EqualValues(t, 0, res.Code)

	// and when username nft issued
	tx = &app.Tx{
		Sum: &app.Tx_IssueUsernameNftMsg{&username.IssueTokenMsg{
			Id:      []byte("anybody@example.com"),
			Owner:   isuserAddr,
			Details: username.TokenDetails{[]username.ChainAddress{}},
		},
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
}
