package scenarios

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/scenarios/bnsdtest"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/iov-one/weave/coin"
	"github.com/stretchr/testify/require"
)

func TestIssueNfts(t *testing.T) {
	env, cleanup := bnsdtest.StartBnsd(t,
		bnsdtest.WithMsgFee("nft/username/issue", coin.NewCoin(5, 0, "IOV")),
	)
	defer cleanup()

	// Ensure suffix is never less than 100, but never more than 10000 (3 or 4 chars)
	// as there were test failures with eg. 10089
	// Min length is defined in x/nft helpers.
	uniqueSuffix := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(8999) + 100
	myBlockchainID := []byte(fmt.Sprintf("aliceChain%d", uniqueSuffix))
	myUserName := []byte(fmt.Sprintf("alice-%d@example.com", uniqueSuffix))
	nfts := []*app.Tx{
		{
			Sum: &app.Tx_IssueUsernameNftMsg{
				IssueUsernameNftMsg: &username.IssueTokenMsg{
					Metadata: &weave.Metadata{Schema: 1},
					ID:       myUserName,
					Owner:    env.Alice.PublicKey().Address(),
					Details: username.TokenDetails{Addresses: []username.ChainAddress{{
						BlockchainID: myBlockchainID,
						Address:      env.Alice.PublicKey().Address().String(),
					}},
					},
				}},
		},
	}
	aNonce := client.NewNonce(env.Client, env.Alice.PublicKey().Address())
	for i, tx := range nfts {
		t.Run(fmt.Sprintf("creating nft %d: %T", i, tx.Sum), func(t *testing.T) {
			// when
			seq, err := aNonce.Next()
			require.NoError(t, err)
			tx.Fee(env.Alice.PublicKey().Address(), coin.NewCoin(6, 0, "IOV"))
			require.NoError(t, client.SignTx(tx, env.Alice, env.ChainID, seq))
			resp := env.Client.BroadcastTx(tx)
			// then
			require.NoError(t, resp.IsError())
		})
	}
}
