package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/rpc/client"
	rpctest "github.com/tendermint/tendermint/rpc/test"
)

// blocks go by fast, no need to wait seconds....
func fastWaiter(delta int64) (abort error) {
	delay := time.Duration(delta) * 5 * time.Millisecond
	time.Sleep(delay)
	return nil
}

var _ client.Waiter = fastWaiter

func TestMainSetup(t *testing.T) {
	config := rpctest.GetConfig()
	assert.Equal(t, "SetInTestMain", config.Moniker)

	conn := client.NewLocal(node)
	status, err := conn.Status()
	require.NoError(t, err)
	assert.Equal(t, "SetInTestMain", status.NodeInfo.Moniker)

	// wait for some blocks to be produced....
	client.WaitForHeight(conn, 5, fastWaiter)
	status, err = conn.Status()
	require.NoError(t, err)
	assert.True(t, status.SyncInfo.LatestBlockHeight > 4)

	// assert.Equal(t, "/dev/null", config.GenesisFile())
}
