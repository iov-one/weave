package scenarios

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/blockchain"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/ticker"
	"github.com/iov-one/weave/cmd/bnsd/x/nft/username"
	"github.com/stretchr/testify/require"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func TestIssueNfts(t *testing.T) {
	// ID must be at least 3 characters, so ensure it's never less than 100.
	// Min length is defined in x/nft helpers.
	uniqueSuffix := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(9999) + 100
	myBlockchainID := []byte(fmt.Sprintf("aliceChain%d", uniqueSuffix))
	myTicker := []byte(fmt.Sprint(uniqueSuffix))
	myUserName := []byte(fmt.Sprintf("alice-%d@example.com", uniqueSuffix))
	nfts := []*app.Tx{
		{
			Sum: &app.Tx_IssueBlockchainNftMsg{&blockchain.IssueTokenMsg{
				ID:      myBlockchainID,
				Owner:   alice.PublicKey().Address(),
				Details: blockchain.TokenDetails{Iov: blockchain.IOV{Codec: "test", CodecConfig: `{ "any" : [ "json", "content" ] }`}},
			},
			},
		}, {
			Sum: &app.Tx_IssueTickerNftMsg{&ticker.IssueTokenMsg{
				ID:      myTicker,
				Owner:   alice.PublicKey().Address(),
				Details: ticker.TokenDetails{myBlockchainID},
			},
			},
		}, {
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

func TestOddTX(t *testing.T) {
	rawJson := `
 	{
    "owner": "4orppuuU/Ii3PrfL1rh7+T65vvA=",
    "id": "YWxpY2VDaGFpbjMzODM=",
    "details": {
     "chain": {},
     "iov": {
      "codec": "test",
      "codec_config": "{ \"any\" : [ \"json\", \"content\" ] }"
     }
    },
    "approvals": null
   }`

	var issueMsg blockchain.IssueTokenMsg
	require.NoError(t, json.Unmarshal([]byte(rawJson), &issueMsg))

	aNonce := client.NewNonce(bnsClient, alice.PublicKey().Address())
	// when
	seq, err := aNonce.Next()
	require.NoError(t, err)
	tx := &app.Tx{Sum: &app.Tx_IssueBlockchainNftMsg{&issueMsg}}
	require.NoError(t, client.SignTx(tx, alice, chainID, seq))
	resp := bnsClient.BroadcastTx(tx)
	// then
	require.NoError(t, resp.IsError())
	embeddedHeight := resp.Response.Height + 1
	var info *ctypes.ResultBlockchainInfo
	for {
		info, err = bnsClient.RawConnectionClient().BlockchainInfo(embeddedHeight, embeddedHeight)
		if err == nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	println(embeddedHeight)
	t.Logf("+++ hash: %s", info.BlockMetas[0].Header.AppHash.String())
}
