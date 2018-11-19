package scenarios

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/ticker"
	"github.com/iov-one/weave/x/nft/username"
	"github.com/stretchr/testify/require"
)

func TestIssueNfts(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	uniqueSuffix := rand.Intn(9999)
	myBlockchainID := []byte(fmt.Sprintf("aliceChain%d", uniqueSuffix))
	myTicker := []byte(fmt.Sprintf("%d", uniqueSuffix))
	myUserName := []byte(fmt.Sprintf("alice-%d@example.com", uniqueSuffix))
	nfts := []*app.Tx{
		{
			Sum: &app.Tx_IssueBlockchainNftMsg{&blockchain.IssueTokenMsg{
				Id:      myBlockchainID,
				Owner:   alice.PublicKey().Address(),
				Details: blockchain.TokenDetails{Iov: blockchain.IOV{Codec: "test", CodecConfig: `{ "any" : [ "json", "content" ] }`}},
			},
			},
		}, {
			Sum: &app.Tx_IssueTickerNftMsg{&ticker.IssueTokenMsg{
				Id:      myTicker,
				Owner:   alice.PublicKey().Address(),
				Details: ticker.TokenDetails{myBlockchainID},
			},
			},
		}, {
			Sum: &app.Tx_IssueUsernameNftMsg{&username.IssueTokenMsg{
				Id:    myUserName,
				Owner: alice.PublicKey().Address(),
				Details: username.TokenDetails{[]username.ChainAddress{{
					ChainID: myBlockchainID,
					Address: []byte(alice.PublicKey().Address()),
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
			//sig, err := sigs.SignTx(alice, tx, chainID, seq)
			//require.NoError(t, err)
			//tx.Signatures = append(tx.Signatures, sig)
			resp := bnsClient.BroadcastTx(tx)
			// then
			require.NoError(t, resp.IsError())
			delayForRateLimits()
		})
	}
}
