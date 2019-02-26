package app_test

import (
	"testing"

	weave_app "github.com/iov-one/weave/app"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/app/testdata/fixtures"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestIssueNfts(t *testing.T) {
	appFixture := fixtures.NewApp()
	isuserAddr := appFixture.GenesisKeyAddress
	issuerPrivKey := appFixture.GenesisKey

	myApp := appFixture.Build()
	myBlockchainID := []byte("myblockchain")

	tx := &app.Tx{
		Sum: &app.Tx_IssueUsernameNftMsg{&username.IssueTokenMsg{
			ID:    []byte("anybody@example.com"),
			Owner: isuserAddr,
			Details: username.TokenDetails{
				Addresses: []username.ChainAddress{
					{BlockchainID: myBlockchainID, Address: "myChainAddress"},
				},
			},
		},
		},
	}

	res := signAndCommit(t, myApp, tx, []Signer{{issuerPrivKey, 0}}, appFixture.ChainID, 3)
	require.EqualValues(t, 0, res.Code)

	query := abci.RequestQuery{Path: "/nft/usernames/chainaddr", Data: []byte("myChainAddress;myblockchain")}
	qRes := myApp.Query(query)
	require.EqualValues(t, 0, qRes.Code, qRes.Log)
	var actual username.UsernameToken
	err := weave_app.UnmarshalOneResult(qRes.Value, &actual)
	require.NoError(t, err)
	require.Equal(t, []byte("anybody@example.com"), actual.GetBase().GetID())
}
