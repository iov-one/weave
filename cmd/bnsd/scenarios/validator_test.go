package scenarios

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/x/validators"
	"github.com/stretchr/testify/require"

	//"github.com/tendermint/tendermint/crypto/ed25519"
	"testing"
)

func TestSetValidator(t *testing.T) {
	newValidator := client.GenPrivateKey()
	//newValidator := ed25519.GenPrivKey()
	tx := client.SetValidatorTx(&validators.ValidatorUpdate{
		// see https://tendermint.com/docs/app-dev/abci-spec.html#data-messages
		Pubkey: validators.Pubkey{
			Type: "ed25519",
			Data: newValidator.PublicKey().GetEd25519(),
		},
		Power: 10,
	})

	// when
	aNonce := client.NewNonce(bnsClient, alice.PublicKey().Address())
	seq, err := aNonce.Next()
	require.NoError(t, err)
	require.NoError(t, client.SignTx(tx, alice, chainID, seq))
	resp := bnsClient.BroadcastTx(tx)
	t.Log(resp)
	// then
	require.NoError(t, resp.IsError())

	// and data is updated
	qResp, err := http.Get(fmt.Sprintf("%s/validators", rpcAddress))
	require.NoError(t, err)
	require.Equal(t, 200, qResp.StatusCode)
	defer qResp.Body.Close()
	body, _ := ioutil.ReadAll(qResp.Body)
	t.Log(string(body))
	// todo: check for pubkey returned to match "newValidator"
	// returned content is amino encoded: see "github.com/tendermint/tendermint/crypto/ed25519"
}
