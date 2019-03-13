package scenarios

import (
	"testing"
	"time"

	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/x/validators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

func TestQueryValidatorUpdateSigner(t *testing.T) {
	// when
	r, err := bnsClient.AbciQuery("/validators", []byte("accounts"))
	// then
	require.NoError(t, err)
	require.Len(t, r.Models, 1)

	var accounts validators.Accounts
	require.NoError(t, accounts.Unmarshal(r.Models[0].Value))
	require.Len(t, accounts.Addresses, 1)
	assert.Contains(t, accounts.Addresses, []byte(multiSigContract.Address()), "multisig address not found")
}

func TestUpdateValidatorSet(t *testing.T) {
	current, err := client.Admin(bnsClient).GetValidators(client.CurrentHeight)
	require.NoError(t, err)

	newValidator := ed25519.GenPrivKey()
	keyEd25519 := newValidator.PubKey().(ed25519.PubKeyEd25519)
	aNonce := client.NewNonce(bnsClient, alice.PublicKey().Address())

	// when adding a new validator
	addValidatorTX := client.SetValidatorTx(
		&validators.ValidatorUpdate{
			Pubkey: validators.Pubkey{
				Type: "ed25519",
				Data: keyEd25519[:],
			},
			Power: 1,
		},
	)
	addValidatorTX = addValidatorTX.WithFee(alice.PublicKey().Address(), antiSpamFee)

	_, _, contractID, _ := multiSigContract.Parse()
	addValidatorTX.Multisig = [][]byte{contractID}

	seq, err := aNonce.Next()
	require.NoError(t, err)
	require.NoError(t, client.SignTx(addValidatorTX, alice, chainID, seq))
	resp := bnsClient.BroadcastTx(addValidatorTX)

	// then
	t.Logf("Adding validator: %X\n", keyEd25519)
	require.NoError(t, resp.IsError())

	// and tendermint validator set is updated
	tmValidatorSet := awaitValidatorUpdate(resp.Response.Height + 2)
	require.NotNil(t, tmValidatorSet)
	require.Len(t, tmValidatorSet.Validators, len(current.Validators)+1)
	require.True(t, contains(tmValidatorSet.Validators, newValidator.PubKey()))

	// and when delete validator
	delValidatorTX := client.SetValidatorTx(
		&validators.ValidatorUpdate{
			Pubkey: validators.Pubkey{
				Type: "ed25519",
				Data: keyEd25519[:],
			},
			Power: 0, // 0 for delete
		},
	)
	delValidatorTX = delValidatorTX.WithFee(alice.PublicKey().Address(), antiSpamFee)
	delValidatorTX.Multisig = [][]byte{contractID}

	// then
	seq, err = aNonce.Next()
	require.NoError(t, err)
	require.NoError(t, client.SignTx(delValidatorTX, alice, chainID, seq))
	resp = bnsClient.BroadcastTx(delValidatorTX)

	// then
	require.NoError(t, resp.IsError())
	t.Logf("Removed validator: %X\n", keyEd25519)

	// and tendermint validator set is updated
	tmValidatorSet = awaitValidatorUpdate(resp.Response.Height + 2)
	require.NotNil(t, tmValidatorSet)
	require.Len(t, tmValidatorSet.Validators, len(current.Validators))
	assert.False(t, contains(tmValidatorSet.Validators, newValidator.PubKey()))
}

func awaitValidatorUpdate(height int64) *ctypes.ResultValidators {
	admin := client.Admin(bnsClient)
	for i := 0; i < 15; i++ {
		v, err := admin.GetValidators(height)
		if err == nil {
			return v
		}
		time.Sleep(time.Duration(i) * 50 * time.Millisecond)
	}
	return nil
}

func contains(got []*types.Validator, exp crypto.PubKey) bool {
	for _, v := range got {
		if exp.Equals(v.PubKey.(ed25519.PubKeyEd25519)) {
			return true
		}
	}
	return false
}
