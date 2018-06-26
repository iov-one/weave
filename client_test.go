package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	rpctest "github.com/tendermint/tendermint/rpc/test"
)

func TestMainSetup(t *testing.T) {
	config := rpctest.GetConfig()
	assert.Equal(t, "SetInTestMain", config.Moniker)
	// assert.Equal(t, "/dev/null", config.GenesisFile())
}
