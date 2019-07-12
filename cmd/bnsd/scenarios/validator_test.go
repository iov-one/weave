package scenarios

import (
	"bytes"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/scenarios/bnsdtest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/validators"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

func TestQueryValidatorUpdateSigner(t *testing.T) {
	env, cleanup := bnsdtest.StartBnsd(t)
	defer cleanup()

	r, err := client.NewClient(env.Client.TendermintClient()).AbciQuery("/validators", []byte("accounts"))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(r.Models))

	var accounts validators.Accounts
	assert.Nil(t, accounts.Unmarshal(r.Models[0].Value))
	assert.Equal(t, 1, len(accounts.Addresses))
	for _, v := range accounts.Addresses {
		if bytes.Equal(v, []byte(env.MultiSigContract.Address())) {
			return
		}
	}
	t.Fatal("multisig address not found")
}

func TestUpdateValidatorSet(t *testing.T) {
	env, cleanup := bnsdtest.StartBnsd(t)
	defer cleanup()

	current, err := client.Admin(client.NewClient(env.Client.TendermintClient())).GetValidators(client.CurrentHeight)
	assert.Nil(t, err)

	newValidator := ed25519.GenPrivKey()
	keyEd25519 := newValidator.PubKey().(ed25519.PubKeyEd25519)
	aNonce := client.NewNonce(env.Client, env.Alice.PublicKey().Address())

	// when adding a new validator
	addValidatorTX := client.SetValidatorTx(
		weave.ValidatorUpdate{
			PubKey: weave.PubKey{
				Type: "ed25519",
				Data: keyEd25519[:],
			},
			Power: 1,
		},
	)
	addValidatorTX.Fee(env.Alice.PublicKey().Address(), env.AntiSpamFee)

	_, _, contractID, err := env.MultiSigContract.Parse()
	if err != nil {
		t.Fatalf("cannot parse multisig contract: %s", err)
	}
	addValidatorTX.Multisig = [][]byte{contractID}

	seq, err := aNonce.Next()
	assert.Nil(t, err)
	assert.Nil(t, client.SignTx(addValidatorTX, env.Alice, env.ChainID, seq))
	resp := env.Client.BroadcastTx(addValidatorTX)

	// then
	t.Logf("Adding validator: %X\n", keyEd25519)
	assert.Nil(t, resp.IsError())

	// and tendermint validator set is updated
	tmValidatorSet := awaitValidatorUpdate(env, resp.Response.Height+2)
	assert.Equal(t, true, tmValidatorSet != nil)
	assert.Equal(t, len(current.Validators)+1, len(tmValidatorSet.Validators))
	assert.Equal(t, true, contains(tmValidatorSet.Validators, newValidator.PubKey()))

	// and when delete validator
	delValidatorTX := client.SetValidatorTx(
		weave.ValidatorUpdate{
			PubKey: weave.PubKey{
				Type: "ed25519",
				Data: keyEd25519[:],
			},
			Power: 0, // 0 for delete
		},
	)
	delValidatorTX.Fee(env.Alice.PublicKey().Address(), env.AntiSpamFee)
	delValidatorTX.Multisig = [][]byte{contractID}

	// then
	seq, err = aNonce.Next()
	assert.Nil(t, err)
	assert.Nil(t, client.SignTx(delValidatorTX, env.Alice, env.ChainID, seq))
	resp = env.Client.BroadcastTx(delValidatorTX)

	// then
	assert.Nil(t, resp.IsError())
	t.Logf("Removed validator: %X\n", keyEd25519)

	// and tendermint validator set is updated
	tmValidatorSet = awaitValidatorUpdate(env, resp.Response.Height+2)
	assert.Equal(t, true, tmValidatorSet != nil)
	assert.Equal(t, len(current.Validators), len(tmValidatorSet.Validators))
	assert.Equal(t, false, contains(tmValidatorSet.Validators, newValidator.PubKey()))
}

func awaitValidatorUpdate(env *bnsdtest.EnvConf, height int64) *ctypes.ResultValidators {
	admin := client.Admin(client.NewClient(env.Client.TendermintClient()))
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
