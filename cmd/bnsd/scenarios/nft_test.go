package scenarios

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/stretchr/testify/require"
)

func TestIssueNfts(t *testing.T) {
	// ID must be at least 3 characters, so ensure it's never less than 100.
	// Min length is defined in x/nft helpers.
	uniqueSuffix := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(9999) + 100
	myBlockchainID := []byte(fmt.Sprintf("aliceChain%d", uniqueSuffix))
	myUserName := []byte(fmt.Sprintf("alice-%d@example.com", uniqueSuffix))
	nfts := []*app.Tx{
		{
			Sum: &app.Tx_IssueUsernameNftMsg{&username.IssueTokenMsg{
				ID:    myUserName,
				Owner: alice.PublicKey().Address(),
				Details: username.TokenDetails{[]username.ChainAddress{{
					BlockchainID: myBlockchainID,
					Address:      alice.PublicKey().Address().String(),
				},
				}},
			},
			},
		},
	}
	aNonce := client.NewNonce(bnsClient, alice.PublicKey().Address())
	for i, tx := range nfts {
		t.Run(fmt.Sprintf("creating nft %d: %T", i, tx.Sum), func(t *testing.T) {
			// when
			seq, err := aNonce.Next()
			require.NoError(t, err)
			require.NoError(t, client.SignTx(tx, alice, chainID, seq))
			resp := bnsClient.BroadcastTx(tx)
			// then
			require.NoError(t, resp.IsError())
			delayForRateLimits()
		})
	}
}
