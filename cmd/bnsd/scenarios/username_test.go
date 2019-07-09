package scenarios

import (
	"testing"

	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/scenarios/bnsdtest"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestUsername(t *testing.T) {
	emilia := client.GenPrivateKey()

	env, cleanup := bnsdtest.StartBnsd(t,
		bnsdtest.WithUsername("emilia*iov", username.Token{
			Owner: emilia.PublicKey().Address(),
			Targets: []username.BlockchainAddress{
				{BlockchainID: "firstchain", Address: "01"},
				{BlockchainID: "secondchain", Address: "02"},
			},
		}),
	)
	defer cleanup()

	resp, err := env.Client.AbciQuery("/usernames", []byte("emilia*iov"))
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if len(resp.Models) == 0 {
		t.Fatal("username not found")
	}
	var token username.Token
	if err := token.Unmarshal(resp.Models[0].Value); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	assert.Equal(t, token.Owner, emilia.PublicKey().Address())
	assert.Equal(t, token.Targets[0].BlockchainID, "firstchain")
	assert.Equal(t, token.Targets[1].BlockchainID, "secondchain")
}
