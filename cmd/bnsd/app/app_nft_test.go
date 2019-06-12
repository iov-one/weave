package app_test

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/app/testdata/fixtures"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/iov-one/weave/coin"
	"github.com/stretchr/testify/require"
)

func TestIssueNfts(t *testing.T) {
	appFixture := fixtures.NewApp()
	issuerAddr := appFixture.GenesisKeyAddress
	issuerPrivKey := appFixture.GenesisKey

	myApp := appFixture.Build()
	myBlockchainID := []byte("myblockchain")

	tx := &app.Tx{
		Sum: &app.Tx_IssueUsernameNftMsg{
			IssueUsernameNftMsg: &username.IssueTokenMsg{
				Metadata: &weave.Metadata{Schema: 1},
				ID:       []byte("anybody@example.com"),
				Owner:    issuerAddr,
				Details: username.TokenDetails{
					Addresses: []username.ChainAddress{
						{BlockchainID: myBlockchainID, Address: "myChainAddress"},
					},
				},
			},
		},
	}
	tx.Fee(issuerAddr, coin.NewCoin(5, 0, "FRNK"))
	res := signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, 0}}, appFixture.ChainID, 3)
	require.EqualValues(t, 0, res.Code)
}
